package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestoretypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/crypto/sha3"

	moveconfig "github.com/initia-labs/initia/x/move/config"
	"github.com/initia-labs/initia/x/move/types"

	vm "github.com/initia-labs/movevm"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"
)

type Keeper struct {
	cdc          codec.Codec
	storeService corestoretypes.KVStoreService

	// used only for staking feature
	distrKeeper         types.DistributionKeeper
	StakingKeeper       types.StakingKeeper
	RewardKeeper        types.RewardKeeper
	communityPoolKeeper types.CommunityPoolKeeper

	// required keepers
	authKeeper   types.AccountKeeper
	bankKeeper   types.BankKeeper
	oracleKeeper types.OracleKeeper

	// Msg server router
	msgRouter  baseapp.MessageRouter
	grpcRouter *baseapp.GRPCQueryRouter

	config moveconfig.MoveConfig

	// moveVM instance
	moveVM types.VMEngine

	feeCollector string
	authority    string

	Schema           collections.Schema
	ExecutionCounter collections.Sequence
	Params           collections.Item[types.RawParams]
	DexPairs         collections.Map[[]byte, []byte]
	VMStore          collections.Map[[]byte, []byte]

	ac address.Codec
	vc address.Codec

	vmQueryWhiteList types.VMQueryWhiteList
}

func NewKeeper(
	cdc codec.Codec,
	storeService corestoretypes.KVStoreService,
	authKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	oracleKeeper types.OracleKeeper,
	msgRouter baseapp.MessageRouter,
	grpcRouter *baseapp.GRPCQueryRouter,
	moveConfig moveconfig.MoveConfig,
	distrKeeper types.DistributionKeeper, // can be nil, if staking not used
	stakingKeeper types.StakingKeeper, // can be nil, if staking not used
	rewardKeeper types.RewardKeeper, // can be nil, if staking not used
	communityPoolKeeper types.CommunityPoolKeeper, // can be nil, if staking not used
	feeCollector string,
	authority string,
	ac, vc address.Codec,
) *Keeper {
	// ensure that authority is a valid AccAddress
	if _, err := ac.StringToBytes(authority); err != nil {
		panic("authority is not a valid acc address")
	}

	if moveConfig.ModuleCacheCapacity == 0 {
		moveConfig.ModuleCacheCapacity = moveconfig.DefaultModuleCacheCapacity
	}

	if moveConfig.ScriptCacheCapacity == 0 {
		moveConfig.ScriptCacheCapacity = moveconfig.DefaultScriptCacheCapacity
	}

	if moveConfig.ContractSimulationGasLimit == 0 {
		moveConfig.ContractSimulationGasLimit = moveconfig.DefaultContractSimulationGasLimit
	}

	moveVM := vm.NewVM(moveConfig.ModuleCacheCapacity, moveConfig.ScriptCacheCapacity)

	sb := collections.NewSchemaBuilder(storeService)
	k := &Keeper{
		cdc:                 cdc,
		storeService:        storeService,
		authKeeper:          authKeeper,
		bankKeeper:          bankKeeper,
		oracleKeeper:        oracleKeeper,
		msgRouter:           msgRouter,
		grpcRouter:          grpcRouter,
		config:              moveConfig,
		moveVM:              &moveVM,
		distrKeeper:         distrKeeper,
		StakingKeeper:       stakingKeeper,
		RewardKeeper:        rewardKeeper,
		communityPoolKeeper: communityPoolKeeper,
		feeCollector:        feeCollector,
		authority:           authority,

		ExecutionCounter: collections.NewSequence(sb, types.ExecutionCounterKey, "execution_counter"),
		Params:           collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.RawParams](cdc)),
		DexPairs:         collections.NewMap(sb, types.DexPairPrefix, "dex_pairs", collections.BytesKey, collections.BytesValue),
		VMStore:          collections.NewMap(sb, types.VMStorePrefix, "vm_store", collections.BytesKey, collections.BytesValue),

		ac: ac,
		vc: vc,

		vmQueryWhiteList: types.DefaultVMQueryWhiteList(ac),
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

// WithVMQueryWhitelist overrides vmQueryWhitelist
func (k Keeper) WithVMQueryWhitelist(vmQueryWhiteList types.VMQueryWhiteList) Keeper {
	k.vmQueryWhiteList = vmQueryWhiteList
	return k
}

