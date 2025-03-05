package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	types "github.com/initia-labs/initia/v1/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Slash a validator for an infraction committed at a known height
// Find the contributing stake at that height and burn the specified slashFactor
// of it, updating unbonding delegations & redelegations appropriately
//
// CONTRACT:
//
//	slashFactor is non-negative
//
// CONTRACT:
//
//	Infraction was committed equal to or less than an unbonding period in the past,
//	so all unbonding delegations and redelegations from that height are stored
//
// CONTRACT:
//
//	Slash will not slash unbonded validators (for the above reason)
//
// CONTRACT:
//
//	Infraction was committed at the current height or at a past height,
//	not at a height in the future
func (k Keeper) Slash(ctx context.Context, consAddr sdk.ConsAddress, infractionHeight int64, slashFactor math.LegacyDec) (sdk.Coins, error) {
	logger := k.Logger(ctx)

	if slashFactor.IsNegative() {
		panic(fmt.Errorf("attempted to slash with a negative slash factor: %v", slashFactor))
	}

	validator, err := k.GetValidatorByConsAddr(ctx, consAddr)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		// If not found, the validator must have been overslashed and removed - so we don't need to do anything
		// NOTE:  Correctness dependent on invariant that unbonding delegations / redelegations must also have been completely
		//        slashed in this case - which we don't explicitly check, but should be true.
		// Log the slash attempt for future reference (maybe we should tag it too)
		conStr, err := k.consensusAddressCodec.BytesToString(consAddr)
		if err != nil {
			return nil, err
		}

		logger.Error(
			"WARNING: ignored attempt to slash a nonexistent validator; we recommend you investigate immediately",
			"validator", conStr,
		)

		return sdk.NewCoins(), nil
	} else if err != nil {
		return nil, err
	}

	// should not be slashing an unbonded validator
	if validator.IsUnbonded() {
		return nil, fmt.Errorf("should not be slashing unbonded validator: %s", validator.GetOperator())
	}

	operatorAddress, err := k.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
	if err != nil {
		return nil, err
	}

	// call the before-modification hook
	if err := k.Hooks().BeforeValidatorModified(ctx, operatorAddress); err != nil {
		k.Logger(ctx).Error("failed to call before validator modified hook", "error", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	switch {
	case infractionHeight > sdkCtx.BlockHeight():
		// Can't slash infractions in the future
		panic(fmt.Sprintf(
			"impossible attempt to slash future infraction at height %d but we are at height %d",
			infractionHeight, sdkCtx.BlockHeight()))

	case infractionHeight == sdkCtx.BlockHeight():
		// Special-case slash at current height for efficiency - we don't need to
		// look through unbonding delegations or redelegations.
		logger.Info(
			"slashing at current height; not scanning unbonding delegations & redelegations",
			"height", infractionHeight,
		)

	case infractionHeight < sdkCtx.BlockHeight():
		// Iterate through unbonding delegations from slashed validator
		unbondingDelegations, err := k.GetUnbondingDelegationsFromValidator(ctx, operatorAddress)
		if err != nil {
			return nil, err
		}

		for _, unbondingDelegation := range unbondingDelegations {
			if _, err := k.SlashUnbondingDelegation(ctx, unbondingDelegation, infractionHeight, slashFactor); err != nil {
				return nil, err
			}
		}

		// Iterate through redelegations from slashed source validator
		redelegations, err := k.GetRedelegationsFromSrcValidator(ctx, operatorAddress)
		if err != nil {
			return nil, err
		}

		for _, redelegation := range redelegations {
			if _, err := k.SlashRedelegation(ctx, validator, redelegation, infractionHeight, slashFactor); err != nil {
				return nil, err
			}
		}
	}

	// call slashing hooks to slash unbonding delegations.
	// here we don't care unbonding start height for the hooks.
	if err := k.SlashingHooks().SlashUnbondingDelegations(ctx, operatorAddress, slashFactor); err != nil {
		logger.Error("failed to call slash unbonding delegations hook", "error", err)
		return nil, err
	}

	tokensToBurn, _ := sdk.NewDecCoinsFromCoins(validator.Tokens...).MulDec(slashFactor).TruncateDecimal()

	// we need to calculate the *effective* slash fraction for distribution
	if !validator.Tokens.IsZero() {
		effectiveFractions := sdk.NewDecCoins()
		for _, coin := range tokensToBurn {
			effectiveFractions = append(effectiveFractions, sdk.NewDecCoinFromDec(
				coin.Denom,
				math.LegacyNewDecFromInt(coin.Amount).
					QuoRoundUp(math.LegacyNewDecFromInt(validator.Tokens.AmountOf(coin.Denom))),
			))
		}

		// call the before-slashed hook
		if err := k.Hooks().BeforeValidatorSlashed(ctx, operatorAddress, effectiveFractions); err != nil {
			logger.Error("failed to call before validator slashed hook", "error", err)
			return nil, err
		}
	}

	// Deduct from validator's bonded tokens and update the validator.
	// Burn the slashed tokens from the pool account and decrease the total supply.
	validator, err = k.RemoveValidatorTokens(ctx, validator, tokensToBurn)
	if err != nil {
		return nil, err
	}

	switch validator.GetStatus() {
	case types.Bonded:
		if err := k.burnBondedTokens(ctx, tokensToBurn); err != nil {
			panic(err)
		}
	case types.Unbonding, types.Unbonded:
		if err := k.burnNotBondedTokens(ctx, tokensToBurn); err != nil {
			panic(err)
		}
	default:
		panic("invalid validator status")
	}

	logger.Info(
		"validator slashed by slash factor",
		"validator", validator.GetOperator(),
		"slash_factor", slashFactor.String(),
		"burned", tokensToBurn,
	)

	return tokensToBurn, nil
}

// SlashWithInfractionReason implementation doesn't require the infraction (types.Infraction) to work but is required by Interchain Security.
func (k Keeper) SlashWithInfractionReason(
	ctx context.Context,
	consAddr sdk.ConsAddress,
	infractionHeight int64,
	slashFactor math.LegacyDec,
	_ types.Infraction,
) (sdk.Coins, error) {
	return k.Slash(ctx, consAddr, infractionHeight, slashFactor)
}

// jail a validator
func (k Keeper) Jail(ctx context.Context, consAddr sdk.ConsAddress) error {
	validator := k.mustGetValidatorByConsAddr(ctx, consAddr)
	if err := k.jailValidator(ctx, validator); err != nil {
		return err
	}

	logger := k.Logger(ctx)
	logger.Info("validator jailed", "validator", consAddr)
	return nil
}

// unjail a validator
func (k Keeper) Unjail(ctx context.Context, consAddr sdk.ConsAddress) error {
	validator := k.mustGetValidatorByConsAddr(ctx, consAddr)
	if err := k.unjailValidator(ctx, validator); err != nil {
		return err
	}

	logger := k.Logger(ctx)
	logger.Info("validator un-jailed", "validator", consAddr)
	return nil
}

// slash an unbonding delegation and update the pool
// return the amount that would have been slashed assuming
// the unbonding delegation had enough stake to slash
// (the amount actually slashed may be less if there's
// insufficient stake remaining)
func (k Keeper) SlashUnbondingDelegation(
	ctx context.Context,
	unbondingDelegation types.UnbondingDelegation,
	infractionHeight int64,
	slashFactor math.LegacyDec,
) (totalSlashAmount sdk.Coins, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockHeader().Time
	totalSlashAmount = sdk.NewCoins()
	burnedAmount := sdk.NewCoins()

	// perform slashing on all entries within the unbonding delegation
	for i, entry := range unbondingDelegation.Entries {
		// If unbonding started before this height, stake didn't contribute to infraction
		if entry.CreationHeight < infractionHeight {
			continue
		}

		if entry.IsMature(now) && !entry.OnHold() {
			// Unbonding delegation no longer eligible for slashing, skip it
			continue
		}

		// Calculate slash amount proportional to stake contributing to infraction
		slashAmountDec := sdk.NewDecCoinsFromCoins(entry.InitialBalance...).MulDec(slashFactor)
		slashAmount, _ := slashAmountDec.TruncateDecimal()
		totalSlashAmount = totalSlashAmount.Add(slashAmount...)

		// Don't slash more tokens than held
		// Possible since the unbonding delegation may already
		// have been slashed, and slash amounts are calculated
		// according to stake held at time of infraction
		unbondingSlashAmount := sdk.NewCoins()
		for _, coin := range slashAmount {
			amount := math.MinInt(coin.Amount, entry.Balance.AmountOf(coin.Denom))
			if amount.IsPositive() {
				unbondingSlashAmount = append(unbondingSlashAmount, sdk.NewCoin(coin.Denom, amount))
			}
		}

		// Update unbonding delegation if necessary
		if unbondingSlashAmount.IsZero() {
			continue
		}

		burnedAmount = burnedAmount.Add(unbondingSlashAmount...)
		entry.Balance = entry.Balance.Sub(unbondingSlashAmount...)
		unbondingDelegation.Entries[i] = entry
		if err := k.SetUnbondingDelegation(ctx, unbondingDelegation); err != nil {
			return nil, err
		}
	}

	if err := k.burnNotBondedTokens(ctx, burnedAmount); err != nil {
		return nil, err
	}

	return totalSlashAmount, nil
}

// slash a redelegation and update the pool
// return the amount that would have been slashed assuming
// the unbonding delegation had enough stake to slash
// (the amount actually slashed may be less if there's
// insufficient stake remaining)
// NOTE this is only slashing for prior infractions from the source validator
func (k Keeper) SlashRedelegation(
	ctx context.Context,
	srcValidator types.Validator,
	redelegation types.Redelegation,
	infractionHeight int64,
	slashFactor math.LegacyDec,
) (sdk.Coins, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockHeader().Time
	totalSlashAmount := sdk.NewCoins()
	bondedBurnedAmount, notBondedBurnedAmount := sdk.NewCoins(), sdk.NewCoins()

	valDstAddr, err := k.validatorAddressCodec.StringToBytes(redelegation.ValidatorDstAddress)
	if err != nil {
		return nil, fmt.Errorf("SlashRedelegation: could not parse validator destination address: %w", err)
	}

	delAddr, err := k.authKeeper.AddressCodec().StringToBytes(redelegation.DelegatorAddress)
	if err != nil {
		return nil, fmt.Errorf("SlashRedelegation: could not parse delegator address: %w", err)
	}

	// perform slashing on all entries within the redelegation
	for _, entry := range redelegation.Entries {
		// If redelegation started before this height, stake didn't contribute to infraction
		if entry.CreationHeight < infractionHeight {
			continue
		}

		if entry.IsMature(now) && !entry.OnHold() {
			// Redelegation no longer eligible for slashing, skip it
			continue
		}

		// Calculate slash amount proportional to stake contributing to infraction
		slashAmountDec := sdk.NewDecCoinsFromCoins(entry.InitialBalance...).MulDec(slashFactor)
		slashAmount, _ := slashAmountDec.TruncateDecimal()
		totalSlashAmount = totalSlashAmount.Add(slashAmount...)

		// Handle undelegation after redelegation
		// Prioritize slashing unbondingDelegation than delegation
		unbondingDelegation, err := k.UnbondingDelegations.Get(ctx, collections.Join(delAddr, valDstAddr))
		if err == nil {
			for i, entry := range unbondingDelegation.Entries {
				// slash with the amount of `slashAmount` if possible, else slash all unbonding token
				unbondingSlashAmount := slashAmount.Min(entry.Balance)

				switch {
				// There's no token to slash
				case unbondingSlashAmount.IsZero():
					continue
				// If unbonding started before this height, stake didn't contribute to infraction
				case entry.CreationHeight < infractionHeight:
					continue
				// Unbonding delegation no longer eligible for slashing, skip it
				case entry.IsMature(now) && !entry.OnHold():
					continue
				// Slash the unbonding delegation
				default:
					// update remaining slashAmount
					slashAmount = slashAmount.Sub(unbondingSlashAmount...)

					notBondedBurnedAmount = notBondedBurnedAmount.Add(unbondingSlashAmount...)
					entry.Balance = entry.Balance.Sub(unbondingSlashAmount...)
					unbondingDelegation.Entries[i] = entry
					if err = k.SetUnbondingDelegation(ctx, unbondingDelegation); err != nil {
						return nil, err
					}
				}
			}
		}

		// Slash the moved delegation
		// Unbond from target validator
		if slashAmount.IsZero() {
			continue
		}

		// Unbond from target validator
		sharesToUnbond := entry.SharesDst.MulDec(slashFactor)
		if sharesToUnbond.IsZero() {
			continue
		}

		delegation, err := k.GetDelegation(ctx, delAddr, valDstAddr)
		if err != nil && errors.Is(err, collections.ErrNotFound) {
			// If deleted, delegation has zero shares, and we can't unbond any more
			continue
		} else if err != nil {
			return nil, err
		}

		for i, share := range sharesToUnbond {
			sharesToUnbond[i].Amount = math.LegacyMinDec(share.Amount, delegation.Shares.AmountOf(share.Denom))
		}

		tokensToBurn, err := k.Unbond(ctx, delAddr, valDstAddr, sharesToUnbond)
		if err != nil {
			return nil, err
		}

		dstValidator, err := k.Validators.Get(ctx, valDstAddr)
		if err != nil {
			return nil, err
		}

		// tokens of a redelegation currently live in the destination validator
		// therefore we must burn tokens from the destination-validator's bonding status
		switch {
		case dstValidator.IsBonded():
			bondedBurnedAmount = bondedBurnedAmount.Add(tokensToBurn...)
		case dstValidator.IsUnbonded() || dstValidator.IsUnbonding():
			notBondedBurnedAmount = notBondedBurnedAmount.Add(tokensToBurn...)
		default:
			panic("unknown validator status")
		}
	}

	if err := k.burnBondedTokens(ctx, bondedBurnedAmount); err != nil {
		return nil, err
	}

	if err := k.burnNotBondedTokens(ctx, notBondedBurnedAmount); err != nil {
		return nil, err
	}

	return totalSlashAmount, nil
}
