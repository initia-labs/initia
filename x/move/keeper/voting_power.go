package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

var _ mstakingtypes.VotingPowerKeeper = VotingPowerKeeper{}

// VotingPowerKeeper implements move wrapper for types.VotingPowerKeeper interface
type VotingPowerKeeper struct {
	*Keeper
}

// NewVotingPowerKeeper return new VotingPowerKeeper instance
func NewVotingPowerKeeper(k *Keeper) VotingPowerKeeper {
	return VotingPowerKeeper{k}
}

// returns voting power weights of bond denoms.
// if denom is base denom, weight is 1.
// if denom is not base denom, weight = locked base balance / total share,
// which means we only consider locked base balance for voting power.
func (k VotingPowerKeeper) GetVotingPowerWeights(ctx context.Context, bondDenoms []string) (sdk.DecCoins, error) {
	baseDenom, err := k.BaseDenom(ctx)
	if err != nil {
		return nil, err
	}

	powerWeights := sdk.NewDecCoins()
	for _, denom := range bondDenoms {
		var powerWeight math.LegacyDec
		if denom == baseDenom {
			powerWeight = math.LegacyOneDec()
		} else {
			metadataLP, err := types.MetadataAddressFromDenom(denom)
			if err != nil {
				// ignore error to avoid chain halt due to wrong denom
				continue
			}

			balanceBase, _, err := NewDexKeeper(k.Keeper).getPoolBalances(ctx, metadataLP)
			if err != nil {
				// ignore error to avoid chain halt due to wrong denom
				continue
			}

			totalShare, err := NewMoveBankKeeper(k.Keeper).GetSupply(ctx, denom)
			if err != nil {
				// ignore error to avoid chain halt due to wrong denom
				continue
			}

			// if balance is zero, use zero power
			if balanceBase.IsZero() || totalShare.IsZero() {
				continue
			}

			// weight = balanceBase / totalShare => compute locked base balance
			powerWeight = math.LegacyNewDecFromInt(balanceBase).QuoInt(totalShare)
		}

		powerWeights = powerWeights.Add(sdk.NewDecCoinFromDec(denom, powerWeight))
	}

	return powerWeights, nil
}
