package keeper

import (
	"github.com/initia-labs/initia/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// GetBondedPool returns the bonded tokens pool's module account
func (k Keeper) GetBondedPool(ctx sdk.Context) (bondedPool authtypes.ModuleAccountI) {
	return k.authKeeper.GetModuleAccount(ctx, types.BondedPoolName)
}

// GetNotBondedPool returns the not bonded tokens pool's module account
func (k Keeper) GetNotBondedPool(ctx sdk.Context) (notBondedPool authtypes.ModuleAccountI) {
	return k.authKeeper.GetModuleAccount(ctx, types.NotBondedPoolName)
}

// bondedTokensToNotBonded transfers coins from the bonded to the not bonded pool within staking
func (k Keeper) bondedTokensToNotBonded(ctx sdk.Context, tokens sdk.Coins) {
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.BondedPoolName, types.NotBondedPoolName, tokens); err != nil {
		panic(err)
	}
}

// notBondedTokensToBonded transfers coins from the not bonded to the bonded pool within staking
func (k Keeper) notBondedTokensToBonded(ctx sdk.Context, tokens sdk.Coins) {
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.NotBondedPoolName, types.BondedPoolName, tokens); err != nil {
		panic(err)
	}
}

// burnBondedTokens removes coins from the bonded pool module account
func (k Keeper) burnBondedTokens(ctx sdk.Context, tokens sdk.Coins) error {
	if tokens.IsAnyNegative() {
		// skip as no coins need to be burned
		return nil
	}

	return k.bankKeeper.BurnCoins(ctx, types.BondedPoolName, tokens)
}

// burnNotBondedTokens removes coins from the not bonded pool module account
func (k Keeper) burnNotBondedTokens(ctx sdk.Context, tokens sdk.Coins) error {
	if tokens.IsAnyNegative() {
		// skip as no coins need to be burned
		return nil
	}

	return k.bankKeeper.BurnCoins(ctx, types.NotBondedPoolName, tokens)
}
