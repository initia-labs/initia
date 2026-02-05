package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	ibctesting "github.com/initia-labs/initia/x/ibc/testing"
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

func (suite *TestSuite) transferForTimeout(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	token sdk.Coin,
	sender, receiver string,
	memo string,
) channeltypes.Packet {

	timeoutHeight := clienttypes.GetSelfHeight(toEndpoint.Chain.GetContext())
	msgTransfer := transfertypes.NewMsgTransfer(
		fromEndpoint.ChannelConfig.PortID,
		fromEndpoint.ChannelID,
		token,
		sender,
		receiver,
		timeoutHeight,
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

func (suite *TestSuite) getBalance(chain *ibctesting.TestChain, addr sdk.AccAddress, denom string) sdk.Coin {
	ctx := chain.GetContext()
	balance := chain.GetInitiaApp().BankKeeper.GetBalance(ctx, addr, denom)
	return balance
}
func TestTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (suite *TestSuite) Test_DispatchableWhitelist_Timeout() {
	suite.setup()

	createDispatchableToken(
		suite.T(), suite.chainA.GetContext(),
		suite.chainA.GetInitiaApp(), []sdk.AccAddress{suite.chainA.SenderAccount.GetAddress()},
	)

	denom := dispatchableTokenDenom(suite.T())

	pathA2B := newTransferPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(pathA2B)
	suite.coordinator.CreateChannels(pathA2B)

	sender := pathA2B.EndpointA.Chain.SenderAccount.GetAddress()
	receiver := pathA2B.EndpointB.Chain.SenderAccount.GetAddress()

	// 1. send transfer with async callback memo
	packet := suite.transferForTimeout(
		pathA2B.EndpointA,
		pathA2B.EndpointB,
		sdk.NewCoin(denom, sdkmath.NewInt(1_000_000)),
		sender.String(),
		receiver.String(),
		"",
	)

	escrowAddress := transfertypes.GetEscrowAddress(pathA2B.EndpointA.ChannelConfig.PortID, pathA2B.EndpointA.ChannelID)
	balance := suite.getBalance(suite.chainA, escrowAddress, denom)
	suite.Require().Equal(sdk.NewCoin(denom, sdkmath.NewInt(10_000_000)), balance)

	// 2. need to update chainA's client representing chainB to prove missing ack
	err := pathA2B.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	// 3. acknowledge the packet on chainA
	err = pathA2B.EndpointA.TimeoutPacket(packet)
	suite.Require().NoError(err)
}
