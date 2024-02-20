package types

import (
	"math/big"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"

	vmapi "github.com/initia-labs/initiavm/api"
	vmtypes "github.com/initia-labs/initiavm/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	// module names
	MoveModuleNameCoin                 = "coin"
	MoveModuleNameStaking              = "staking"
	MoveModuleNameDex                  = "dex"
	MoveModuleNameNft                  = "nft"
	MoveModuleNameCode                 = "code"
	MoveModuleNameFungibleAsset        = "fungible_asset"
	MoveModuleNamePrimaryFungibleStore = "primary_fungible_store"
	MoveModuleNameManagedCoin          = "managed_coin"
	MoveModuleNameObject               = "object"
	MoveModuleNameSimpleNft            = "initia_nft"
	MoveModuleNameCollection           = "collection"

	// function names for managed_coin
	FunctionNameManagedCoinInitialize = "initialize"
	FunctionNameManagedCoinMint       = "mint"
	FunctionNameManagedCoinBurn       = "burn"

	// function names for simple_nft
	FunctionNameSimpleNftInitialize = "create_collection"
	FunctionNameSimpleNftMint       = "mint"
	FunctionNameSimpleNftBurn       = "burn"

	// function names for coin
	FunctionNameCoinBalance   = "balance"
	FunctionNameCoinRegister  = "register"
	FunctionNameCoinTransfer  = "transfer"
	FunctionNameCoinWhitelist = "whitelist"

	// function names for staking
	FunctionNameStakingInitialize           = "initialize_for_chain"
	FunctionNameStakingDepositReward        = "deposit_reward_for_chain"
	FunctionNameStakingDelegate             = "delegate_script"
	FunctionNameStakingUndelegate           = "undelegate_script"
	FunctionNameStakingRegister             = "register"
	FunctionNameStakingDepositUnbondingCoin = "deposit_unbonding_coin_for_chain"
	FunctionNameStakingSlashUnbondingCoin   = "slash_unbonding_for_chain"

	// function names for dex
	FunctionNameDexInitialize        = "initialize_for_chain"
	FunctionNameDexProvideLiquidity  = "provide_liquidity_script"
	FunctionNameDexWithdrawLiquidity = "withdraw_liquidity_script"
	FunctionNameDexRegister          = "register"
	FunctionNameDexSwap              = "swap_script"

	// function names for object
	FunctionNameObjectTransfer = "transfer"

	// function names for code
	FunctionNameCodePublish              = "publish"
	FunctionNameCodeInitGenesis          = "init_genesis"
	FunctionNameCodeSetAllowArbitrary    = "set_allow_arbitrary"
	FunctionNameCodeSetAllowedPublishers = "set_allowed_publishers"

	// resource names
	ResourceNameFungibleStore = "FungibleStore"
	ResourceNameMetadata      = "Metadata"
	ResourceNameModuleStore   = "ModuleStore"
	ResourceNameSupply        = "Supply"
	ResourceNamePool          = "Pool"
	ResourceNameStakingState  = "StakingState"
	ResourceNameConfig        = "Config"
	ResourceNameMetadataStore = "MetadataStore"
	ResourceNameIssuer        = "Issuer"
	ResourceNameManagingRefs  = "ManagingRefs"
	ResourceNameCollection    = "Collection"
	ResourceNameSimpleNft     = "InitiaNft"
	ResourceNameNft           = "Nft"
)

// TypeTagFromStructTag return type tag with struct tag
func TypeTagFromStructTag(structTag vmtypes.StructTag) vmtypes.TypeTag {
	return &vmtypes.TypeTag__Struct{Value: structTag}
}

// TypeTagsFromTypeArgs convert type args to type tags
func TypeTagsFromTypeArgs(typeArgs []string) ([]vmtypes.TypeTag, error) {
	typeTags := make([]vmtypes.TypeTag, len(typeArgs))
	for i, typeArg := range typeArgs {
		typeTag, err := vmapi.TypeTagFromString(typeArg)
		if err != nil {
			return nil, err
		}

		typeTags[i] = typeTag
	}

	return typeTags, nil
}