// GetAuthority returns the x/move module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetExecutionCounter get execution counter for genesis
func (k Keeper) GetExecutionCounter(
	ctx context.Context,
) (uint64, error) {
	counter, err := k.ExecutionCounter.Peek(ctx)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return counter, nil
}

// SetModule store Module bytes
// This function should be used only when InitGenesis
func (k Keeper) SetModule(
	ctx context.Context,
	addr vmtypes.AccountAddress,
	moduleName string,
	moduleBytes []byte,
) error {
	if moduleKey, err := types.GetModuleKey(addr, moduleName); err != nil {
		return err
	} else if err := k.VMStore.Set(ctx, moduleKey, moduleBytes); err != nil {
		return err
	}

	checksum := sha3.Sum256(moduleBytes)
	if checksumKey, err := types.GetChecksumKey(addr, moduleName); err != nil {
		return err
	} else if err := k.VMStore.Set(ctx, checksumKey, checksum[:]); err != nil {
		return err
	}

	return nil
}

// GetModule return Module of the given account address and name
func (k Keeper) GetModule(
	ctx context.Context,
	addr vmtypes.AccountAddress,
	moduleName string,
) (types.Module, error) {
	bz, err := types.GetModuleKey(addr, moduleName)
	if err != nil {
		return types.Module{}, err
	}

	moduleBytes, err := k.VMStore.Get(ctx, bz)
	if err != nil {
		return types.Module{}, err
	}

	bz, err = k.DecodeModuleBytes(moduleBytes)
	if err != nil {
		return types.Module{}, err
	}

	policy, err := NewCodeKeeper(&k).GetUpgradePolicy(ctx, addr, moduleName)
	if err != nil {
		return types.Module{}, err
	}

	return types.Module{
		Address:       addr.String(),
		ModuleName:    moduleName,
		Abi:           string(bz),
		RawBytes:      moduleBytes,
		UpgradePolicy: policy,
	}, nil
}

// HasModule return boolean of whether module exists or not
func (k Keeper) HasModule(
	ctx context.Context,
	addr vmtypes.AccountAddress,
	moduleName string,
) (bool, error) {
	bz, err := types.GetModuleKey(addr, moduleName)
	if err != nil {
		return false, err
	}

	return k.VMStore.Has(ctx, bz)
}

// SetResource store Resource bytes
// This function should be used only when InitGenesis
func (k Keeper) SetResource(
	ctx context.Context,
	addr vmtypes.AccountAddress,
	structTag vmtypes.StructTag,
	resourceBytes []byte,
) error {
	bz, err := types.GetResourceKey(addr, structTag)
	if err != nil {
		return err
	}

	return k.VMStore.Set(ctx, bz, resourceBytes)
}

// HasResource return boolean wether the store contains a data with the resource key
func (k Keeper) HasResource(
	ctx context.Context,
	addr vmtypes.AccountAddress,
	structTag vmtypes.StructTag,
) (bool, error) {
	keyBz, err := types.GetResourceKey(addr, structTag)
	if err != nil {
		return false, err
	}

	return k.VMStore.Has(ctx, keyBz)
}

// GetResourceBytes return Resource bytes of the given account address and struct tag
func (k Keeper) GetResourceBytes(
	ctx context.Context,
	addr vmtypes.AccountAddress,
	structTag vmtypes.StructTag,
) ([]byte, error) {
	keyBz, err := types.GetResourceKey(addr, structTag)
	if err != nil {
		return nil, err
	}

	return k.VMStore.Get(ctx, keyBz)
}

// GetResource return Resource of the given account address and struct tag
func (k Keeper) GetResource(
	ctx context.Context,
	addr vmtypes.AccountAddress,
	structTag vmtypes.StructTag,
) (types.Resource, error) {
	keyBz, err := types.GetResourceKey(addr, structTag)
	if err != nil {
		return types.Resource{}, err
	}

	resourceBytes, err := k.VMStore.Get(ctx, keyBz)
	if err != nil {
		return types.Resource{}, err
	}

	bz, err := k.DecodeMoveResource(ctx, structTag, resourceBytes)
	if err != nil {
		return types.Resource{}, err
	}

	structTagStr, err := vmapi.StringifyStructTag(structTag)
	if err != nil {
		return types.Resource{}, err
	}

	return types.Resource{
		Address:      addr.String(),
		StructTag:    structTagStr,
		MoveResource: string(bz),
		RawBytes:     resourceBytes,
	}, nil
}

