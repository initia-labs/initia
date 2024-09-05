package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	govtypes "github.com/initia-labs/initia/x/gov/types"
	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	stakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"

	vmtypes "github.com/initia-labs/movevm/types"

	slinkytypes "github.com/skip-mev/slinky/pkg/types"
	oracletypes "github.com/skip-mev/slinky/x/oracle/types"
)

func Test_GetAccountInfo(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	vmaddr, err := vmtypes.NewAccountAddressFromBytes(addrs[0])
	require.NoError(t, err)

	api := keeper.NewApi(input.MoveKeeper, ctx)

	// base account
	input.AccountKeeper.SetAccount(ctx, input.AccountKeeper.NewAccountWithAddress(ctx, addrs[0]))
	found, accountNumber, sequence, accountType, isBlocked := api.GetAccountInfo(vmaddr)
	require.True(t, found)
	require.False(t, isBlocked)

	acc := input.AccountKeeper.GetAccount(ctx, addrs[0])
	require.Equal(t, acc.GetAccountNumber(), accountNumber)
	require.Equal(t, acc.GetSequence(), sequence)
	require.Equal(t, vmtypes.AccountType_Base, accountType)

	// module account
	govAcc := input.AccountKeeper.GetModuleAccount(ctx, "gov")
	govAddr := govAcc.GetAddress()
	govVmAddr, err := vmtypes.NewAccountAddressFromBytes(govAddr.Bytes())
	require.NoError(t, err)
	found, accountNumber, sequence, accountType, isBlocked = api.GetAccountInfo(govVmAddr)
	require.True(t, found)
	require.True(t, isBlocked)

	acc = input.AccountKeeper.GetAccount(ctx, govAddr)
	require.Equal(t, acc.GetAccountNumber(), accountNumber)
	require.Equal(t, acc.GetSequence(), sequence)
	require.Equal(t, vmtypes.AccountType_Module, accountType)

	// not found
	vmaddr, err = vmtypes.NewAccountAddress("0x3")
	require.NoError(t, err)

	found, _, _, _, isBlocked = api.GetAccountInfo(vmaddr)
	require.False(t, found)
	require.False(t, isBlocked)
}

func Test_CreateTypedAccounts(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	vmaddr, err := vmtypes.NewAccountAddressFromBytes(addrs[0])
	require.NoError(t, err)

	api := keeper.NewApi(input.MoveKeeper, ctx)

	input.AccountKeeper.SetAccount(ctx, input.AccountKeeper.NewAccountWithAddress(ctx, addrs[0]))
	found, accountNumber, sequence, accountType, _ := api.GetAccountInfo(vmaddr)
	require.True(t, found)

	acc := input.AccountKeeper.GetAccount(ctx, addrs[0])
	require.Equal(t, acc.GetAccountNumber(), accountNumber)
	require.Equal(t, acc.GetSequence(), sequence)
	require.Equal(t, vmtypes.AccountType_Base, accountType)

	vmaddr, err = vmtypes.NewAccountAddress("0x3")
	require.NoError(t, err)

	found, _, _, _, _ = api.GetAccountInfo(vmaddr)
	require.False(t, found)
}

func Test_AmountToShareAPI(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := valAddrs[0]
	valPubKey := valPubKeys[0]

	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddrs[0])
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))

	sh := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err = sh.CreateValidator(ctx, newTestMsgCreateValidator(valAddrStr, valPubKey, math.NewInt(100_000)))
	require.NoError(t, err)

	metadata, err := types.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	api := keeper.NewApi(input.MoveKeeper, ctx)
	amount, err := api.AmountToShare([]byte(valAddr.String()), metadata, 150)
	require.NoError(t, err)
	require.Equal(t, "150.000000000000000000", amount)
}

func Test_AmountToShareAPI_InvalidAddr(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := valAddrs[0]
	valPubKey := valPubKeys[0]

	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddrs[0])
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))

	sh := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err = sh.CreateValidator(ctx, newTestMsgCreateValidator(valAddrStr, valPubKey, math.NewInt(100_000)))
	require.NoError(t, err)

	metadata, err := types.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	api := keeper.NewApi(input.MoveKeeper, ctx)
	_, err = api.AmountToShare(valAddr, metadata, 150)
	require.Error(t, err)
}

func Test_ShareToAmountAPI(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := valAddrs[0]
	valPubKey := valPubKeys[0]

	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddrs[0])
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))

	sh := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err = sh.CreateValidator(ctx, newTestMsgCreateValidator(valAddrStr, valPubKey, math.NewInt(100_000)))
	require.NoError(t, err)

	metadata, err := types.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	api := keeper.NewApi(input.MoveKeeper, ctx)
	amount, err := api.ShareToAmount([]byte(valAddr.String()), metadata, "150")
	require.NoError(t, err)
	require.Equal(t, uint64(150), amount)
}