// BuildExecuteEntryFunctionPayload return execute entry function payload
func BuildExecuteEntryFunctionPayload(
	moduleAddr vmtypes.AccountAddress,
	moduleName string,
	functionName string,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) (vmtypes.EntryFunction, error) {
	if len(moduleName) == 0 {
		return vmtypes.EntryFunction{}, errors.Wrap(sdkerrors.ErrInvalidRequest, "empty module name")
	}

	if len(functionName) == 0 {
		return vmtypes.EntryFunction{}, errors.Wrap(sdkerrors.ErrInvalidRequest, "empty function name")
	}

	return vmtypes.EntryFunction{
		Module: vmtypes.ModuleId{
			Address: moduleAddr,
			Name:    vmtypes.Identifier(moduleName),
		},
		Function: vmtypes.Identifier(functionName),
		TyArgs:   typeArgs,
		Args:     args,
	}, nil
}

// BuildExecuteScriptPayload return script payload
func BuildExecuteScriptPayload(
	byteCodes []byte,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) (vmtypes.Script, error) {
	if len(byteCodes) == 0 {
		return vmtypes.Script{}, errors.Wrap(sdkerrors.ErrInvalidRequest, "empty code bytes")
	}

	payload := vmtypes.Script{
		Code:   byteCodes,
		TyArgs: typeArgs,
		Args:   args,
	}

	return payload, nil
}

// BuildExecuteViewFunctionPayload return execute view function payload
func BuildExecuteViewFunctionPayload(
	moduleAddr vmtypes.AccountAddress,
	moduleName string,
	functionName string,
	typeArgs []vmtypes.TypeTag,
	args [][]byte,
) (vmtypes.ViewFunction, error) {
	if len(moduleName) == 0 {
		return vmtypes.ViewFunction{}, errors.Wrap(sdkerrors.ErrInvalidRequest, "empty module name")
	}

	if len(functionName) == 0 {
		return vmtypes.ViewFunction{}, errors.Wrap(sdkerrors.ErrInvalidRequest, "empty function name")
	}

	return vmtypes.ViewFunction{
		Module: vmtypes.ModuleId{
			Address: moduleAddr,
			Name:    vmtypes.Identifier(moduleName),
		},
		Function: vmtypes.Identifier(functionName),
		TyArgs:   typeArgs,
		Args:     args,
	}, nil
}

// DeserializeUint64 deserialize uint64 bytes to math.Int
func DeserializeUint64(bz []byte) (math.Int, error) {
	val, err := vmtypes.DeserializeUint64(bz)
	if err != nil {
		return math.ZeroInt(), err
	}

	num := math.NewIntFromUint64(val)
	return num, nil
}

// DeserializeDecimal deserialize uint128 bytes to math.Int
func DeserializeDecimal(bz []byte) (math.LegacyDec, error) {
	num, err := DeserializeUint128(bz)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	// fractional part length is 18
	return math.LegacyNewDecFromInt(num).Mul(math.LegacyNewDecWithPrec(1, 18)), nil
}

// DeserializeUint128 deserialize uint128 bytes to math.Int
func DeserializeUint128(bz []byte) (math.Int, error) {
	high, low, err := vmtypes.DeserializeUint128(bz)
	if err != nil {
		return math.ZeroInt(), err
	}

	n := big.NewInt(0).SetUint64(high)
	n.Lsh(n, 64)
	n.Add(n, big.NewInt(0).SetUint64(low))
	num := math.NewIntFromBigInt(n)
	return num, nil
}

// StructTagToTypeTag convert struct tag to type tag
func StructTagToTypeTag(structTag vmtypes.StructTag) vmtypes.TypeTag {
	return &vmtypes.TypeTag__Struct{
		Value: structTag,
	}
}