// HasTableInfo return existence of table info
func (k Keeper) HasTableInfo(
	ctx context.Context,
	tableAddr vmtypes.AccountAddress,
) (bool, error) {
	keyBz := types.GetTableInfoKey(tableAddr)
	return k.VMStore.Has(ctx, keyBz)
}

// GetTableInfo return table entry
func (k Keeper) GetTableInfo(
	ctx context.Context,
	tableAddr vmtypes.AccountAddress,
) (types.TableInfo, error) {
	bz, err := k.VMStore.Get(ctx, types.GetTableInfoKey(tableAddr))
	if err != nil {
		return types.TableInfo{}, err
	}

	tableInfo, err := vmtypes.BcsDeserializeTableInfo(bz)
	if err != nil {
		return types.TableInfo{}, err
	}

	keyType, err := vmapi.StringifyTypeTag(tableInfo.KeyType)
	if err != nil {
		return types.TableInfo{}, err
	}

	valueType, err := vmapi.StringifyTypeTag(tableInfo.ValueType)
	if err != nil {
		return types.TableInfo{}, err
	}

	return types.TableInfo{
		Address:   tableAddr.String(),
		KeyType:   keyType,
		ValueType: valueType,
	}, nil
}

// SetTableInfo store table info data
func (k Keeper) SetTableInfo(
	ctx context.Context,
	tableInfo types.TableInfo,
) error {
	tableAddr, err := types.AccAddressFromString(k.ac, tableInfo.Address)
	if err != nil {
		return err
	}

	keyType, err := vmapi.TypeTagFromString(tableInfo.KeyType)
	if err != nil {
		return err
	}

	valueType, err := vmapi.TypeTagFromString(tableInfo.ValueType)
	if err != nil {
		return err
	}

	info := vmtypes.TableInfo{
		KeyType:   keyType,
		ValueType: valueType,
	}
	infoBz, err := info.BcsSerialize()
	if err != nil {
		return err
	}

	return k.VMStore.Set(ctx, types.GetTableInfoKey(tableAddr), infoBz)
}

// HasTableEntry return existence of table entry
func (k Keeper) HasTableEntry(
	ctx context.Context,
	tableAddr vmtypes.AccountAddress,
	key []byte,
) (bool, error) {
	return k.VMStore.Has(ctx, types.GetTableEntryKey(tableAddr, key))
}

// GetTableEntry return table entry
func (k Keeper) GetTableEntry(
	ctx context.Context,
	tableAddr vmtypes.AccountAddress,
	keyBz []byte,
) (types.TableEntry, error) {
	info, err := k.GetTableInfo(ctx, tableAddr)
	if err != nil {
		return types.TableEntry{}, err
	}

	valueBz, err := k.VMStore.Get(ctx, types.GetTableEntryKey(tableAddr, keyBz))
	if err != nil {
		return types.TableEntry{}, err
	}

	keyTypeTag, err := vmapi.TypeTagFromString(info.KeyType)
	if err != nil {
		return types.TableEntry{}, err
	}

	valueTypeTag, err := vmapi.TypeTagFromString(info.ValueType)
	if err != nil {
		return types.TableEntry{}, err
	}

	vmStore := types.NewVMStore(ctx, k.VMStore)
	keyStr, err := vmapi.DecodeMoveValue(vmStore, keyTypeTag, keyBz)
	if err != nil {
		return types.TableEntry{}, err
	}

	valueStr, err := vmapi.DecodeMoveValue(vmStore, valueTypeTag, valueBz)
	if err != nil {
		return types.TableEntry{}, err
	}

	return types.TableEntry{
		Address:    tableAddr.String(),
		Key:        string(keyStr),
		Value:      string(valueStr),
		KeyBytes:   keyBz,
		ValueBytes: valueBz,
	}, nil
}

// GetTableEntryBytes return a raw table entry without decoding the
// key and value.
func (k Keeper) GetTableEntryBytes(
	ctx context.Context,
	tableAddr vmtypes.AccountAddress,
	key []byte,
) (types.TableEntry, error) {
	bz, err := k.VMStore.Get(ctx, types.GetTableEntryKey(tableAddr, key))
	if err != nil {
		return types.TableEntry{}, err
	}

	return types.TableEntry{
		Address:    tableAddr.String(),
		KeyBytes:   key,
		ValueBytes: bz,
	}, nil
}

