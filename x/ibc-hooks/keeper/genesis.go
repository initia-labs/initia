package keeper

import (
	"context"

	"github.com/initia-labs/initia/v1/x/ibc-hooks/types"
)

// InitGenesis initializes the hook module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	if err := k.Params.Set(ctx, genState.Params); err != nil {
		panic(err)
	}

	for _, acl := range genState.Acls {
		addr, err := k.ac.StringToBytes(acl.Address)
		if err != nil {
			panic(err)
		}

		if err := k.ACLs.Set(ctx, addr, acl.Allowed); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the hook module's genesis state.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}

	acls := []types.ACL{}
	err = k.ACLs.Walk(ctx, nil, func(addr []byte, allowed bool) (stop bool, err error) {
		addrStr, err := k.ac.BytesToString(addr)
		if err != nil {
			return true, err
		}

		acls = append(acls, types.ACL{
			Address: addrStr,
			Allowed: allowed,
		})

		return false, nil
	})
	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		Params: params,
		Acls:   acls,
	}
}
