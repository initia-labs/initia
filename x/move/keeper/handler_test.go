package keeper_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/initia-labs/initia/x/move/ante"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/initiavm/types"
)

func TestPublishModuleBundle(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	err := input.MoveKeeper.PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(vmtypes.NewModule(basicCoinModule)), types.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	// republish to update upgrade policy to immutable
	err = input.MoveKeeper.PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(vmtypes.NewModule(basicCoinModule)), types.UpgradePolicy_IMMUTABLE)
	require.NoError(t, err)

	// republish not allowed
	err = input.MoveKeeper.PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(vmtypes.NewModule(basicCoinModule)), types.UpgradePolicy_ARBITRARY)
	require.Error(t, err)
}

func TestPublishModuleBundle_ArbitraryNotEnabled(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.MoveKeeper.SetArbitraryEnabled(ctx, false)

	// arbitrary not allowed
	err := input.MoveKeeper.PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(vmtypes.NewModule(basicCoinModule)), types.UpgradePolicy_ARBITRARY)
	require.Error(t, err)

	err = input.MoveKeeper.PublishModuleBundle(ctx, vmtypes.StdAddress, vmtypes.NewModuleBundle(vmtypes.NewModule(basicCoinModule)), types.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)
}

func TestExecuteEntryFunction(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	argBz, err := vmtypes.SerializeUint64(100)
	require.NoError(t, err)
	err = input.MoveKeeper.ExecuteEntryFunction(ctx, vmtypes.TestAddress, vmtypes.StdAddress,
		"BasicCoin",
		"mint",
		[]vmtypes.TypeTag{MustConvertStringToTypeTag("0x1::BasicCoin::Initia")},
		[][]byte{argBz})
	require.NoError(t, err)

	events := ctx.EventManager().Events()
	event := events[len(events)-1]

	require.Equal(t, sdk.NewEvent(types.EventTypeMove,
		sdk.NewAttribute(types.AttributeKeyTypeTag, "0x1::BasicCoin::MintEvent"),
		sdk.NewAttribute(types.AttributeKeyData, `{"account":"0x2","amount":"100","coin_type":"0x1::BasicCoin::Initia"}`),
	), event)
}

func TestExecuteScript(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	bz, err := hex.DecodeString("0000000000000000000000000000000000000002")
	require.NoError(t, err)
	twoAddr := sdk.AccAddress(bz)

	err = input.MoveKeeper.ExecuteScript(ctx, twoAddr,
		basicCoinMintScript,
		[]vmtypes.TypeTag{MustConvertStringToTypeTag("0x1::BasicCoin::Initia"), MustConvertStringToTypeTag("bool")},
		[][]byte{},
	)
	require.NoError(t, err)
}

func TestDispatchDelegateMessage(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	delegatorAddr, err := vmtypes.NewAccountAddressFromBytes(addrs[0])
	require.NoError(t, err)

	delegator := addrs[0]
	validator := valAddrs[0]
	require.NoError(t, err)
	denom := bondDenom
	amount := sdk.NewInt(100)

	metadata, err := types.MetadataAddressFromDenom(denom)
	require.NoError(t, err)

	validatorBz, err := vmtypes.SerializeString(validator.String())
	require.NoError(t, err)

	amountBz, err := vmtypes.SerializeUint64(amount.Uint64())
	require.NoError(t, err)
	err = input.MoveKeeper.ExecuteEntryFunction(ctx, delegatorAddr, vmtypes.StdAddress,
		"cosmos",
		"delegate",
		[]vmtypes.TypeTag{},
		[][]byte{
			validatorBz,
			metadata[:],
			amountBz,
		})
	require.NoError(t, err)

	events := ctx.EventManager().Events()
	event := events[len(events)-1]

	require.Equal(t, sdk.NewEvent("delegate",
		sdk.NewAttribute("delegator_address", delegator.String()),
		sdk.NewAttribute("validator_address", validator.String()),
		sdk.NewAttribute("amount", sdk.NewCoins(sdk.NewCoin(denom, amount)).String()),
	), event)
}