// SetTableEntry store table entry data
func (k Keeper) SetTableEntry(
	ctx context.Context,
	tableEntry types.TableEntry,
) error {
	tableAddr, err := types.AccAddressFromString(k.ac, tableEntry.Address)
	if err != nil {
		return err
	}

	return k.VMStore.Set(ctx, types.GetTableEntryKey(tableAddr, tableEntry.KeyBytes), tableEntry.ValueBytes)
}

// IterateVMStore iterate VMStore store for genesis export
func (k Keeper) IterateVMStore(ctx context.Context, cb func(*types.Module, *types.Resource, *types.TableInfo, *types.TableEntry)) error {
	err := k.VMStore.Walk(ctx, nil, func(key, value []byte) (stop bool, err error) {
		cursor := types.AddressBytesLength
		addrBytes := key[:cursor]
		separator := key[cursor]

		vmAddr, err := vmtypes.NewAccountAddressFromBytes(addrBytes)
		if err != nil {
			return true, err
		}

		cursor += 1
		if separator == types.ModuleSeparator {
			// Module
			moduleName, err := vmtypes.BcsDeserializeIdentifier(key[cursor:])
			if err != nil {
				return true, err
			}

			policy, err := NewCodeKeeper(&k).GetUpgradePolicy(ctx, vmAddr, string(moduleName))
			if err != nil {
				return true, err
			}

			cb(&types.Module{
				Address:       vmAddr.String(),
				ModuleName:    string(moduleName),
				RawBytes:      value,
				UpgradePolicy: policy,
			}, nil, nil, nil)
		} else if separator == types.ResourceSeparator {
			// Resource
			structTag, err := vmtypes.BcsDeserializeStructTag(key[cursor:])
			if err != nil {
				return true, err
			}

			structTagStr, err := vmapi.StringifyStructTag(structTag)
			if err != nil {
				return true, err
			}

			cb(nil, &types.Resource{
				Address:   vmAddr.String(),
				StructTag: structTagStr,
				RawBytes:  value,
			}, nil, nil)
		} else if separator == types.TableInfoSeparator {
			// Table Info
			tableInfo, err := vmtypes.BcsDeserializeTableInfo(value)
			if err != nil {
				return true, err
			}

			keyType, err := vmapi.StringifyTypeTag(tableInfo.KeyType)
			if err != nil {
				return true, err
			}

			valueType, err := vmapi.StringifyTypeTag(tableInfo.ValueType)
			if err != nil {
				return true, err
			}

			cb(nil, nil, &types.TableInfo{
				Address:   vmAddr.String(),
				KeyType:   keyType,
				ValueType: valueType,
			}, nil)
		} else if separator == types.TableEntrySeparator {
			// Table Entry
			cb(nil, nil, nil, &types.TableEntry{
				Address:    vmAddr.String(),
				KeyBytes:   key[cursor:],
				ValueBytes: value,
			})
		} else if separator == types.ChecksumSeparator {
			// ignore checksum
		} else {
			return true, errors.New("unknown prefix")
		}

		return false, nil
	})

	return err
}

// DecodeMoveResource decode raw move resource bytes
// into `MoveResource` json string
func (k Keeper) DecodeMoveResource(
	ctx context.Context,
	structTag vmtypes.StructTag,
	resourceBytes []byte,
) ([]byte, error) {
	return vmapi.DecodeMoveResource(
		types.NewVMStore(ctx, k.VMStore),
		structTag,
		resourceBytes,
	)
}

// DecodeModuleBytes decode raw module bytes
// into `MoveModule` json string
func (k Keeper) DecodeModuleBytes(
	moduleBytes []byte,
) ([]byte, error) {
	return vmapi.DecodeModuleBytes(
		moduleBytes,
	)
}

// DecodeScriptBytes decode raw script bytes
// into `MoveFunction` json string
func (k Keeper) DecodeScriptBytes(
	scriptBytes []byte,
) ([]byte, error) {
	return vmapi.DecodeScriptBytes(
		scriptBytes,
	)
}
