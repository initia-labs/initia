package keeper

import (
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

// returns voting power weights of bond denoms
func (k VotingPowerKeeper) GetVotingPowerWeights(ctx sdk.Context, bondDenoms []string) sdk.DecCoins {
	baseDenom := k.BaseDenom(ctx)
	powerWeights := sdk.NewDecCoins()

	for _, denom := range bondDenoms {
		var powerWeight sdk.Dec
		if denom == baseDenom {
			powerWeight = sdk.OneDec()
		} else {
			metadataLP, err := types.MetadataAddressFromDenom(denom)
			if err != nil {
				// ignore error to avoid chain halt due to wrong denom
				continue
			}

			balanceBase, _, weightBase, weightQuote, err := NewDexKeeper(k.Keeper).getPoolInfo(ctx, metadataLP)
			if err != nil {
				// ignore error to avoid chain halt due to wrong denom
				continue
			}

			totalShare, err := NewMoveBankKeeper(k.Keeper).GetSupply(ctx, denom)
			if err != nil {
				// ignore error to avoid chain halt due to wrong denom
				continue
			}

			if balanceBase.IsZero() || totalShare.IsZero() ||
				weightBase.IsZero() || weightQuote.IsZero() {
				continue
			}

			// weight = balanceBase / totalShare => compute locked base balance
			//          * (weightBase + weightQuote) / weightBase => dilute weight
			powerWeight = sdk.NewDecFromInt(balanceBase).QuoInt(totalShare).
				Quo(weightBase).Mul(weightBase.Add(weightQuote))
		}

		powerWeights = powerWeights.Add(sdk.NewDecCoinFromDec(denom, powerWeight))
	}

	return powerWeights
}
