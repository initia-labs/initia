package move_hooks_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"

	ibctesting "github.com/initia-labs/initia/x/ibc/testing"
	movetypes "github.com/initia-labs/initia/x/move/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

type TestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (suite *TestSuite) setup() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))
}

func newTransferPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = transfertypes.PortID
	path.EndpointB.ChannelConfig.PortID = transfertypes.PortID
	path.EndpointA.ChannelConfig.Version = transfertypes.Version
	path.EndpointB.ChannelConfig.Version = transfertypes.Version

	return path
}

func (suite *TestSuite) mintToken(chain *ibctesting.TestChain, recipient sdk.AccAddress, amount sdk.Coins) {
	ctx := chain.GetContext()
	err := chain.GetInitiaApp().MoveKeeper.MoveBankKeeper().MintCoins(ctx, recipient, amount)
	suite.Require().NoError(err)
}

func (suite *TestSuite) transfer(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	token sdk.Coin,
	sender, receiver string,
	memo string,
) channeltypes.Packet {
	msgTransfer := transfertypes.NewMsgTransfer(
		fromEndpoint.ChannelConfig.PortID,
		fromEndpoint.ChannelID,
		token,
		sender,
		receiver,
		toEndpoint.Chain.GetTimeoutHeight(),
		0,
		memo,
	)

	res, err := fromEndpoint.Chain.SendMsgs(msgTransfer)
	suite.Require().NoError(err)

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	var data transfertypes.FungibleTokenPacketData
	err = suite.chainA.Codec.UnmarshalJSON(packet.GetData(), &data)
	suite.Require().NoError(err)

	return packet
}

func (suite *TestSuite) receive(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	packet channeltypes.Packet,
) {

	var data transfertypes.FungibleTokenPacketData
	err := suite.chainA.Codec.UnmarshalJSON(packet.GetData(), &data)
	suite.Require().NoError(err)

	// get proof of packet commitment from chainA
	err = toEndpoint.UpdateClient()
	suite.Require().NoError(err)

	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := fromEndpoint.QueryProof(packetKey)

	recvMsg := channeltypes.NewMsgRecvPacket(
		packet, proof, proofHeight, toEndpoint.Chain.SenderAccount.GetAddress().String())
	_, err = toEndpoint.Chain.SendMsgs(recvMsg)
	suite.Require().NoError(err) // message committed
}

func (suite *TestSuite) ack(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	packet channeltypes.Packet,
	acknowledgement []byte,
) {
	var data transfertypes.FungibleTokenPacketData
	err := suite.chainA.Codec.UnmarshalJSON(packet.GetData(), &data)
	suite.Require().NoError(err)

	err = fromEndpoint.UpdateClient()
	suite.Require().NoError(err)

}

func (suite *TestSuite) getBalance(chain *ibctesting.TestChain, addr sdk.AccAddress, denom string) sdk.Coin {
	ctx := chain.GetContext()
	balance := chain.GetInitiaApp().BankKeeper.GetBalance(ctx, addr, denom)
	return balance
}

func (suite *TestSuite) publishModule(chain *ibctesting.TestChain, addr sdk.AccAddress, moduleBundle vmtypes.ModuleBundle) {
	ctx := chain.GetContext()
	moveKeeper := chain.GetInitiaApp().MoveKeeper

	err := moveKeeper.PublishModuleBundle(ctx, movetypes.ConvertSDKAddressToVMAddress(addr), moduleBundle, movetypes.UpgradePolicy_COMPATIBLE)
	suite.Require().NoError(err)
}

func (suite *TestSuite) allowACL(chain *ibctesting.TestChain, addr sdk.AccAddress) {
	ctx := chain.GetContext()
	ibcHooksKeeper := chain.GetInitiaApp().IBCHooksKeeper

	err := ibcHooksKeeper.SetAllowed(ctx, movetypes.ConvertVMAddressToSDKAddress(movetypes.ConvertSDKAddressToVMAddress(addr)), true)
	suite.Require().NoError(err)
}

func (suite *TestSuite) getAsyncCallback(chain *ibctesting.TestChain, sourcePortID, sourceChannelID string, packetID uint64, shouldEmpty bool) []byte {
	ctx := chain.GetContext()
	ibcHooksKeeper := chain.GetInitiaApp().IBCHooksKeeper

	callback, err := ibcHooksKeeper.GetAsyncCallback(ctx, sourcePortID, sourceChannelID, packetID)
	if shouldEmpty {
		suite.Require().ErrorContains(err, "not found")
	} else {
		suite.Require().NoError(err)
	}

	return callback
}

func TestTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (suite *TestSuite) TestAsyncCallback() {
	suite.setup()

	pathA2B := newTransferPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(pathA2B)
	suite.coordinator.CreateChannels(pathA2B)

	sender := pathA2B.EndpointA.Chain.SenderAccount.GetAddress()
	receiver := pathA2B.EndpointB.Chain.SenderAccount.GetAddress()

	mintAmount := sdk.NewCoins(sdk.NewCoin("uinit", sdkmath.NewInt(1_000_000_000_000)))
	suite.mintToken(suite.chainA, sender, mintAmount)
	suite.publishModule(suite.chainA, movetypes.StdAddr, vmtypes.NewModuleBundle(vmtypes.NewModule(counterModule)))
	suite.allowACL(suite.chainA, movetypes.StdAddr)

	// 1. send transfer with async callback memo
	packet := suite.transfer(
		pathA2B.EndpointA,
		pathA2B.EndpointB,
		sdk.NewCoin("uinit", sdkmath.NewInt(1_000_000)),
		sender.String(),
		receiver.String(),
		"{\"move\":{\"async_callback\":{\"id\":1,\"module_address\":\"0x1\",\"module_name\":\"Counter\"}}}",
	)

	escrowAddress := transfertypes.GetEscrowAddress(pathA2B.EndpointA.ChannelConfig.PortID, pathA2B.EndpointA.ChannelID)
	balance := suite.getBalance(suite.chainA, escrowAddress, "uinit")
	suite.Require().Equal(sdk.NewCoin("uinit", sdkmath.NewInt(1_000_000)), balance)

	callback := suite.getAsyncCallback(suite.chainA, packet.SourcePort, packet.SourceChannel, packet.Sequence, false)
	suite.Require().Equal([]byte("{\"id\":1,\"module_address\":\"0x1\",\"module_name\":\"Counter\"}"), callback)

	// 2. receive the packet on chainB
	// packet found, relay from A to B
	err := pathA2B.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	res, err := pathA2B.EndpointB.RecvPacketWithResult(packet)
	suite.Require().NoError(err)

	ack, err := ibctesting.ParseAckFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	// 3. acknowledge the packet on chainA
	err = pathA2B.EndpointA.AcknowledgePacket(packet, ack)
	suite.Require().NoError(err)

	// check the contract state
	queryRes, _, err := suite.chainA.GetInitiaApp().MoveKeeper.ExecuteViewFunctionJSON(
		suite.chainA.GetContext(),
		vmtypes.StdAddress,
		"Counter",
		"get",
		[]vmtypes.TypeTag{},
		[]string{},
	)
	suite.Require().NoError(err)
	suite.Require().Equal("\"1\"", queryRes.Ret)
	suite.getAsyncCallback(suite.chainA, packet.SourcePort, packet.SourceChannel, packet.Sequence, true)
}
