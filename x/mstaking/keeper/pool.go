package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/mstaking/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetBondedPool returns the bonded tokens pool's module account
func (k Keeper) GetBondedPool(ctx context.Context) (bondedPool sdk.ModuleAccountI) {
	return k.authKeeper.GetModuleAccount(ctx, types.BondedPoolName)
}

// GetNotBondedPool returns the not bonded tokens pool's module account
func (k Keeper) GetNotBondedPool(ctx context.Context) (notBondedPool sdk.ModuleAccountI) {
	return k.authKeeper.GetModuleAccount(ctx, types.NotBondedPoolName)
}

// bondedTokensToNotBonded transfers coins from the bonded to the not bonded pool within staking
func (k Keeper) bondedTokensToNotBonded(ctx context.Context, tokens sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.BondedPoolName, types.NotBondedPoolName, tokens)
}

// notBondedTokensToBonded transfers coins from the not bonded to the bonded pool within staking
func (k Keeper) notBondedTokensToBonded(ctx context.Context, tokens sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.NotBondedPoolName, types.BondedPoolName, tokens)
}

// burnBondedTokens removes coins from the bonded pool module account
func (k Keeper) burnBondedTokens(ctx context.Context, tokens sdk.Coins) error {
	if tokens.IsAnyNegative() {
		// skip as no coins need to be burned
		return nil
	}

	return k.bankKeeper.BurnCoins(ctx, types.BondedPoolName, tokens)
}

// burnNotBondedTokens removes coins from the not bonded pool module account
func (k Keeper) burnNotBondedTokens(ctx context.Context, tokens sdk.Coins) error {
	if tokens.IsAnyNegative() {
		// skip as no coins need to be burned
		return nil
	}

	return k.bankKeeper.BurnCoins(ctx, types.NotBondedPoolName, tokens)
}

// TotalBondedTokens total staking tokens supply which is bonded
func (k Keeper) TotalBondedTokens(ctx context.Context) (math.Int, error) {
	bondedPool := k.GetBondedPool(ctx)
	bondDenoms, err := k.BondDenoms(ctx)
	if err != nil {
		return math.ZeroInt(), err
	}
	var total math.Int
	for _, bondDenom := range bondDenoms {
		total = total.Add(k.bankKeeper.GetBalance(ctx, bondedPool.GetAddress(), bondDenom).Amount)
	}
	return total, nil
}
