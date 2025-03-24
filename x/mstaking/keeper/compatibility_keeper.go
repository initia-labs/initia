package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	cmtprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"

	"github.com/initia-labs/initia/x/mstaking/types"
)

type CompatibilityKeeper struct {
	*Keeper
}

func NewCompatibilityKeeper(k *Keeper) CompatibilityKeeper {
	return CompatibilityKeeper{k}
}

func (k CompatibilityKeeper) Validator(ctx context.Context, addr sdk.ValAddress) (cosmostypes.ValidatorI, error) {
	val, err := k.Keeper.GetValidator(ctx, addr)
	if err != nil {
		return nil, err
	}
	return ValidatorToCosmosValidator(val), nil
}

func (k CompatibilityKeeper) ValidatorByConsAddr(ctx context.Context, addr sdk.ConsAddress) (cosmostypes.ValidatorI, error) {
	val, err := k.Keeper.GetValidatorByConsAddr(ctx, addr)
	if err != nil {
		return nil, err
	}
	return ValidatorToCosmosValidator(val), nil
}

func ValidatorToCosmosValidator(val types.Validator) cosmostypes.Validator {
	unbondingOnHoldRefCount := int64(0)
	unbondingIds := []uint64{}
	if val.UnbondingId >= types.DefaultUnbondingIdStart {
		unbondingOnHoldRefCount++
		unbondingIds = append(unbondingIds, val.UnbondingId)
	}

	return cosmostypes.Validator{
		OperatorAddress:         val.OperatorAddress,
		ConsensusPubkey:         val.ConsensusPubkey,
		Jailed:                  val.Jailed,
		Status:                  cosmostypes.BondStatus(val.Status),
		Tokens:                  val.VotingPower,
		Description:             cosmostypes.Description(val.Description),
		UnbondingHeight:         val.UnbondingHeight,
		UnbondingTime:           val.UnbondingTime,
		Commission:              cosmostypes.NewCommission(val.Commission.Rate, val.Commission.MaxRate, val.Commission.MaxChangeRate),
		MinSelfDelegation:       math.OneInt(),
		UnbondingOnHoldRefCount: unbondingOnHoldRefCount,
		UnbondingIds:            unbondingIds,
		DelegatorShares:         math.LegacyZeroDec(), // not supported
	}
}

// TotalBondedTokens returns sum of voting power
func (k CompatibilityKeeper) TotalBondedTokens(ctx context.Context) (math.Int, error) {
	total := math.ZeroInt()
	err := k.IterateLastValidatorConsPowers(ctx, func(operator sdk.ValAddress, power int64) (stop bool, err error) {
		votingPower := k.VotingPowerFromConsensusPower(ctx, power)
		total = total.Add(votingPower)
		return false, nil
	})

	return total, err
}

func (k CompatibilityKeeper) GetPubKeyByConsAddr(ctx context.Context, addr sdk.ConsAddress) (cmtprotocrypto.PublicKey, error) {
	v, err := k.GetValidatorByConsAddr(ctx, addr)
	if err != nil {
		return cmtprotocrypto.PublicKey{}, err
	}

	pubkey, err := v.CmtConsPublicKey()
	if err != nil {
		return cmtprotocrypto.PublicKey{}, err
	}

	return pubkey, nil
}
