package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"
)

func TestGetModule(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	err := input.MoveKeeper.PublishModuleBundle(ctx,
		vmtypes.StdAddress,
		vmtypes.NewModuleBundle(vmtypes.NewModule(basicCoinModule)),
		types.UpgradePolicy_IMMUTABLE,
	)
	require.NoError(t, err)

	module, err := input.MoveKeeper.GetModule(ctx, vmtypes.StdAddress, "BasicCoin")
	require.NoError(t, err)
	bz, err := input.MoveKeeper.DecodeModuleBytes(basicCoinModule)
	require.NoError(t, err)

	require.Equal(t, types.Module{
		Address:       vmtypes.StdAddress.String(),
		ModuleName:    "BasicCoin",
		Abi:           string(bz),
		RawBytes:      basicCoinModule,
		UpgradePolicy: types.UpgradePolicy_IMMUTABLE,
	}, module)
}

func TestSetModule(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.MoveKeeper.SetModule(ctx, vmtypes.StdAddress, "BasicCoin", basicCoinModule)
	module, err := input.MoveKeeper.GetModule(ctx, vmtypes.StdAddress, "BasicCoin")
	require.NoError(t, err)

	bz, err := input.MoveKeeper.DecodeModuleBytes(basicCoinModule)
	require.NoError(t, err)
	require.Equal(t, types.Module{
		Address:       "0x1",
		ModuleName:    "BasicCoin",
		Abi:           string(bz),
		RawBytes:      basicCoinModule,
		UpgradePolicy: types.UpgradePolicy_COMPATIBLE,
	}, module)
}

func TestGetAndSetChecksum(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	err := input.MoveKeeper.PublishModuleBundle(ctx,
		vmtypes.StdAddress,
		vmtypes.NewModuleBundle(vmtypes.NewModule(basicCoinModule)),
		types.UpgradePolicy_IMMUTABLE,
	)
	require.NoError(t, err)

	basicCoinChecksum := types.ModuleBzToChecksum(basicCoinModule)

	checksum, err := input.MoveKeeper.GetChecksum(ctx, vmtypes.StdAddress, "BasicCoin")
	require.NoError(t, err)

	require.Equal(t, types.Checksum{
		Address:    vmtypes.StdAddress.String(),
		ModuleName: "BasicCoin",
		Checksum:   basicCoinChecksum[:],
	}, checksum)
}

func TestGetAndSetResource(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	testDenom := testDenoms[0]
	testDenomMetadata, err := types.MetadataAddressFromDenom(testDenom)
	require.NoError(t, err)

	structTagStr := "0x1::fungible_asset::FungibleStore"
	structTag, err := vmapi.ParseStructTag(structTagStr)
	require.NoError(t, err)

	var data []byte

	// metadata
	data = append(data, testDenomMetadata[:]...)

	// balance
	bz, err := vmtypes.SerializeUint64(100)
	require.NoError(t, err)
	data = append(data, bz...)

	// frozen
	bz, err = vmtypes.SerializeBool(false)
	require.NoError(t, err)
	data = append(data, bz...)

	input.MoveKeeper.SetResource(ctx, vmtypes.TestAddress, structTag, data)
	resource, err := input.MoveKeeper.GetResource(ctx, vmtypes.TestAddress, structTag)
	require.NoError(t, err)

	require.Equal(t, types.Resource{
		Address:      vmtypes.TestAddress.String(),
		StructTag:    structTagStr,
		MoveResource: fmt.Sprintf(`{"type":"0x1::fungible_asset::FungibleStore","data":{"balance":"100","frozen":false,"metadata":{"inner":"%s"}}}`, testDenomMetadata.String()),
		RawBytes:     data,
	}, resource)
}