func Test_ShareToAmountAPI_InvalidAddr(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := valAddrs[0]
	valPubKey := valPubKeys[0]

	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddrs[0])
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, math.NewInt(100_000_000)))

	sh := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err = sh.CreateValidator(ctx, newTestMsgCreateValidator(valAddrStr, valPubKey, math.NewInt(100_000)))
	require.NoError(t, err)

	metadata, err := types.MetadataAddressFromDenom(bondDenom)
	require.NoError(t, err)

	api := keeper.NewApi(input.MoveKeeper, ctx)
	_, err = api.ShareToAmount(valAddr, metadata, "150")
	require.Error(t, err)
}

func Test_UnbondTimestamp(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// set UnbondingTime
	stakingParams, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)

	stakingParams.UnbondingTime = time.Second * 60 * 60 * 24 * 7
	input.StakingKeeper.SetParams(ctx, stakingParams)

	now := time.Now()
	api := keeper.NewApi(input.MoveKeeper, ctx.WithBlockTime(now))

	resTimestamp := api.UnbondTimestamp()
	require.Equal(t, uint64(now.Unix()+60*60*24*7), resTimestamp)
}

func Test_GetPrice(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	pairId := "BITCOIN/USD"
	cp, err := slinkytypes.CurrencyPairFromString(pairId)
	require.NoError(t, err)

	price := math.NewInt(111111).MulRaw(1_000_000_000).MulRaw(1_000_000_000).MulRaw(1_000_000_000)
	now := time.Now()

	err = input.OracleKeeper.SetPriceForCurrencyPair(ctx, cp, oracletypes.QuotePrice{
		Price:          price,
		BlockTimestamp: now,
		BlockHeight:    100,
	})
	require.NoError(t, err)

	pairIdArg, err := vmtypes.SerializeString(pairId)
	require.NoError(t, err)

	res, _, err := input.MoveKeeper.ExecuteViewFunction(
		ctx,
		vmtypes.StdAddress,
		"oracle",
		"get_price",
		[]vmtypes.TypeTag{},
		[][]byte{pairIdArg},
	)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("[\"%s\",\"%d\",\"%d\"]", price.String(), now.Unix(), cp.LegacyDecimals()), res.Ret)
}

func Test_API_Query(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	proposal := govtypes.Proposal{
		Id:      1,
		Title:   "title",
		Summary: "summary",
		Status:  govtypesv1.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD,
	}

	// set Proposal
	err := input.GovKeeper.SetProposal(ctx, proposal)
	require.NoError(t, err)

	now := time.Now()
	api := keeper.NewApi(input.MoveKeeper, ctx.WithBlockTime(now))

	// out of gas
	require.Panics(t, func() {
		_, _, _ = api.Query(vmtypes.QueryRequest{
			Stargate: &vmtypes.StargateQuery{
				Path: "/initia.gov.v1.Query/Proposal",
				Data: []byte(`{"proposal_id": "1"}`),
			},
		}, 100)
	})

	// valid query
	gasBalance := uint64(2000)
	resBz, gasUsed, err := api.Query(vmtypes.QueryRequest{
		Stargate: &vmtypes.StargateQuery{
			Path: "/initia.gov.v1.Query/Proposal",
			Data: []byte(`{"proposal_id": "1"}`),
		},
	}, gasBalance)
	require.NoError(t, err)
	require.Greater(t, gasBalance, gasUsed)

	// expected proposal res json bytes
	expectedResBz, err := input.EncodingConfig.Codec.MarshalJSON(&govtypes.QueryProposalResponse{
		Proposal: &proposal,
	})
	require.NoError(t, err)
	require.Equal(t, expectedResBz, resBz)
}

func Test_API_CustomQuery(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	err := input.MoveKeeper.PublishModuleBundle(ctx, vmtypes.TestAddress, vmtypes.NewModuleBundle(vmtypes.NewModule(testAddressModule)), types.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	vmAddr, err := vmtypes.NewAccountAddressFromBytes(addrs[0])
	require.NoError(t, err)

	// to sdk
	res, _, err := input.MoveKeeper.ExecuteViewFunction(ctx, vmtypes.TestAddress, "TestAddress", "to_sdk", []vmtypes.TypeTag{}, [][]byte{vmAddr.Bytes()})
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("\"%s\"", addrs[0].String()), res.Ret)

	// from sdk
	inputBz, err := vmtypes.SerializeString(addrs[0].String())
	require.NoError(t, err)
	res, _, err = input.MoveKeeper.ExecuteViewFunction(ctx, vmtypes.TestAddress, "TestAddress", "from_sdk", []vmtypes.TypeTag{}, [][]byte{inputBz})
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("\"%s\"", vmAddr.String()), res.Ret)
}
