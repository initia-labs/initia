package gov

import (
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/initia-labs/initia/x/gov/keeper"

	customtypes "github.com/initia-labs/initia/x/gov/types"
)

// EndBlocker called every block, process inflation, update validator set.
func EndBlocker(ctx sdk.Context, k *keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	logger := k.Logger(ctx)
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	// delete dead proposals from store and returns theirs deposits.
	// A proposal is dead when it's inactive and didn't get enough deposit on time to get into voting phase.
	rng := collections.NewPrefixUntilPairRange[time.Time, uint64](ctx.BlockTime())
	err = k.InactiveProposalsQueue.Walk(ctx, rng, func(key collections.Pair[time.Time, uint64], _ uint64) (bool, error) {
		proposal, err := k.Proposals.Get(ctx, key.K2())
		if err != nil {
			// if the proposal has an encoding error, this means it cannot be processed by x/gov
			// this could be due to some types missing their registration
			// instead of returning an error (i.e, halting the chain), we fail the proposal
			if errors.Is(err, collections.ErrEncoding) {
				proposal.Id = key.K2()
				if err := failUnsupportedProposal(ctx, k, proposal, err.Error(), false); err != nil {
					return false, err
				}

				if err = k.DeleteProposal(ctx, proposal.Id); err != nil {
					return false, err
				}

				return false, nil
			}

			return false, err
		}

		if err = k.DeleteProposal(ctx, proposal.Id); err != nil {
			return false, err
		}

		if !params.BurnProposalDepositPrevote {
			err = k.RefundAndDeleteDeposits(ctx, proposal.Id) // refund deposit if proposal got removed without getting 100% of the proposal
		} else {
			err = k.DeleteAndBurnDeposits(ctx, proposal.Id) // burn the deposit if proposal got removed without getting 100% of the proposal
		}

		if err != nil {
			return false, err
		}

		// called when proposal become inactive
		cacheCtx, writeCache := ctx.CacheContext()
		err = k.Hooks().AfterProposalFailedMinDeposit(cacheCtx, proposal.Id)
		if err == nil { // purposely ignoring the error here not to halt the chain if the hook fails
			writeCache()
		} else {
			logger.Error("failed to execute AfterProposalFailedMinDeposit hook", "error", err)
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeInactiveProposal,
				sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
				sdk.NewAttribute(types.AttributeKeyProposalResult, types.AttributeValueProposalDropped),
			),
		)

		logger.Info(
			"proposal did not meet minimum deposit; deleted",
			"proposal", proposal.Id,
			"expedited", proposal.Expedited,
			"title", proposal.Title,
			"min_deposit", sdk.NewCoins(proposal.GetMinDepositFromParams(params)...).String(),
			"total_deposit", sdk.NewCoins(proposal.TotalDeposit...).String(),
		)

		return false, nil
	})
	if err != nil {
		return err
	}

	rng = collections.NewPrefixUntilPairRange[time.Time, uint64](ctx.BlockTime())
	err = k.ActiveProposalsQueue.Walk(ctx, rng, func(key collections.Pair[time.Time, uint64], _ uint64) (bool, error) {
		proposal, err := k.Proposals.Get(ctx, key.K2())
		if err != nil {
			// if the proposal has an encoding error, this means it cannot be processed by x/gov
			// this could be due to some types missing their registration
			// instead of returning an error (i.e, halting the chain), we fail the proposal
			if errors.Is(err, collections.ErrEncoding) {
				proposal.Id = key.K2()
				if err := failUnsupportedProposal(ctx, k, proposal, err.Error(), true); err != nil {
					return false, err
				}

				if err = k.ActiveProposalsQueue.Remove(ctx, collections.Join(*proposal.VotingEndTime, proposal.Id)); err != nil {
					return false, err
				}

				if proposal.Emergency {
					if err = k.EmergencyProposalsQueue.Remove(ctx, collections.Join(*proposal.EmergencyNextTallyTime, proposal.Id)); err != nil {
						return false, err
					}
					if err = k.EmergencyProposals.Remove(ctx, proposal.Id); err != nil {
						return false, err
					}
				}

				return false, nil
			}

			return false, err
		}

		// emergency proposals are handled separately
		if proposal.Emergency {
			return false, nil
		}

		_, passed, burnDeposits, tallyResults, err := k.Tally(ctx, params, proposal)
		if err != nil {
			return false, err
		}

		err = handleTallyResult(ctx, k, proposal, passed, burnDeposits, tallyResults)
		if err != nil {
			return false, err
		}

		return false, nil
	})
	if err != nil {
		return err
	}

	err = k.EmergencyProposalsQueue.Walk(ctx, rng, func(key collections.Pair[time.Time, uint64], _ uint64) (bool, error) {
		proposal, err := k.Proposals.Get(ctx, key.K2())
		if err != nil {
			// if the proposal has an encoding error, this means it cannot be processed by x/gov
			// this could be due to some types missing their registration
			// instead of returning an error (i.e, halting the chain), we fail the proposal
			if errors.Is(err, collections.ErrEncoding) {
				proposal.Id = key.K2()
				if err := failUnsupportedProposal(ctx, k, proposal, err.Error(), true); err != nil {
					return false, err
				}

				if err = k.ActiveProposalsQueue.Remove(ctx, collections.Join(*proposal.VotingEndTime, proposal.Id)); err != nil {
					return false, err
				}

				if err = k.EmergencyProposalsQueue.Remove(ctx, key); err != nil {
					return false, err
				}

				if err = k.EmergencyProposals.Remove(ctx, proposal.Id); err != nil {
					return false, err
				}
				return false, nil
			}
			return false, err
		}

		cacheCtx, writeCache := ctx.CacheContext()
		quorumReached, passed, burnDeposits, tallyResults, err := k.Tally(cacheCtx, params, proposal)
		if err != nil {
			return false, err
		}

		// schedule the next tally only if quorum is not reached and voting period is not over
		if !quorumReached && proposal.VotingEndTime.After(ctx.BlockTime()) {
			nextTallyTime := ctx.BlockTime().Add(params.EmergencyTallyInterval)

			// if the next tally time is after the voting end time, set it to the voting end time
			if nextTallyTime.After(*proposal.VotingEndTime) {
				nextTallyTime = *proposal.VotingEndTime
			}

			if err = k.EmergencyProposalsQueue.Set(ctx, collections.Join(nextTallyTime, proposal.Id), proposal.Id); err != nil {
				return false, err
			}
			proposal.EmergencyNextTallyTime = &nextTallyTime
			if err = k.EmergencyProposalsQueue.Remove(ctx, key); err != nil {
				return false, err
			}
			err = k.Proposals.Set(ctx, proposal.Id, proposal)
			if err != nil {
				return false, err
			}
			return false, nil
		}

		// quorum reached; commit the state changes from k.Tally()
		writeCache()

		err = handleTallyResult(ctx, k, proposal, passed, burnDeposits, tallyResults)
		if err != nil {
			return false, err
		}

		return false, nil
	})

	return err
}

