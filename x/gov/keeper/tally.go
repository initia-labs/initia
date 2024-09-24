package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	customtypes "github.com/initia-labs/initia/x/gov/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

// TODO: Break into several smaller functions for clarity

// Tally iterates over the votes and updates the tally of a proposal based on the voting power of the
// voters
func (k Keeper) Tally(ctx context.Context, params customtypes.Params, proposal customtypes.Proposal) (quorumReached, passed bool, burnDeposits bool, tallyResults customtypes.TallyResult, err error) {
	weights, err := k.sk.GetVotingPowerWeights(ctx)
	if err != nil {
		return false, false, false, tallyResults, err
	}

	results := make(map[v1.VoteOption]math.LegacyDec)
	results[v1.OptionYes] = math.LegacyZeroDec()
	results[v1.OptionAbstain] = math.LegacyZeroDec()
	results[v1.OptionNo] = math.LegacyZeroDec()
	results[v1.OptionNoWithVeto] = math.LegacyZeroDec()

	totalVotingPower := math.LegacyZeroDec()
	totalStakingPower := math.ZeroInt()
	totalVestingPower := math.ZeroInt()
	currValidators := make(map[string]customtypes.ValidatorGovInfo)

	// fetch all the bonded validators, insert them into currValidators
	err = k.sk.IterateBondedValidatorsByPower(ctx, func(validator stakingtypes.ValidatorI) (stop bool, err error) {
		valAddr, err := k.sk.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
		if err != nil {
			return false, err
		}

		currValidators[validator.GetOperator()] = customtypes.NewValidatorGovInfo(
			valAddr,
			validator.GetBondedTokens(),
			validator.GetDelegatorShares(),
			sdk.NewDecCoins(),
			v1.WeightedVoteOptions{},
		)

		votingPower, _ := stakingtypes.CalculateVotingPower(validator.GetBondedTokens(), weights)
		totalStakingPower = totalStakingPower.Add(votingPower)

		return false, nil
	})
	if err != nil {
		return false, false, false, tallyResults, err
	}

	// fetch vesting table handle if vesting params are provided
	var vestingHandle *sdk.AccAddress
	if params.Vesting != nil {
		vesting := *params.Vesting
		moduleAddr, err := k.authKeeper.AddressCodec().StringToBytes(vesting.ModuleAddr)
		if err != nil {
			return false, false, false, tallyResults, err
		}

		creatorAddr, err := k.authKeeper.AddressCodec().StringToBytes(vesting.CreatorAddr)
		if err != nil {
			return false, false, false, tallyResults, err
		}

		// base denom validation will be done in vesting keeper
		vestingHandle, err = k.vestingKeeper.GetVestingHandle(ctx, moduleAddr, vesting.ModuleName, creatorAddr)
		if err != nil {
			return false, false, false, tallyResults, err
		}
	}

	rng := collections.NewPrefixedPairRange[uint64, sdk.AccAddress](proposal.Id)
	err = k.Votes.Walk(ctx, rng, func(key collections.Pair[uint64, sdk.AccAddress], vote v1.Vote) (bool, error) {
		// if validator, just record it in the map
		voter := sdk.MustAccAddressFromBech32(vote.Voter)

		valAddrStr := sdk.ValAddress(voter.Bytes()).String()
		if val, ok := currValidators[valAddrStr]; ok {
			val.Vote = vote.Options
			currValidators[valAddrStr] = val
		}

		// add vesting voting power if vesting params are provided
		if vestingHandle != nil {
			amount, err := k.vestingKeeper.GetUnclaimedVestedAmount(ctx, *vestingHandle, voter)
			if err != nil {
				return false, err
			}

			if !amount.IsZero() {
				// `the vesting token == the base denom`` check is done in vesting keeper,
				// so we can safely convert the amount to voting power
				votingPower := math.LegacyNewDecFromInt(amount)
				for _, option := range vote.Options {
					subPower := votingPower.Mul(math.LegacyMustNewDecFromStr(option.Weight))
					results[option.Option] = results[option.Option].Add(subPower)
				}
				totalVotingPower = totalVotingPower.Add(votingPower)
				totalVestingPower = totalVestingPower.Add(amount)
			}
		}

		// iterate over all delegations from voter, deduct from any delegated-to validators
		err = k.sk.IterateDelegations(ctx, voter, func(delegation stakingtypes.DelegationI) (stop bool, err error) {
			valAddrStr := delegation.GetValidatorAddr()

			if val, ok := currValidators[valAddrStr]; ok {
				// There is no need to handle the special case that validator address equal to voter address.
				// Because voter's voting power will tally again even if there will deduct voter's voting power from validator.
				val.DelegatorDeductions = val.DelegatorDeductions.Add(delegation.GetShares()...)
				currValidators[valAddrStr] = val

				// votingPower = delegation shares * bonded / total shares * denom weight
				votingPower := math.LegacyZeroDec()
				for _, share := range delegation.GetShares() {
					votingPower = votingPower.Add(
						share.Amount.
							MulInt(val.BondedTokens.AmountOf(share.Denom)).
							Quo(val.DelegatorShares.AmountOf(share.Denom)).
							Mul(weights.AmountOf(share.Denom)),
					)
				}

				for _, option := range vote.Options {
					subPower := votingPower.Mul(math.LegacyMustNewDecFromStr(option.Weight))
					results[option.Option] = results[option.Option].Add(subPower)
				}
				totalVotingPower = totalVotingPower.Add(votingPower)
			}

			return false, nil
		})
		if err != nil {
			return false, err
		}

		return false, k.Votes.Remove(ctx, collections.Join(vote.ProposalId, voter))
	})
	if err != nil {
		return false, false, false, tallyResults, err
	}

	// iterate over the validators again to tally their voting power
	for _, val := range currValidators {
		if len(val.Vote) == 0 {
			continue
		}

		sharesAfterDeductions := val.DelegatorShares.Sub(val.DelegatorDeductions)
		votingPower := math.LegacyZeroDec()
		for _, share := range sharesAfterDeductions {
			votingPower = votingPower.Add(
				share.Amount.
					MulInt(val.BondedTokens.AmountOf(share.Denom)).
					Quo(val.DelegatorShares.AmountOf(share.Denom)).
					Mul(weights.AmountOf(share.Denom)),
			)
		}

		for _, option := range val.Vote {
			subPower := votingPower.Mul(math.LegacyMustNewDecFromStr(option.Weight))
			results[option.Option] = results[option.Option].Add(subPower)
		}
		totalVotingPower = totalVotingPower.Add(votingPower)
	}

	v1TallyResult := v1.NewTallyResultFromMap(results)
	tallyResults = customtypes.TallyResult{
		V1TallyResult:     &v1TallyResult,
		TallyHeight:       uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()),
		TotalStakingPower: totalStakingPower.String(),
		TotalVestingPower: totalVestingPower.String(),
	}

	// If there is no (staked + vesting) coins, the proposal fails
	totalPower := totalVestingPower.Add(totalStakingPower)
	if totalPower.IsZero() {
		return false, false, false, tallyResults, nil
	}

	// If there is not enough quorum of votes, the proposal fails
	percentVoting := totalVotingPower.Quo(math.LegacyNewDecFromInt(totalPower))
	quorum, _ := math.LegacyNewDecFromStr(params.Quorum)
	if percentVoting.LT(quorum) {
		return false, false, params.BurnVoteQuorum, tallyResults, nil
	}

	// If no one votes (everyone abstains), proposal fails
	if totalVotingPower.Sub(results[v1.OptionAbstain]).Equal(math.LegacyZeroDec()) {
		return true, false, false, tallyResults, nil
	}

	// If more than 1/3 of voters veto, proposal fails
	vetoThreshold, _ := math.LegacyNewDecFromStr(params.VetoThreshold)
	if results[v1.OptionNoWithVeto].Quo(totalVotingPower).GT(vetoThreshold) {
		return true, false, params.BurnVoteVeto, tallyResults, nil
	}

	// If more than 1/2 of non-abstaining voters vote Yes, proposal passes
	// For expedited or emergency, 2/3
	var thresholdStr string
	if (proposal.Emergency || proposal.Expedited) && !IsLowThresholdProposal(params, proposal) {
		thresholdStr = params.GetExpeditedThreshold()
	} else {
		thresholdStr = params.GetThreshold()
	}

	threshold, _ := math.LegacyNewDecFromStr(thresholdStr)

	if results[v1.OptionYes].Quo(totalVotingPower.Sub(results[v1.OptionAbstain])).GT(threshold) {
		return true, true, false, tallyResults, nil
	}

	// If more than 1/2 of non-abstaining voters vote No, proposal fails
	return true, false, false, tallyResults, nil
}

// IsLowThresholdProposal checks if the proposal is a low threshold proposal
func IsLowThresholdProposal(params customtypes.Params, proposal customtypes.Proposal) bool {
	messages, err := proposal.GetMsgs()
	if err != nil {
		return false
	}

	for _, msg := range messages {

		var fid string
		switch msg := msg.(type) {
		case *movetypes.MsgExecute:
			fid = fmt.Sprintf("%s::%s::%s", msg.ModuleAddress, msg.ModuleName, msg.FunctionName)
		case *movetypes.MsgExecuteJSON:
			fid = fmt.Sprintf("%s::%s::%s", msg.ModuleAddress, msg.ModuleName, msg.FunctionName)
		default:
			return false
		}

		if !params.IsLowThresholdFunction(fid) {
			return false
		}
	}

	return true
}
