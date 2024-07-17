package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

// InitGenesis initializes the ibc-transfer state and binds to PortID.
func (k Keeper) InitGenesis(ctx context.Context, state types.GenesisState) {
	if err := k.PortID.Set(ctx, state.PortId); err != nil {
		panic(err)
	}

	for _, trace := range state.ClassTraces {
		if err := k.ClassTraces.Set(ctx, trace.Hash(), trace); err != nil {
			panic(err)
		}
	}

	for _, data := range state.ClassData {
		if err := k.ClassData.Set(ctx, data.TraceHash, data.Data); err != nil {
			panic(err)
		}
	}

	for _, data := range state.TokenData {
		if err := k.TokenData.Set(ctx, collections.Join(data.TraceHash, data.TokenId), data.Data); err != nil {
			panic(err)
		}
	}

	// Only try to bind to port if it is not already bound, since we may already own
	// port capability from capability InitGenesis
	if !k.IsBound(ctx, state.PortId) {
		// transfer module binds to the transfer port on InitChain
		// and claims the returned capability
		err := k.BindPort(ctx, state.PortId)
		if err != nil {
			panic(fmt.Sprintf("could not claim port capability: %v", err))
		}
	}

	if err := k.Params.Set(ctx, state.Params); err != nil {
		panic(err)
	}
}

// ExportGenesis exports ibc-transfer module's portID and denom trace info into its genesis state.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	portID, err := k.PortID.Get(ctx)
	if err != nil {
		panic(err)
	}

	allTraces, err := k.GetAllClassTraces(ctx)
	if err != nil {
		panic(err)
	}

	var classData []types.ClassData
	err = k.ClassData.Walk(ctx, nil, func(key []byte, value string) (stop bool, err error) {
		classData = append(classData, types.ClassData{
			TraceHash: key,
			Data:      value,
		})
		return false, nil
	})
	if err != nil {
		panic(err)
	}

	var tokenData []types.TokenData
	err = k.TokenData.Walk(ctx, nil, func(key collections.Pair[[]byte, string], value string) (stop bool, err error) {
		tokenData = append(tokenData, types.TokenData{
			TraceHash: key.K1(),
			TokenId:   key.K2(),
			Data:      value,
		})
		return false, nil
	})
	if err != nil {
		panic(err)
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		PortId:      portID,
		ClassTraces: allTraces,
		ClassData:   classData,
		TokenData:   tokenData,
		Params:      params,
	}
}