func handleTallyResult(
	ctx sdk.Context,
	k *keeper.Keeper,
	proposal customtypes.Proposal,
	passed, burnDeposits bool,
	tallyResults customtypes.TallyResult,
) (err error) {
	// If an expedited proposal fails, we do not want to update
	// the deposit at this point since the proposal is converted to regular.
	// As a result, the deposits are either deleted or refunded in all cases
	// EXCEPT when an expedited proposal fails.
	if !proposal.Expedited || passed {
		if burnDeposits {
			err = k.DeleteAndBurnDeposits(ctx, proposal.Id)
		} else {
			err = k.RefundAndDeleteDeposits(ctx, proposal.Id)
		}
		if err != nil {
			return err
		}
	}

	if err = k.ActiveProposalsQueue.Remove(ctx, collections.Join(*proposal.VotingEndTime, proposal.Id)); err != nil {
		return err
	}

	if proposal.Emergency {
		if err = k.EmergencyProposalsQueue.Remove(ctx, collections.Join(*proposal.EmergencyNextTallyTime, proposal.Id)); err != nil {
			return err
		}

		if err = k.EmergencyProposals.Remove(ctx, proposal.Id); err != nil {
			return err
		}
	}

	var tagValue, logMsg string

	switch {
	case passed:
		var (
			idx    int
			events sdk.Events
			msg    sdk.Msg
		)

		// attempt to execute all messages within the passed proposal
		// Messages may mutate state thus we use a cached context. If one of
		// the handlers fails, no state mutation is written and the error
		// message is logged.
		cacheCtx, writeCache := ctx.CacheContext()
		messages, err := proposal.GetMsgs()
		if err != nil {
			proposal.Status = v1.StatusFailed
			proposal.FailedReason = err.Error()
			tagValue = types.AttributeValueProposalFailed
			logMsg = fmt.Sprintf("passed proposal (%v) failed to execute; msgs: %s", proposal, err)

			break
		}

		// execute all messages
		for idx, msg = range messages {
			handler := k.Router().Handler(msg)
			var res *sdk.Result
			res, err = safeExecuteHandler(cacheCtx, msg, handler)
			if err != nil {
				break
			}

			events = append(events, res.GetEvents()...)
		}

		// `err == nil` when all handlers passed.
		// Or else, `idx` and `err` are populated with the msg index and error.
		if err == nil {
			proposal.Status = v1.StatusPassed
			tagValue = types.AttributeValueProposalPassed
			logMsg = "passed"

			// write state to the underlying multi-store
			writeCache()

			// propagate the msg events to the current context
			ctx.EventManager().EmitEvents(events)
		} else {
			proposal.Status = v1.StatusFailed
			proposal.FailedReason = err.Error()
			tagValue = types.AttributeValueProposalFailed
			logMsg = fmt.Sprintf("passed, but msg %d (%s) failed on execution: %s", idx, sdk.MsgTypeURL(msg), err)
		}
	case proposal.Expedited:
		// When expedited proposal fails, it is converted
		// to a regular proposal. As a result, the voting period is extended, and,
		// once the regular voting period expires again, the tally is repeated
		// according to the regular proposal rules.
		proposal.Expedited = false
		params, err := k.Params.Get(ctx)
		if err != nil {
			return err
		}
		endTime := proposal.VotingStartTime.Add(params.VotingPeriod)
		proposal.VotingEndTime = &endTime

		err = k.ActiveProposalsQueue.Set(ctx, collections.Join(*proposal.VotingEndTime, proposal.Id), proposal.Id)
		if err != nil {
			return err
		}

		tagValue = types.AttributeValueExpeditedProposalRejected
		logMsg = "expedited proposal converted to regular"
	default:
		proposal.Status = v1.StatusRejected
		proposal.FailedReason = "proposal did not get enough votes to pass"
		tagValue = types.AttributeValueProposalRejected
		logMsg = "rejected"
	}

	proposal.FinalTallyResult = tallyResults

	err = k.SetProposal(ctx, proposal)
	if err != nil {
		return err
	}

	// when proposal become active
	cacheCtx, writeCache := ctx.CacheContext()
	err = k.Hooks().AfterProposalVotingPeriodEnded(cacheCtx, proposal.Id)
	if err == nil { // purposely ignoring the error here not to halt the chain if the hook fails
		writeCache()
	} else {
		k.Logger(ctx).Error("failed to execute AfterProposalVotingPeriodEnded hook", "error", err)
	}

	k.Logger(ctx).Info(
		"proposal tallied",
		"proposal", proposal.Id,
		"status", proposal.Status.String(),
		"expedited", proposal.Expedited,
		"title", proposal.Title,
		"results", logMsg,
	)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeActiveProposal,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
			sdk.NewAttribute(types.AttributeKeyProposalResult, tagValue),
			sdk.NewAttribute(types.AttributeKeyProposalLog, logMsg),
		),
	)

	return
}