func TestDispatchFundCommunityPoolMessage(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	depositor := addrs[0]
	depositorAddr, err := vmtypes.NewAccountAddressFromBytes(addrs[0])
	denom := bondDenom
	amount := sdk.NewInt(100)

	metadata, err := types.MetadataAddressFromDenom(denom)
	require.NoError(t, err)

	amountBz, err := vmtypes.SerializeUint64(amount.Uint64())
	require.NoError(t, err)
	err = input.MoveKeeper.ExecuteEntryFunction(ctx, depositorAddr, vmtypes.StdAddress,
		"cosmos",
		"fund_community_pool",
		[]vmtypes.TypeTag{},
		[][]byte{
			metadata[:],
			amountBz,
		})
	require.NoError(t, err)

	events := ctx.EventManager().Events()
	event := events[len(events)-1]

	require.Equal(t, sdk.NewEvent("fund_community_pool",
		sdk.NewAttribute("depositor_address", depositor.String()),
		sdk.NewAttribute("amount", sdk.NewCoins(sdk.NewCoin(denom, amount)).String()),
	), event)
}

func TestDispatchTransferMessage(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	sender := addrs[0]
	senderAddr, err := vmtypes.NewAccountAddressFromBytes(addrs[0])
	receiver := valAddrs[0]
	denom := bondDenom
	amount := sdk.NewInt(100)
	sourcePort := "port-1"
	sourceChannel := "channel-1"
	revisionNumber := uint64(1)
	revisionHeight := uint64(2)
	timeoutTimestamp := uint64(100)
	memo := "memo"

	metadata, err := types.MetadataAddressFromDenom(denom)
	require.NoError(t, err)

	receiverBz, err := vmtypes.SerializeString(receiver.String())
	require.NoError(t, err)

	amountBz, err := vmtypes.SerializeUint64(amount.Uint64())
	require.NoError(t, err)

	sourcePortBz, err := vmtypes.SerializeString(sourcePort)
	require.NoError(t, err)

	sourceChannelBz, err := vmtypes.SerializeString(sourceChannel)
	require.NoError(t, err)

	revisionNumberBz, err := vmtypes.SerializeUint64(revisionNumber)
	require.NoError(t, err)

	revisionHeightBz, err := vmtypes.SerializeUint64(revisionHeight)
	require.NoError(t, err)

	timeoutTimestampBz, err := vmtypes.SerializeUint64(timeoutTimestamp)
	require.NoError(t, err)

	memoBz, err := vmtypes.SerializeString(memo)
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunction(ctx, senderAddr, vmtypes.StdAddress,
		"cosmos",
		"transfer",
		[]vmtypes.TypeTag{},
		[][]byte{
			receiverBz,
			metadata[:],
			amountBz,
			sourcePortBz,
			sourceChannelBz,
			revisionNumberBz,
			revisionHeightBz,
			timeoutTimestampBz,
			memoBz,
		})
	require.NoError(t, err)

	events := ctx.EventManager().Events()
	event := events[len(events)-1]

	require.Equal(t, sdk.NewEvent("transfer",
		sdk.NewAttribute("sender", sender.String()),
		sdk.NewAttribute("receiver", receiver.String()),
		sdk.NewAttribute("token", sdk.NewCoin(denom, amount).String()),
		sdk.NewAttribute("source_port", sourcePort),
		sdk.NewAttribute("source_channel", sourceChannel),
		sdk.NewAttribute("timeout_height", clienttypes.NewHeight(revisionNumber, revisionHeight).String()),
		sdk.NewAttribute("timeout_timestamp", fmt.Sprint(timeoutTimestamp)),
		sdk.NewAttribute("memo", memo),
	), event)
}

