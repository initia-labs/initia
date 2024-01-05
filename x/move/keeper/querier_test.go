package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

func TestViewFunction(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.Faucet.Fund(
		ctx,
		types.TestAddr,
		sdk.NewCoin(bondDenom, math.NewInt(1000000)),
	)

	argBz, err := vmtypes.SerializeUint64(100)
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunction(
		ctx,
		vmtypes.TestAddress,
		vmtypes.StdAddress,
		"BasicCoin",
		"mint",
		[]vmtypes.TypeTag{MustConvertStringToTypeTag("0x1::BasicCoin::Initia")},
		[][]byte{argBz},
	)
	require.NoError(t, err)

	querier := keeper.NewQuerier(&input.MoveKeeper)

	res, err := querier.ViewFunction(
		ctx,
		&types.QueryViewFunctionRequest{
			Address:      vmtypes.StdAddress.String(),
			ModuleName:   "BasicCoin",
			FunctionName: "get",
			TypeArgs:     []string{"0x1::BasicCoin::Initia"},
			Args:         [][]byte{vmtypes.TestAddress.Bytes()},
		})
	require.NoError(t, err)
	require.Equal(t, res.Data, "\"100\"")
}

func TestModules(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	err := input.MoveKeeper.PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(vmtypes.NewModule(basicCoinModule)), types.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	querier := keeper.NewQuerier(&input.MoveKeeper)

	moduleRes, err := querier.Module(
		ctx,
		&types.QueryModuleRequest{
			Address:    vmtypes.StdAddress.String(),
			ModuleName: "BasicCoin",
		},
	)
	require.NoError(t, err)
	require.Equal(t, basicCoinModuleAbi, moduleRes.Module.Abi)
	require.Equal(t, moduleRes.Module, types.Module{
		Address:       "0x1",
		ModuleName:    "BasicCoin",
		Abi:           basicCoinModuleAbi,
		RawBytes:      basicCoinModule,
		UpgradePolicy: types.UpgradePolicy_COMPATIBLE,
	})

	modulesRes, err := querier.Modules(
		ctx,
		&types.QueryModulesRequest{
			Address: vmtypes.StdAddress.String(),
			Pagination: &query.PageRequest{
				Limit: 100,
			},
		},
	)

	require.NoError(t, err)
	require.Contains(t, modulesRes.Modules, types.Module{
		Address:       "0x1",
		ModuleName:    "BasicCoin",
		Abi:           basicCoinModuleAbi,
		RawBytes:      basicCoinModule,
		UpgradePolicy: types.UpgradePolicy_COMPATIBLE,
	})
}

func TestResources(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.Faucet.Fund(ctx, types.TestAddr, sdk.NewCoin(bondDenom, math.NewInt(1000000)))

	argBz, err := vmtypes.SerializeUint64(100)
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunction(ctx, vmtypes.TestAddress, vmtypes.StdAddress,
		"BasicCoin",
		"mint",
		[]vmtypes.TypeTag{MustConvertStringToTypeTag("0x1::BasicCoin::Initia")},
		[][]byte{argBz})
	require.NoError(t, err)

	querier := keeper.NewQuerier(&input.MoveKeeper)

	resourceRes, err := querier.Resource(
		ctx,
		&types.QueryResourceRequest{
			Address:   vmtypes.TestAddress.String(),
			StructTag: "0x1::BasicCoin::Coin<0x1::BasicCoin::Initia>",
		},
	)
	require.NoError(t, err)
	require.Equal(t, types.Resource{
		Address:      "0x2",
		StructTag:    "0x1::BasicCoin::Coin<0x1::BasicCoin::Initia>",
		MoveResource: `{"type":"0x1::BasicCoin::Coin<0x1::BasicCoin::Initia>","data":{"test":true,"value":"100"}}`,
		RawBytes:     append(argBz, 1),
	}, resourceRes.Resource)

	resourcesRes, err := querier.Resources(
		ctx,
		&types.QueryResourcesRequest{
			Address: vmtypes.TestAddress.String(),
		},
	)
	require.NoError(t, err)
	require.Contains(t, resourcesRes.Resources, types.Resource{
		Address:      "0x2",
		StructTag:    "0x1::BasicCoin::Coin<0x1::BasicCoin::Initia>",
		MoveResource: `{"type":"0x1::BasicCoin::Coin<0x1::BasicCoin::Initia>","data":{"test":true,"value":"100"}}`,
		RawBytes:     append(argBz, 1),
	})
}

