package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/initiavm/api"
	vmtypes "github.com/initia-labs/initiavm/types"
)

func (k Keeper) Initialize(
	ctx sdk.Context,
	moduleBytes [][]byte,
	allowArbitrary bool,
	allowedPublishers []string,
) error {
	ctx = ctx.WithTxBytes(make([]byte, 32))

	api := NewApi(k, ctx)
	env := types.NewEnv(
		ctx,
		types.NextAccountNumber(ctx, k.authKeeper),
		k.IncreaseExecutionCounter(ctx),
	)

	modules := make([]vmtypes.Module, len(moduleBytes))
	for i, moduleBz := range moduleBytes {
		modules[i] = vmtypes.NewModule(moduleBz)
	}

	_allowedPublishers := make([]vmtypes.AccountAddress, len(allowedPublishers))
	for i, addr := range allowedPublishers {
		addr, err := types.AccAddressFromString(addr)
		if err != nil {
			return err
		}

		_allowedPublishers[i] = addr
	}

	// The default upgrade policy is compatible when it's not set,
	// so skip the registration at initialize.
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	if err := k.moveVM.Initialize(kvStore, api, env, vmtypes.NewModuleBundle(modules...), allowArbitrary, _allowedPublishers); err != nil {
		return err
	}

	return nil
}

// InitGenesis sets supply information for genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) ([]abci.ValidatorUpdate, error) {
	k.authKeeper.GetModuleAccount(ctx, types.MoveStakingModuleName)

	params := genState.GetParams()
	if err := k.SetRawParams(ctx, params.ToRaw()); err != nil {
		return nil, err
	}
	k.SetExecutionCounter(ctx, genState.ExecutionCounter)

	if len(genState.GetModules()) == 0 {
		if err := k.Initialize(ctx, genState.GetStdlibs(), params.ArbitraryEnabled, params.AllowedPublishers); err != nil {
			return nil, err
		}
	}

	for _, module := range genState.GetModules() {
		addr, err := types.AccAddressFromString(module.Address)
		if err != nil {
			return nil, err
		}

		if err := k.SetModule(ctx, addr, module.ModuleName, module.RawBytes); err != nil {
			return nil, err
		}
	}

	for _, resource := range genState.GetResources() {
		addr, err := types.AccAddressFromString(resource.Address)
		if err != nil {
			return nil, err
		}

		structTag, err := vmapi.ParseStructTag(resource.StructTag)
		if err != nil {
			return nil, err
		}

		_ = k.SetResource(ctx, addr, structTag, resource.RawBytes)
	}

	for _, tableInfo := range genState.GetTableInfos() {
		err := k.SetTableInfo(ctx, tableInfo)
		if err != nil {
			return nil, err
		}
	}

	for _, tableEntry := range genState.GetTableEntries() {
		err := k.SetTableEntry(ctx, tableEntry)
		if err != nil {
			return nil, err
		}
	}

	dexKeeper := NewDexKeeper(&k)
	for _, dexPair := range genState.GetDexPairs() {
		err := dexKeeper.SetDexPair(ctx, dexPair)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// ExportGenesis export genesis state
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var genState types.GenesisState

	genState.Params = k.GetParams(ctx)

	var modules []types.Module
	var resources []types.Resource
	var tableEntries []types.TableEntry
	var tableInfos []types.TableInfo
	k.IterateVMStore(ctx, func(
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

	dexKeeper := NewDexKeeper(&k)

	var dexPairs []types.DexPair
	dexKeeper.IterateDexPair(ctx, func(dexPair types.DexPair) {
		dexPairs = append(dexPairs, dexPair)
	})

	genState.Modules = modules
	genState.Resources = resources
	genState.TableInfos = tableInfos
	genState.TableEntries = tableEntries
	genState.ExecutionCounter = k.GetExecutionCounter(ctx)
	genState.DexPairs = dexPairs

	return &genState
}
