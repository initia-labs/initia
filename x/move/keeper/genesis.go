package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"
)

func (k Keeper) Initialize(
	ctx context.Context,
	moduleBytes [][]byte,
	allowedPublishers []string,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = sdkCtx.WithTxBytes(make([]byte, 32))

	ec, err := k.ExecutionCounter.Next(ctx)
	if err != nil {
		return err
	}

	api := NewApi(k, ctx)
	env := types.NewEnv(
		ctx,
		types.NextAccountNumber(ctx, k.authKeeper),
		ec,
	)

	modules := make([]vmtypes.Module, len(moduleBytes))
	for i, moduleBz := range moduleBytes {
		modules[i] = vmtypes.NewModule(moduleBz)
	}

	_allowedPublishers := make([]vmtypes.AccountAddress, len(allowedPublishers))
	for i, addr := range allowedPublishers {
		addr, err := types.AccAddressFromString(k.ac, addr)
		if err != nil {
			return err
		}

		_allowedPublishers[i] = addr
	}

	// The default upgrade policy is compatible when it's not set,
	// so skip the registration at initialize.
	vmStore := types.NewVMStore(ctx, k.VMStore)
	if err := k.moveVM.Initialize(vmStore, api, env, vmtypes.NewModuleBundle(modules...), _allowedPublishers); err != nil {
		return err
	}

	return nil
}

// InitGenesis sets supply information for genesis.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	k.authKeeper.GetModuleAccount(ctx, types.MoveStakingModuleName)

	params := genState.GetParams()
	if err := k.SetRawParams(ctx, params.ToRaw()); err != nil {
		return err
	}
	if err := k.ExecutionCounter.Set(ctx, genState.ExecutionCounter); err != nil {
		return err
	}

	if len(genState.GetModules()) == 0 {
		if err := k.Initialize(ctx, genState.GetStdlibs(), params.AllowedPublishers); err != nil {
			return err
		}
	}

	for _, module := range genState.GetModules() {
		addr, err := types.AccAddressFromString(k.ac, module.Address)
		if err != nil {
			return err
		}

		if err := k.SetModule(ctx, addr, module.ModuleName, module.RawBytes); err != nil {
			return err
		}
	}

	for _, resource := range genState.GetResources() {
		addr, err := types.AccAddressFromString(k.ac, resource.Address)
		if err != nil {
			return err
		}

		structTag, err := vmapi.ParseStructTag(resource.StructTag)
		if err != nil {
			return err
		}

		_ = k.SetResource(ctx, addr, structTag, resource.RawBytes)
	}

	for _, tableInfo := range genState.GetTableInfos() {
		err := k.SetTableInfo(ctx, tableInfo)
		if err != nil {
			return err
		}
	}

	for _, tableEntry := range genState.GetTableEntries() {
		err := k.SetTableEntry(ctx, tableEntry)
		if err != nil {
			return err
		}
	}

	dexKeeper := NewDexKeeper(&k)
	for _, dexPair := range genState.GetDexPairs() {
		err := dexKeeper.SetDexPair(ctx, dexPair)
		if err != nil {
			return err
		}
	}

	return nil
}

// ExportGenesis export genesis state
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	var genState types.GenesisState

	var err error
	genState.Params, err = k.GetParams(ctx)
	if err != nil {
		panic(err)
	}

	var modules []types.Module
	var resources []types.Resource
	var tableEntries []types.TableEntry
	var tableInfos []types.TableInfo
	err = k.IterateVMStore(ctx, func(
		module *types.Module,
		resource *types.Resource,
		tableInfo *types.TableInfo,
		tableEntry *types.TableEntry,
	) {
		if module != nil {
			modules = append(modules, *module)
		}

		if resource != nil {
			resources = append(resources, *resource)
		}

		if tableInfo != nil {
			tableInfos = append(tableInfos, *tableInfo)
		}

		if tableEntry != nil {
			tableEntries = append(tableEntries, *tableEntry)
		}

	})
	if err != nil {
		panic(err)
	}

	dexKeeper := NewDexKeeper(&k)

	var dexPairs []types.DexPair
	err = dexKeeper.IterateDexPair(ctx, func(dexPair types.DexPair) (bool, error) {
		dexPairs = append(dexPairs, dexPair)
		return false, nil
	})
	if err != nil {
		panic(err)
	}

	genState.Modules = modules
	genState.Resources = resources
	genState.TableInfos = tableInfos
	genState.TableEntries = tableEntries
	genState.DexPairs = dexPairs

	genState.ExecutionCounter, err = k.GetExecutionCounter(ctx)
	if err != nil {
		panic(err)
	}

	return &genState
}
