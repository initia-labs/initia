package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) SetAllowed(ctx context.Context, addr sdk.AccAddress, allowed bool) error {
	return k.ACLs.Set(ctx, addr.Bytes(), allowed)
}

func (k Keeper) GetAllowed(ctx context.Context, addr sdk.AccAddress) (bool, error) {
	acl, err := k.ACLs.Get(ctx, addr.Bytes())
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		params, err := k.Params.Get(ctx)
		if err != nil {
			return false, err
		}

		return params.DefaultAllowed, nil
	} else if err != nil {
		return false, err
	}

	return acl, nil
}