// executes handle(msg) and recovers from panic.
func safeExecuteHandler(ctx sdk.Context, msg sdk.Msg, handler baseapp.MsgServiceHandler,
) (res *sdk.Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handling x/gov proposal msg [%s] PANICKED: %v", msg, r)
		}
	}()
	res, err = handler(ctx, msg)
	return
}

// failUnsupportedProposal fails a proposal that cannot be processed by gov
func failUnsupportedProposal(
	ctx sdk.Context,
	k *keeper.Keeper,
	proposal customtypes.Proposal,
	errMsg string,
	active bool,
) error {
	proposal.Status = v1.StatusFailed
	proposal.FailedReason = fmt.Sprintf("proposal failed because it cannot be processed by gov: %s", errMsg)
	proposal.Messages = nil // clear out the messages

	if err := k.SetProposal(ctx, proposal); err != nil {
		return err
	}

	if err := k.RefundAndDeleteDeposits(ctx, proposal.Id); err != nil {
		return err
	}

	eventType := types.EventTypeInactiveProposal
	if active {
		eventType = types.EventTypeActiveProposal
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
			sdk.NewAttribute(types.AttributeKeyProposalResult, types.AttributeValueProposalFailed),
		),
	)

	k.Logger(ctx).Info(
		"proposal failed to decode; deleted",
		"proposal", proposal.Id,
		"expedited", proposal.Expedited,
		"title", proposal.Title,
		"results", errMsg,
	)

	return nil
}
