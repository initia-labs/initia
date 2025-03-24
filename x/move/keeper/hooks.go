package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

// SlashHooks
// Wrapper struct
type Hooks struct {
	k Keeper
}

var _ stakingtypes.SlashingHooks = Hooks{}

// Create new distribution hooks
func (k Keeper) Hooks() Hooks { return Hooks{k} }

func (h Hooks) SlashUnbondingDelegations(ctx context.Context, valAddr sdk.ValAddress, fraction math.LegacyDec) error {
	return h.k.SlashUnbondingDelegations(ctx, valAddr, fraction)
}