func TestDispatchPayFeeMessage(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	sender := addrs[0]
	senderAddr, err := vmtypes.NewAccountAddressFromBytes(addrs[0])
	recvFeeDenom := testDenoms[0]
	recvFeeAmount := sdk.NewInt(100)
	ackFeeDenom := testDenoms[1]
	ackFeeAmount := sdk.NewInt(200)
	timeoutFeeDenom := testDenoms[2]
	timeoutFeeAmount := sdk.NewInt(300)

	sourcePort := "port-1"
	sourceChannel := "channel-1"

	recvFeeMetadata, err := types.MetadataAddressFromDenom(recvFeeDenom)
	require.NoError(t, err)
	ackFeeMetadata, err := types.MetadataAddressFromDenom(ackFeeDenom)
	require.NoError(t, err)
	timeoutFeeMetadata, err := types.MetadataAddressFromDenom(timeoutFeeDenom)
	require.NoError(t, err)

	recvFeeAmountBz, err := vmtypes.SerializeUint64(recvFeeAmount.Uint64())
	require.NoError(t, err)
	ackFeeAmountBz, err := vmtypes.SerializeUint64(ackFeeAmount.Uint64())
	require.NoError(t, err)
	timeoutFeeAmountBz, err := vmtypes.SerializeUint64(timeoutFeeAmount.Uint64())
	require.NoError(t, err)

	sourcePortBz, err := vmtypes.SerializeString(sourcePort)
	require.NoError(t, err)

	sourceChannelBz, err := vmtypes.SerializeString(sourceChannel)
	require.NoError(t, err)

	err = input.MoveKeeper.ExecuteEntryFunction(ctx, senderAddr, vmtypes.StdAddress,
		"cosmos",
		"pay_fee",
		[]vmtypes.TypeTag{},
		[][]byte{
			sourcePortBz,
			sourceChannelBz,
			recvFeeMetadata[:],
			recvFeeAmountBz,
			ackFeeMetadata[:],
			ackFeeAmountBz,
			timeoutFeeMetadata[:],
			timeoutFeeAmountBz,
		})
	require.NoError(t, err)

	events := ctx.EventManager().Events()
	event := events[len(events)-1]

	require.Equal(t, sdk.NewEvent("pay_fee",
		sdk.NewAttribute("signer", sender.String()),
		sdk.NewAttribute("source_port", sourcePort),
		sdk.NewAttribute("source_channel", sourceChannel),
		sdk.NewAttribute("recv_fee", sdk.NewCoins(sdk.NewCoin(recvFeeDenom, recvFeeAmount)).String()),
		sdk.NewAttribute("ack_fee", sdk.NewCoins(sdk.NewCoin(ackFeeDenom, ackFeeAmount)).String()),
		sdk.NewAttribute("timeout_fee", sdk.NewCoins(sdk.NewCoin(timeoutFeeDenom, timeoutFeeAmount)).String()),
		sdk.NewAttribute("relayers", ""),
	), event)
}

func Test_ContractSharedRevenue(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	stdAddr, err := vmtypes.NewAccountAddress("0x1")
	require.NoError(t, err)

	twoAddr, err := vmtypes.NewAccountAddress("0x2")
	require.NoError(t, err)

	gasUsages := []vmtypes.GasUsage{
		{
			ModuleId: vmtypes.ModuleId{
				Address: stdAddr,
			},
			GasUsed: 100,
		},
		{
			ModuleId: vmtypes.ModuleId{
				Address: twoAddr,
			},
			GasUsed: 200,
		},
	}

	// fund fee collector account
	feeCollectorAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	input.Faucet.Fund(ctx, feeCollectorAddr, sdk.NewCoin(bondDenom, sdk.NewInt(1_000_000_000_000)))

	// distribute without gas prices context value
	err = input.MoveKeeper.DistributeContractSharedRevenue(ctx, gasUsages)
	require.NoError(t, err)

	// should be zero
	require.Equal(t, sdk.ZeroInt(), input.BankKeeper.GetBalance(ctx, types.ConvertVMAddressToSDKAddress(stdAddr), bondDenom).Amount)
	require.Equal(t, sdk.ZeroInt(), input.BankKeeper.GetBalance(ctx, types.ConvertVMAddressToSDKAddress(twoAddr), bondDenom).Amount)

	// set gas prices as `1 bondDenom``
	ctx = ctx.WithValue(ante.GasPricesContextKey, sdk.NewDecCoinsFromCoins(sdk.NewCoin(bondDenom, sdk.NewInt(1))))

	// distribute with gas prices context value
	err = input.MoveKeeper.DistributeContractSharedRevenue(ctx, gasUsages)
	require.NoError(t, err)

	// 0x1 should be zero, but 0x2 should receive the coins
	require.Equal(t, sdk.ZeroInt(), input.BankKeeper.GetBalance(ctx, types.ConvertVMAddressToSDKAddress(stdAddr), bondDenom).Amount)
	require.Equal(t, input.MoveKeeper.ContractSharedRevenueRatio(ctx).MulInt64(200).TruncateInt(), input.BankKeeper.GetBalance(ctx, types.ConvertVMAddressToSDKAddress(twoAddr), bondDenom).Amount)
}