func TestGetTableEntry(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// publish test module
	err := input.MoveKeeper.PublishModuleBundle(ctx, vmtypes.TestAddress, vmtypes.NewModuleBundle(vmtypes.NewModule(tableGeneratorModule)), types.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	threeBz, err := vmtypes.SerializeUint64(3)
	require.NoError(t, err)

	// create table entries
	err = input.MoveKeeper.ExecuteEntryFunction(
		ctx,
		vmtypes.TestAddress,
		vmtypes.TestAddress,
		"TableGenerator",
		"generate_table",
		[]vmtypes.TypeTag{},
		[][]byte{threeBz})
	require.NoError(t, err)

	s, err := vmapi.ParseStructTag("0x2::TableGenerator::S<u64, u64>")
	require.NoError(t, err)

	resourceBz, err := input.MoveKeeper.GetResourceBytes(ctx, vmtypes.TestAddress, s)
	require.NoError(t, err)

	tableAddr, err := types.ReadTableHandleFromTable(resourceBz)
	require.NoError(t, err)

	oneBz, err := vmtypes.SerializeUint64(1)
	require.NoError(t, err)

	info, err := input.MoveKeeper.GetTableInfo(ctx, tableAddr)
	require.NoError(t, err)
	require.Equal(t, types.TableInfo{
		Address:   tableAddr.String(),
		KeyType:   "u64",
		ValueType: "u64",
	}, info)

	entry, err := input.MoveKeeper.GetTableEntry(ctx, tableAddr, oneBz)
	require.NoError(t, err)
	require.Equal(t, types.TableEntry{
		Address:    tableAddr.String(),
		Key:        "\"1\"",
		Value:      "\"1\"",
		KeyBytes:   oneBz,
		ValueBytes: oneBz,
	}, entry)
}

func TestIterateVMStore(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.MoveKeeper.SetModule(ctx, vmtypes.StdAddress, "BasicCoin", basicCoinModule)

	structTagStr := "0x1::BasicCoin::Coin<0x1::BasicCoin::Initia>"
	structTag, err := vmapi.ParseStructTag(structTagStr)
	require.NoError(t, err)

	data, err := vmtypes.SerializeUint64(100)
	require.NoError(t, err)

	basicCoinChecksum := types.ModuleBzToChecksum(basicCoinModule)
	input.MoveKeeper.SetChecksum(ctx, vmtypes.StdAddress, "BasicCoin", basicCoinChecksum[:])

	input.MoveKeeper.SetResource(ctx, vmtypes.TestAddress, structTag, data)
	input.MoveKeeper.SetTableInfo(ctx, types.TableInfo{
		Address:   vmtypes.TestAddress.String(),
		KeyType:   "u64",
		ValueType: "u64",
	})
	input.MoveKeeper.SetTableEntry(ctx, types.TableEntry{
		Address:    vmtypes.TestAddress.String(),
		KeyBytes:   []byte{1, 2, 3},
		ValueBytes: []byte{4, 5, 6},
	})
	input.MoveKeeper.IterateVMStore(ctx, func(module *types.Module, checksum *types.Checksum, resource *types.Resource, tableInfo *types.TableInfo, tableEntry *types.TableEntry) {
		if module != nil && module.ModuleName == "BasicCoin" {
			require.Equal(t, types.Module{
				Address:       "0x1",
				ModuleName:    "BasicCoin",
				RawBytes:      basicCoinModule,
				UpgradePolicy: types.UpgradePolicy_COMPATIBLE,
			}, *module)
		}

		if checksum != nil && checksum.ModuleName == "BasicCoin" {
			require.Equal(t, types.Checksum{
				Address:    "0x1",
				ModuleName: "BasicCoin",
				Checksum:   basicCoinChecksum[:],
			}, *checksum)
		}

		if resource != nil && resource.Address == "0x2" {
			require.Equal(t, types.Resource{
				Address:   "0x2",
				StructTag: structTagStr,
				RawBytes:  data,
			}, *resource)
		}

		if tableInfo != nil && tableInfo.Address == "0x2" {
			require.Equal(t, types.TableInfo{
				Address:   vmtypes.TestAddress.String(),
				KeyType:   "u64",
				ValueType: "u64",
			}, *tableInfo)
		}

		if tableEntry != nil && tableEntry.Address == "0x2" {
			require.Equal(t, types.TableEntry{
				Address:    vmtypes.TestAddress.String(),
				KeyBytes:   []byte{1, 2, 3},
				ValueBytes: []byte{4, 5, 6},
			}, *tableEntry)
		}
	})
}
