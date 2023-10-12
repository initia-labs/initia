package keeper

import (
	"github.com/cometbft/cometbft/libs/log"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	moveconfig "github.com/initia-labs/initia/x/move/config"
	"github.com/initia-labs/initia/x/move/types"

	vm "github.com/initia-labs/initiavm"
	vmapi "github.com/initia-labs/initiavm/api"
	vmtypes "github.com/initia-labs/initiavm/types"
)

type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey

	// used only for staking feature
	bankKeeper    types.BankKeeper
	distrKeeper   types.DistributionKeeper
	StakingKeeper types.StakingKeeper
	RewardKeeper  types.RewardKeeper

	// required keepers
	authKeeper          types.AccountKeeper
	communityPoolKeeper types.CommunityPoolKeeper
	// nftTransferKeeper   types.NftTransferKeeper
	msgRouter types.MessageRouter

	config moveconfig.MoveConfig

	// moveVM instance
	moveVM       types.VMEngine
	abciListener *ABCIListener

	feeCollector string
	authority    string
}

func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	authKeeper types.AccountKeeper,
	communityPoolKeeper types.CommunityPoolKeeper,
	// nftTransferKeeper types.NftTransferKeeper,
	msgRouter types.MessageRouter,
	moveConfig moveconfig.MoveConfig,
	bankKeeper types.BankKeeper, // can be nil, if staking not used
	distrKeeper types.DistributionKeeper, // can be nil, if staking not used
	stakingKeeper types.StakingKeeper, // can be nil, if staking not used
	rewardKeeper types.RewardKeeper, // can be nil, if staking not used
	feeCollector string,
	authority string,
) Keeper {
	// ensure that authority is a valid AccAddress
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic("authority is not a valid acc address")
	}

	if moveConfig.ContractSimulationGasLimit == 0 {
		moveConfig.ContractSimulationGasLimit = moveconfig.DefaultContractSimulationGasLimit
	}

	if moveConfig.ContractQueryGasLimit == 0 {
		moveConfig.ContractQueryGasLimit = moveconfig.DefaultContractQueryGasLimit
	}

	moveVM := vm.NewVM()
	abciListener := newABCIListener(&moveVM)
	return Keeper{
		cdc:                 cdc,
		storeKey:            storeKey,
		authKeeper:          authKeeper,
		communityPoolKeeper: communityPoolKeeper,
		// nftTransferKeeper:   nftTransferKeeper,
		msgRouter:     msgRouter,
		config:        moveConfig,
		moveVM:        &moveVM,
		abciListener:  &abciListener,
		bankKeeper:    bankKeeper,
		distrKeeper:   distrKeeper,
		StakingKeeper: stakingKeeper,
		RewardKeeper:  rewardKeeper,
		feeCollector:  feeCollector,
		authority:     authority,
	}
}