// TypeTagToStructTag converts coinType(TypeTag) to denom
func TypeTagToStructTag(coinType vmtypes.TypeTag) (vmtypes.StructTag, error) {
	if structTag, ok := coinType.(*vmtypes.TypeTag__Struct); ok {
		return structTag.Value, nil
	}

	return vmtypes.StructTag{}, ErrMalformedStructTag
}

// convert UpgradePolicy to vm UpgradePolicy
func (policy UpgradePolicy) ToVmUpgradePolicy() uint8 {
	// 0 => Arbitrary
	// 1 => Compatible
	// 2 => Immutable
	return uint8(policy)
}

func ReadTableHandleFromTable(bz []byte) (vmtypes.AccountAddress, error) {
	return vmtypes.NewAccountAddressFromBytes(bz[:AddressBytesLength])
}

// ReadSymbolFromMetadata util function to read symbol from Metadata
func ReadSymbolFromMetadata(bz []byte) string {
	cursor := int(0)

	// read name
	nameLen, len := readULEB128(bz[cursor:])
	cursor += (nameLen + len)

	// read symbol length
	symbolLen, len := readULEB128(bz[cursor:])
	cursor += len

	// read symbol
	symbol := string(bz[cursor : cursor+symbolLen])
	return symbol
}

// ReadSupplyFromSupply util function to read supply from Supply
func ReadSupplyFromSupply(bz []byte) (math.Int, error) {
	cursor := int(0)

	supplyBz := bz[cursor : cursor+16]
	num, err := DeserializeUint128(supplyBz)
	if err != nil {
		return math.ZeroInt(), err
	}

	return num, nil
}

// ReadBalanceFromFungibleStore util function to read balance from FungibleStore
func ReadBalanceFromFungibleStore(bz []byte) (vmtypes.AccountAddress, math.Int, error) {
	cursor := int(0)

	// read metadata object
	metadata, err := vmtypes.NewAccountAddressFromBytes(bz[cursor : cursor+AddressBytesLength])
	if err != nil {
		return vmtypes.AccountAddress{}, math.ZeroInt(), err
	}
	cursor += AddressBytesLength

	// read balance
	amount, err := vmtypes.DeserializeUint64(bz[cursor : cursor+8])
	if err != nil {
		return vmtypes.AccountAddress{}, math.ZeroInt(), err
	}

	return metadata, math.NewIntFromUint64(amount), nil
}

// ReadIssuersTableHandleFromModuleStore util function to read issuers table handle from primary_fungible_store::ModuleStore
func ReadIssuersTableHandleFromModuleStore(bz []byte) (vmtypes.AccountAddress, error) {
	cursor := int(0)

	// read issuers table
	issuersTable := bz[cursor : cursor+AddressBytesLength+8]
	return ReadTableHandleFromTable(issuersTable)
}

// ReadUserStoresTableHandleFromModuleStore util function to read user_stores table handle from primary_fungible_store::ModuleStore
func ReadUserStoresTableHandleFromModuleStore(bz []byte) (vmtypes.AccountAddress, error) {
	cursor := int(0)

	// read issuers table
	cursor += AddressBytesLength + 8

	// read user stores table
	userStoresTable := bz[cursor : cursor+AddressBytesLength+8]
	return ReadTableHandleFromTable(userStoresTable)
}