func TestTableInfo(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	ac := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	twoAddr, err := types.AccAddressFromString(ac, "0x2")
	require.NoError(t, err)

	err = input.MoveKeeper.PublishModuleBundle(ctx, twoAddr,
		vmtypes.NewModuleBundle(vmtypes.NewModule(tableGeneratorModule)),
		types.UpgradePolicy_COMPATIBLE,
	)
	require.NoError(t, err)

	argBz, err := vmtypes.SerializeUint64(4)
	require.NoError(t, err)

	// 1:1, 2:2, 3:3, 4:4 table
	err = input.MoveKeeper.ExecuteEntryFunction(
		ctx,
		twoAddr,
		twoAddr,
		"TableGenerator",
		"generate_table",
		[]vmtypes.TypeTag{},
		[][]byte{argBz},
	)
	require.NoError(t, err)

	querier := keeper.NewQuerier(&input.MoveKeeper)
	resource, err := querier.Resource(
		ctx,
		&types.QueryResourceRequest{
			Address:   vmtypes.TestAddress.String(),
			StructTag: "0x2::TableGenerator::S<u64,u64>",
		},
	)
	require.NoError(t, err)

	tableAddrBz := resource.RawBytes[0:types.AddressBytesLength]
	tableAddr, err := vmtypes.NewAccountAddressFromBytes(tableAddrBz)
	require.NoError(t, err)

	infoRes, err := querier.TableInfo(
		ctx,
		&types.QueryTableInfoRequest{
			Address: tableAddr.String(),
		},
	)
	require.NoError(t, err)
	require.Equal(t, types.TableInfo{
		Address:   tableAddr.String(),
		KeyType:   "u64",
		ValueType: "u64",
	}, infoRes.TableInfo)
}

func TestTableEntries(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	ac := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	twoAddr, err := types.AccAddressFromString(ac, "0x2")
	require.NoError(t, err)

	err = input.MoveKeeper.PublishModuleBundle(ctx, twoAddr,
		vmtypes.NewModuleBundle(vmtypes.NewModule(tableGeneratorModule)),
		types.UpgradePolicy_COMPATIBLE,
	)
	require.NoError(t, err)

	argBz, err := vmtypes.SerializeUint64(4)
	require.NoError(t, err)

	// 1:1, 2:2, 3:3, 4:4 table
	err = input.MoveKeeper.ExecuteEntryFunction(
		ctx,
		twoAddr,
		twoAddr,
		"TableGenerator",
		"generate_table",
		[]vmtypes.TypeTag{},
		[][]byte{argBz},
	)
	require.NoError(t, err)

	querier := keeper.NewQuerier(&input.MoveKeeper)
	resource, err := querier.Resource(
		ctx,
		&types.QueryResourceRequest{
			Address:   vmtypes.TestAddress.String(),
			StructTag: "0x2::TableGenerator::S<u64,u64>",
		},
	)
	require.NoError(t, err)

	tableAddrBz := resource.RawBytes[0:types.AddressBytesLength]
	tableAddr, err := vmtypes.NewAccountAddressFromBytes(tableAddrBz)
	require.NoError(t, err)

	zeroBz, err := vmtypes.SerializeUint64(0)
	require.NoError(t, err)
	oneBz, err := vmtypes.SerializeUint64(1)
	require.NoError(t, err)
	twoBz, err := vmtypes.SerializeUint64(2)
	require.NoError(t, err)
	thirdBz, err := vmtypes.SerializeUint64(3)
	require.NoError(t, err)

	entryRes, err := querier.TableEntry(
		ctx,
		&types.QueryTableEntryRequest{
			Address:  tableAddr.String(),
			KeyBytes: oneBz,
		},
	)
	require.NoError(t, err)
	require.Equal(t, types.TableEntry{
		Address:    tableAddr.String(),
		Key:        "\"1\"",
		Value:      "\"1\"",
		KeyBytes:   oneBz,
		ValueBytes: oneBz,
	}, entryRes.TableEntry)

	entriesRes, err := querier.TableEntries(
		ctx,
		&types.QueryTableEntriesRequest{
			Address: tableAddr.String(),
		},
	)
	require.NoError(t, err)
	require.Equal(t, []types.TableEntry{{
		Address:    tableAddr.String(),
		Key:        "\"0\"",
		Value:      "\"0\"",
		KeyBytes:   zeroBz,
		ValueBytes: zeroBz,
	},
		{
			Address:    tableAddr.String(),
			Key:        "\"1\"",
			Value:      "\"1\"",
			KeyBytes:   oneBz,
			ValueBytes: oneBz,
		},
		{
			Address:    tableAddr.String(),
			Key:        "\"2\"",
			Value:      "\"2\"",
			KeyBytes:   twoBz,
			ValueBytes: twoBz,
		},
		{
			Address:    tableAddr.String(),
			Key:        "\"3\"",
			Value:      "\"3\"",
			KeyBytes:   thirdBz,
			ValueBytes: thirdBz,
		}}, entriesRes.TableEntries)

}

func TestScriptABI(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	querier := keeper.NewQuerier(&input.MoveKeeper)
	abi, err := querier.ScriptABI(
		ctx,
		&types.QueryScriptABIRequest{
			CodeBytes: basicCoinMintScript,
		},
	)
	require.NoError(t, err)
	expectedABI := `{"name":"main","visibility":"public","is_entry":true,"is_view":false,"generic_type_params":[{"constraints":[]},{"constraints":[]}],"params":["signer"],"return":[]}`
	require.Equal(t, []byte(expectedABI), abi.Abi)
}

func TestParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	querier := keeper.NewQuerier(&input.MoveKeeper)
	params, err := querier.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)

	expectedParams, err := input.MoveKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, expectedParams, params.Params)
}