// GetAuthority returns the x/move module's authority.
func (ak Keeper) GetAuthority() string {
	return ak.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// GetABCIListener return ABCIListener pointer
func (k Keeper) GetABCIListener() *ABCIListener {
	return k.abciListener
}

// Build simulation vm to avoid moveVM loader cache corruption.
// Currently moveVM does not support cache delegation, so the
// simulation requires whole loader cache flush.
func (k Keeper) buildSimulationVM() types.VMEngine {
	vm := vm.NewVM()
	return &vm
}

// GetExecutionCounter get execution counter for genesis
func (k Keeper) GetExecutionCounter(
	ctx sdk.Context,
) math.Int {
	kvStore := ctx.KVStore(k.storeKey)
	bz := kvStore.Get(types.KeyExecutionCounter)

	counter := sdk.IntProto{Int: sdk.ZeroInt()}
	if len(bz) != 0 {
		k.cdc.MustUnmarshal(bz, &counter)
	}

	return counter.Int
}

// SetExecutionCounter set execution counter for genesis
func (k Keeper) SetExecutionCounter(
	ctx sdk.Context,
	counter math.Int,
) {
	kvStore := ctx.KVStore(k.storeKey)
	kvStore.Set(types.KeyExecutionCounter, k.cdc.MustMarshal(&sdk.IntProto{Int: counter}))
}

// IncreaseExecutionCounter increase execution counter by one
func (k Keeper) IncreaseExecutionCounter(
	ctx sdk.Context,
) math.Int {
	kvStore := ctx.KVStore(k.storeKey)
	bz := kvStore.Get(types.KeyExecutionCounter)

	counter := sdk.IntProto{Int: sdk.ZeroInt()}
	if len(bz) != 0 {
		k.cdc.MustUnmarshal(bz, &counter)
	}

	resCounter := counter.Int
	kvStore.Set(
		types.KeyExecutionCounter,
		k.cdc.MustMarshal(
			&sdk.IntProto{
				Int: resCounter.Add(sdk.OneInt()),
			},
		),
	)

	return resCounter
}

// SetModule store Module bytes
// This function should be used only when InitGenesis
func (k Keeper) SetModule(
	ctx sdk.Context,
	addr vmtypes.AccountAddress,
	moduleName string,
	moduleBytes []byte,
) error {
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	bz, err := types.GetModuleKey(addr, moduleName)
	if err != nil {
		return err
	}

	kvStore.Set(bz, moduleBytes)

	return nil
}

// GetModule return Module of the given account address and name
func (k Keeper) GetModule(
	ctx sdk.Context,
	addr vmtypes.AccountAddress,
	moduleName string,
) (types.Module, error) {
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	bz, err := types.GetModuleKey(addr, moduleName)
	if err != nil {
		return types.Module{}, err
	}

	moduleBytes := kvStore.Get(bz)
	if len(moduleBytes) == 0 {
		return types.Module{}, sdkerrors.ErrNotFound
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
	ctx sdk.Context,
	addr vmtypes.AccountAddress,
	moduleName string,
) (bool, error) {
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	bz, err := types.GetModuleKey(addr, moduleName)
	if err != nil {
		return false, err
	}

	return kvStore.Has(bz), nil
}

// SetResource store Resource bytes
// This function should be used only when InitGenesis
func (k Keeper) SetResource(
	ctx sdk.Context,
	addr vmtypes.AccountAddress,
	structTag vmtypes.StructTag,
	resourceBytes []byte,
) error {
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	bz, err := types.GetResourceKey(addr, structTag)
	if err != nil {
		return err
	}

	kvStore.Set(bz, resourceBytes)
	return nil
}

// HasResource return boolean wether the store contains a data with the resource key
func (k Keeper) HasResource(
	ctx sdk.Context,
	addr vmtypes.AccountAddress,
	structTag vmtypes.StructTag,
) (bool, error) {
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	keyBz, err := types.GetResourceKey(addr, structTag)
	if err != nil {
		return false, err
	}

	return kvStore.Has(keyBz), nil
}

// GetResourceBytes return Resource bytes of the given account address and struct tag
func (k Keeper) GetResourceBytes(
	ctx sdk.Context,
	addr vmtypes.AccountAddress,
	structTag vmtypes.StructTag,
) ([]byte, error) {
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	keyBz, err := types.GetResourceKey(addr, structTag)
	if err != nil {
		return nil, err
	}

	resourceBytes := kvStore.Get(keyBz)
	if len(resourceBytes) == 0 {
		return nil, sdkerrors.ErrNotFound
	}

	return resourceBytes, nil
}

// GetResource return Resource of the given account address and struct tag
func (k Keeper) GetResource(
	ctx sdk.Context,
	addr vmtypes.AccountAddress,
	structTag vmtypes.StructTag,
) (types.Resource, error) {
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	keyBz, err := types.GetResourceKey(addr, structTag)
	if err != nil {
		return types.Resource{}, err
	}

	resourceBytes := kvStore.Get(keyBz)
	if len(resourceBytes) == 0 {
		return types.Resource{}, sdkerrors.ErrNotFound
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
	ctx sdk.Context,
	tableAddr vmtypes.AccountAddress,
) bool {
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	return kvStore.Has(types.GetTableInfoKey(tableAddr))
}

// GetTableInfo return table entry
func (k Keeper) GetTableInfo(
	ctx sdk.Context,
	tableAddr vmtypes.AccountAddress,
) (types.TableInfo, error) {
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	bz := kvStore.Get(types.GetTableInfoKey(tableAddr))
	if bz == nil {
		return types.TableInfo{}, sdkerrors.ErrNotFound
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
	ctx sdk.Context,
	tableInfo types.TableInfo,
) error {
	tableAddr, err := types.AccAddressFromString(tableInfo.Address)
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

	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	kvStore.Set(types.GetTableInfoKey(tableAddr), infoBz)

	return nil
}

// HasTableEntry return existence of table entry
func (k Keeper) HasTableEntry(
	ctx sdk.Context,
	tableAddr vmtypes.AccountAddress,
	key []byte,
) bool {
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	return kvStore.Has(types.GetTableEntryKey(tableAddr, key))
}

// GetTableEntry return table entry
func (k Keeper) GetTableEntry(
	ctx sdk.Context,
	tableAddr vmtypes.AccountAddress,
	keyBz []byte,
) (types.TableEntry, error) {
	info, err := k.GetTableInfo(ctx, tableAddr)
	if err != nil {
		return types.TableEntry{}, err
	}

	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	valueBz := kvStore.Get(types.GetTableEntryKey(tableAddr, keyBz))
	if valueBz == nil {
		return types.TableEntry{}, sdkerrors.ErrNotFound
	}

	keyTypeTag, err := vmapi.TypeTagFromString(info.KeyType)
	if err != nil {
		return types.TableEntry{}, err
	}

	valueTypeTag, err := vmapi.TypeTagFromString(info.ValueType)
	if err != nil {
		return types.TableEntry{}, err
	}

	keyStr, err := vmapi.DecodeMoveValue(kvStore, keyTypeTag, keyBz)
	if err != nil {
		return types.TableEntry{}, err
	}

	valueStr, err := vmapi.DecodeMoveValue(kvStore, valueTypeTag, valueBz)
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
	ctx sdk.Context,
	tableAddr vmtypes.AccountAddress,
	key []byte,
) (types.TableEntry, error) {
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	bz := kvStore.Get(types.GetTableEntryKey(tableAddr, key))
	if bz == nil {
		return types.TableEntry{}, sdkerrors.ErrNotFound
	}

	return types.TableEntry{
		Address:    tableAddr.String(),
		KeyBytes:   key,
		ValueBytes: bz,
	}, nil
}

// SetTableEntry store table entry data
func (k Keeper) SetTableEntry(
	ctx sdk.Context,
	tableEntry types.TableEntry,
) error {
	tableAddr, err := types.AccAddressFromString(tableEntry.Address)
	if err != nil {
		return err
	}

	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	kvStore.Set(types.GetTableEntryKey(tableAddr, tableEntry.KeyBytes), tableEntry.ValueBytes)
	return nil
}

// IterateVMStore iterate VMStore store for genesis export
func (k Keeper) IterateVMStore(ctx sdk.Context, cb func(*types.Module, *types.Resource, *types.TableInfo, *types.TableEntry)) {
	kvStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore)
	iter := kvStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		cursor := types.AddressBytesLength
		addrBytes := key[:cursor]
		separator := key[cursor]

		vmAddr, err := vmtypes.NewAccountAddressFromBytes(addrBytes)
		if err != nil {
			panic(err)
		}

		cursor += 1
		if separator == types.ModuleSeparator {
			// Module
			moduleName, err := vmtypes.BcsDeserializeIdentifier(key[cursor:])
			if err != nil {
				panic(err)
			}

			policy, err := NewCodeKeeper(&k).GetUpgradePolicy(ctx, vmAddr, string(moduleName))
			if err != nil {
				panic(err)
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
				panic(err)
			}

			structTagStr, err := vmapi.StringifyStructTag(structTag)
			if err != nil {
				panic(err)
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
				panic(err)
			}

			keyType, err := vmapi.StringifyTypeTag(tableInfo.KeyType)
			if err != nil {
				panic(err)
			}

			valueType, err := vmapi.StringifyTypeTag(tableInfo.ValueType)
			if err != nil {
				panic(err)
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
		} else {
			panic("unknown prefix")
		}
	}
}

func (k Keeper) ExecuteViewFunction(
	ctx sdk.Context,
	moduleAddr vmtypes.AccountAddress,
	moduleName string,
	functionName string,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) (string, error) {
	if payload, err := types.BuildExecuteViewFunctionPayload(
		moduleAddr,
		moduleName,
		functionName,
		typeArgs,
		args,
	); err != nil {
		return "", err
	} else {

		api := NewApi(k, ctx)
		env := types.NewEnv(
			ctx,
			types.NextAccountNumber(ctx, k.authKeeper),
			k.IncreaseExecutionCounter(ctx),
		)

		return k.moveVM.ExecuteViewFunction(
			prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore),
			api,
			env,
			k.config.ContractQueryGasLimit,
			payload,
		)
	}
}

// DecodeMoveResource decode raw move resource bytes
// into `MoveResource` json string
func (k Keeper) DecodeMoveResource(
	ctx sdk.Context,
	structTag vmtypes.StructTag,
	resourceBytes []byte,
) ([]byte, error) {
	return vmapi.DecodeMoveResource(
		prefix.NewStore(ctx.KVStore(k.storeKey), types.PrefixKeyVMStore),
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