// ReadWeightsFromDexConfig util function to read pool balances from the DexConfig
func ReadWeightsFromDexConfig(timestamp math.Int, bz []byte) (math.LegacyDec, math.LegacyDec, error) {
	cursor := int(0)

	// read extend_ref
	cursor += AddressBytesLength + 8

	// before weights
	weightCoinABefore, err := DeserializeDecimal(bz[cursor : cursor+16])
	if err != nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), err
	}
	weightCoinBBefore, err := DeserializeDecimal(bz[cursor+16 : cursor+32])
	if err != nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), err
	}
	timestampBefore, err := DeserializeUint64(bz[cursor+32 : cursor+40])
	if err != nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), err
	}

	cursor += 40

	// after weights
	weightCoinAAfter, err := DeserializeDecimal(bz[cursor : cursor+16])
	if err != nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), err
	}
	weightCoinBAfter, err := DeserializeDecimal(bz[cursor+16 : cursor+32])
	if err != nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), err
	}
	timestampAfter, err := DeserializeUint64(bz[cursor+32 : cursor+40])
	if err != nil {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), err
	}

	return GetPoolWeights(
		weightCoinABefore,
		weightCoinBBefore,
		weightCoinAAfter,
		weightCoinBAfter,
		timestampBefore,
		timestampAfter,
		timestamp,
	)
}

// ReadStoresFromPool util function to read pool stores from the Pool
func ReadStoresFromPool(bz []byte) (vmtypes.AccountAddress, vmtypes.AccountAddress, error) {
	cursor := int(0)

	storeA := bz[cursor : cursor+AddressBytesLength]
	cursor += AddressBytesLength

	storeAAddr, err := vmtypes.NewAccountAddressFromBytes(storeA)
	if err != nil {
		return vmtypes.AccountAddress{}, vmtypes.AccountAddress{}, err
	}

	storeB := bz[cursor : cursor+AddressBytesLength]
	cursor += AddressBytesLength //nolint

	storeBAddr, err := vmtypes.NewAccountAddressFromBytes(storeB)
	if err != nil {
		return vmtypes.AccountAddress{}, vmtypes.AccountAddress{}, err
	}

	return storeAAddr, storeBAddr, nil
}

// GetDexWeight conduct same calculation with `get_weight` of dex contract
func GetPoolWeights(
	weightCoinABefore, weightCoinBBefore, weightCoinAAfter, weightCoinBAfter math.LegacyDec,
	timestampBefore, timestampAfter, timestamp math.Int,
) (math.LegacyDec, math.LegacyDec, error) {
	if timestampBefore.GT(timestamp) {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), ErrInvalidDexConfig
	}

	if timestamp.GTE(timestampAfter) {
		return weightCoinAAfter, weightCoinBAfter, nil
	}

	interval := timestampAfter.Sub(timestampBefore)
	timeDiffBefore := timestamp.Sub(timestampBefore)
	timeDiffAfter := timestampAfter.Sub(timestamp)

	weightCoinA := weightCoinAAfter.MulInt(timeDiffBefore).Add(weightCoinABefore.MulInt(timeDiffAfter)).QuoInt(interval)
	weightCoinB := weightCoinBAfter.MulInt(timeDiffBefore).Add(weightCoinBBefore.MulInt(timeDiffAfter)).QuoInt(interval)
	return weightCoinA, weightCoinB, nil
}

// GetPoolSpotPrice return quote price in base unit
func GetPoolSpotPrice(
	balanceBase, balanceQuote math.Int,
	weightBase, weightQuote math.LegacyDec,
) math.LegacyDec {
	numerator := weightQuote.MulInt(balanceBase)
	denominator := weightBase.MulInt(balanceQuote)

	return numerator.Quo(denominator)
}

// ReadUnbondingInfosFromStakingState util function to read unbonding coin amount from the StakingState
func ReadUnbondingInfosFromStakingState(bz []byte) (unbondingShare math.Int, unbondingCoinStore vmtypes.AccountAddress, err error) {
	cursor := int(0)

	// read metadata
	cursor += AddressBytesLength

	// read validator
	valLen, len := readULEB128(bz[cursor:])
	cursor += (valLen + len)

	// read total_share
	cursor += 16

	// read unbonding_share
	unbondingShare, err = DeserializeUint128(bz[cursor : cursor+16])
	if err != nil {
		return
	}

	cursor += 16

	// read reward_index(Decimal128)
	cursor += 16

	// read reward_coin_store_ref(ExtendRef)
	cursor += AddressBytesLength + 8

	// read unbonding_coin_store_ref(ExtendRef)
	cursor += AddressBytesLength + 8

	// read reward_coin_store
	cursor += AddressBytesLength

	// read unbonding_coin_store
	unbondingCoinStore, err = vmtypes.NewAccountAddressFromBytes(bz[cursor : cursor+AddressBytesLength])
	if err != nil {
		return
	}

	return
}

