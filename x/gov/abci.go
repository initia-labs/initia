package gov

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/initia-labs/initia/x/gov/keeper"
)

// EndBlocker called every block, process inflation, update validator set.
func EndBlocker(ctx sdk.Context, k *keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	logger := k.Logger(ctx)
	params := k.GetParams(ctx)

	// delete inactive proposal from store and its deposits
	k.IterateInactiveProposalsQueue(ctx, ctx.BlockHeader().Time, func(proposal v1.Proposal) bool {
		k.DeleteProposal(ctx, proposal.Id)

		if !params.BurnProposalDepositPrevote {
			k.RefundAndDeleteDeposits(ctx, proposal.Id) // refund deposit if proposal got removed without getting 100% of the proposal
		} else {
			k.DeleteAndBurnDeposits(ctx, proposal.Id) // burn the deposit if proposal got removed without getting 100% of the proposal
		}

		// called when proposal become inactive
		k.Hooks().AfterProposalFailedMinDeposit(ctx, proposal.Id)

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
			"title", proposal.GetTitle(),
			"min_deposit", sdk.NewCoins(params.MinDeposit...).String(),
			"total_deposit", sdk.NewCoins(proposal.TotalDeposit...).String(),
		)

		return false
	})

	// fetch active proposals whose voting periods have ended (are passed the block time)
	k.IterateActiveProposalsQueue(ctx, ctx.BlockHeader().Time, func(proposal v1.Proposal) bool {

		_, passed, burnDeposits, tallyResults := k.Tally(ctx, proposal)
		handleTallyResult(ctx, k, proposal, passed, burnDeposits, tallyResults)

		return false
	})

	// periodically tally emergency proposal
	if ctx.BlockTime().After(k.GetLastEmergencyProposalTallyTimestamp(ctx).Add(params.EmergencyTallyInterval)) {
		k.IterateEmergencyProposals(ctx, func(proposal v1.Proposal) (stop bool) {
			cacheCtx, writeCache := ctx.CacheContext()

			// Tally internally delete votes, so use cache context to prevent
			// deleting votes of proposal in progress.
			quorumReached, passed, burnDeposits, tallyResults := k.Tally(cacheCtx, proposal)
			if !quorumReached {
				return false
			}

			// quorum reached; commit the state changes from k.Tally()
			writeCache()

			// handle tally result
			handleTallyResult(ctx, k, proposal, passed, burnDeposits, tallyResults)

			return false
		})

		k.RecordLastEmergencyProposalTallyTimestamp(ctx)
	}

}

func handleTallyResult(
	ctx sdk.Context,
	k *keeper.Keeper,
	proposal v1.Proposal,
	passed, burnDeposits bool,
	tallyResults v1.TallyResult,
) {
	if burnDeposits {
		k.DeleteAndBurnDeposits(ctx, proposal.Id)
	} else {
		k.RefundAndDeleteDeposits(ctx, proposal.Id)
	}

	var tagValue, logMsg string

	if passed {
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
		if err == nil {
			for idx, msg = range messages {
				handler := k.Router().Handler(msg)

				var res *sdk.Result
				res, err = handler(cacheCtx, msg)
				if err != nil {
					break
				}

				events = append(events, res.GetEvents()...)
			}
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
			tagValue = types.AttributeValueProposalFailed
			logMsg = fmt.Sprintf("passed, but msg %d (%s) failed on execution: %s", idx, sdk.MsgTypeURL(msg), err)
		}
	} else {
		proposal.Status = v1.StatusRejected
		tagValue = types.AttributeValueProposalRejected
		logMsg = "rejected"
	}

	proposal.FinalTallyResult = &tallyResults

	k.SetProposal(ctx, proposal)
	k.RemoveFromActiveProposalQueue(ctx, proposal.Id, *proposal.VotingEndTime)
	k.RemoveFromEmergencyProposalQueue(ctx, proposal.Id)

	// When proposal removed from the active queue
	k.Hooks().AfterProposalVotingPeriodEnded(ctx, proposal.Id)

	k.Logger(ctx).Info(
		"proposal tallied",
		"proposal", proposal.Id,
		"result", logMsg,
	)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeActiveProposal,
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposal.Id)),
			sdk.NewAttribute(types.AttributeKeyProposalResult, tagValue),
		),
	)

	return
}