// ReadCollectionInfo util function to read collection info from the raw bytes (bcs)
func ReadCollectionInfo(bz []byte) (
	creator vmtypes.AccountAddress,
	name, uri, desc string,
	err error,
) {
	cursor := int(0)

	// read creator
	creator, err = vmtypes.NewAccountAddressFromBytes(bz[cursor : cursor+AddressBytesLength])
	if err != nil {
		return
	}

	cursor += AddressBytesLength

	// read desc
	descLen, len := readULEB128(bz[cursor:])
	cursor += len

	desc = string(bz[cursor : cursor+descLen])
	cursor += descLen

	// read name
	nameLen, len := readULEB128(bz[cursor:])
	cursor += len

	name = string(bz[cursor : cursor+nameLen])
	cursor += nameLen

	// read uri
	uriLen, len := readULEB128(bz[cursor:])
	cursor += len

	uri = string(bz[cursor : cursor+uriLen])
	cursor += uriLen //nolint

	return
}

// ReadNftInfo util function to read nft info from the raw bytes (bcs)
func ReadNftInfo(bz []byte) (tokenId, uri, desc string) {
	cursor := int(0)

	// read collection
	cursor += AddressBytesLength

	// read description
	descLen, len := readULEB128(bz[cursor:])
	cursor += len

	desc = string(bz[cursor : cursor+descLen])
	cursor += descLen

	// read tokenId
	tokenIdLen, len := readULEB128(bz[cursor:])
	cursor += len

	tokenId = string(bz[cursor : cursor+tokenIdLen])
	cursor += tokenIdLen

	// read uri
	uriLen, len := readULEB128(bz[cursor:])
	cursor += len

	uri = string(bz[cursor : cursor+uriLen])
	cursor += uriLen //nolint

	return
}

// ReadMetadataTableHandleFromMetadataStore util function to read metadata table handle from the MetadataStore raw bytes
func ReadMetadataTableHandleFromMetadataStore(bz []byte) (tableHandle vmtypes.AccountAddress, err error) {
	cursor := int(0)

	return ReadTableHandleFromTable(bz[cursor : cursor+AddressBytesLength+8])
}

func ReadUpgradePolicyFromModuleMetadata(bz []byte) (UpgradePolicy, error) {
	cursor := int(0)
	upgradePolicy := int8(bz[cursor])

	return UpgradePolicy(upgradePolicy), nil
}

func ReadStakingStatesTableHandleFromModuleStore(bz []byte) (vmtypes.AccountAddress, error) {
	cursor := int(0)

	return ReadTableHandleFromTable(bz[cursor : cursor+AddressBytesLength+8])
}

func ReadCodeModuleStore(bz []byte) (bool, []vmtypes.AccountAddress, error) {
	cursor := int(0)

	allowArbitrary := bz[cursor] == 1
	cursor += 1

	addrsLen, len := readULEB128(bz[cursor:])
	cursor += len

	allowedPublishers := make([]vmtypes.AccountAddress, addrsLen)
	for i := 0; i < addrsLen; i++ {
		var err error
		allowedPublishers[i], err = vmtypes.NewAccountAddressFromBytes(bz[cursor : cursor+AddressBytesLength])
		if err != nil {
			return false, nil, err
		}

		cursor += AddressBytesLength
	}

	return allowArbitrary, allowedPublishers, nil
}

// readULEB128 converts a uleb128-encoded byte array into an int.
func readULEB128(r []byte) (total int, len int) {
	var shift uint64

	for {
		b := r[len]
		len++
		total |= (int(b&0x7F) << shift)
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}

	return
}
