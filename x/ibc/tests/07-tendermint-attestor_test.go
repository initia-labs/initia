package tests

import (
	"testing"

	"github.com/stretchr/testify/suite"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibctesting "github.com/initia-labs/initia/x/ibc/testing"

	sdkmath "cosmossdk.io/math"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TMAttestorTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func TestTMAttestorTestSuite(t *testing.T) {
	suite.Run(t, new(TMAttestorTestSuite))
}

func (suite *TMAttestorTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
}

func (suite *TMAttestorTestSuite) TestTendermintAttestor() {
	suite.SetupTest()

	path := ibctesting.NewPathWithOneTendermintAttestor(suite.chainA, suite.chainB, 5, 3)

	suite.coordinator.SetupConnections(path)
	suite.coordinator.CreateTransferChannels(path)

	msgTransfer := transfertypes.NewMsgTransfer(
		ibctesting.TransferPort,
		ibctesting.FirstChannelID,
		sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100)),
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(),
		clienttypes.NewHeight(1, 100),
		0,
		"",
	)
	_, err := suite.chainA.SendMsgs(msgTransfer)
	suite.Require().NoError(err)
}

func (suite *TMAttestorTestSuite) TestTendermintZeroAttestor() {
	suite.SetupTest()

	path := ibctesting.NewPathWithOneTendermintAttestor(suite.chainA, suite.chainB, 0, 0)

	suite.coordinator.SetupConnections(path)
	suite.coordinator.CreateTransferChannels(path)

	msgTransfer := transfertypes.NewMsgTransfer(
		ibctesting.TransferPort,
		ibctesting.FirstChannelID,
		sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100)),
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(),
		clienttypes.NewHeight(1, 100),
		0,
		"",
	)
	_, err := suite.chainA.SendMsgs(msgTransfer)
	suite.Require().NoError(err)
}

func (suite *TMAttestorTestSuite) TestTendermintAttestorAnotherConnection() {
	suite.SetupTest()

	path := ibctesting.NewPathWithOneTendermintAttestor(suite.chainA, suite.chainB, 0, 0)

	suite.coordinator.SetupConnections(path)
	suite.coordinator.CreateTransferChannels(path)

	suite.Require().Equal(path.EndpointA.ConnectionID, ibctesting.FirstConnectionID)
	suite.Require().Equal(path.EndpointB.ConnectionID, ibctesting.FirstConnectionID)

	err := suite.coordinator.CreateConnections(path)
	suite.Require().NoError(err)
	suite.coordinator.CreateTransferChannels(path)

	suite.Require().Equal(path.EndpointA.ConnectionID, ibctesting.SecondConnectionID)
	suite.Require().Equal(path.EndpointB.ConnectionID, ibctesting.SecondConnectionID)

	msgTransfer := transfertypes.NewMsgTransfer(
		ibctesting.TransferPort,
		ibctesting.FirstChannelID,
		sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100)),
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(),
		clienttypes.NewHeight(1, 100),
		0,
		"",
	)
	_, err = suite.chainA.SendMsgs(msgTransfer)
	suite.Require().NoError(err)
}

func (suite *TMAttestorTestSuite) TestTendermintChangeAttestorSet() {
	suite.SetupTest()

	path := ibctesting.NewPathWithOneTendermintAttestor(suite.chainA, suite.chainB, 0, 0)

	suite.coordinator.SetupConnections(path)
	suite.Require().Equal(path.EndpointA.ClientID, ibctesting.FirstAttestorClientID)
	suite.Require().Equal(path.EndpointB.ClientID, ibctesting.FirstClientID)
	suite.Require().Equal(path.EndpointA.ConnectionID, ibctesting.FirstConnectionID)
	suite.Require().Equal(path.EndpointB.ConnectionID, ibctesting.FirstConnectionID)
	suite.coordinator.CreateTransferChannels(path)

	path.EndpointA.ClientConfig = ibctesting.NewTendermintAttestorConfig(2, 3)
	path.EndpointA.CreateClient()
	err := suite.coordinator.CreateConnections(path)
	suite.Require().Error(err)

	path.EndpointA.ClientConfig = ibctesting.NewTendermintAttestorConfig(5, 3)
	path.EndpointA.CreateClient()
	suite.Require().Equal(path.EndpointA.ClientID, ibctesting.ThirdAttestorClientID)
	suite.Require().Equal(path.EndpointB.ClientID, ibctesting.FirstClientID)
	err = suite.coordinator.CreateConnections(path)
	suite.Require().NoError(err)
	suite.Require().Equal(path.EndpointA.ConnectionID, ibctesting.ThirdConnectionID)
	suite.Require().Equal(path.EndpointB.ConnectionID, ibctesting.ThirdConnectionID)

	suite.coordinator.CreateTransferChannels(path)
	suite.Require().Equal(path.EndpointA.ChannelID, ibctesting.SecondChannelID)
	suite.Require().Equal(path.EndpointB.ChannelID, ibctesting.SecondChannelID)

	msgTransfer := transfertypes.NewMsgTransfer(
		ibctesting.TransferPort,
		ibctesting.FirstChannelID,
		sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100)),
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(),
		clienttypes.NewHeight(1, 100),
		0,
		"",
	)
	_, err = suite.chainA.SendMsgs(msgTransfer)
	suite.Require().NoError(err)
}

func (suite *TMAttestorTestSuite) TestTendermintChangeLightClientAndUpgradeChannel() {
	suite.SetupTest()

	// create a path with 07-tendermint light clients
	path := ibctesting.NewPath(suite.chainA, suite.chainB)

	suite.coordinator.SetupConnections(path)
	suite.Require().Equal(path.EndpointA.ClientID, ibctesting.FirstClientID)
	suite.Require().Equal(path.EndpointB.ClientID, ibctesting.FirstClientID)
	suite.Require().Equal(path.EndpointA.ConnectionID, ibctesting.FirstConnectionID)
	suite.Require().Equal(path.EndpointB.ConnectionID, ibctesting.FirstConnectionID)
	suite.coordinator.CreateTransferChannels(path)

	suite.Require().Equal(path.EndpointA.ChannelID, ibctesting.FirstChannelID)
	suite.Require().Equal(path.EndpointB.ChannelID, ibctesting.FirstChannelID)

	msgTransfer := transfertypes.NewMsgTransfer(
		ibctesting.TransferPort,
		ibctesting.FirstChannelID,
		sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100)),
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(),
		clienttypes.NewHeight(1, 100),
		0,
		"",
	)
	res, err := suite.chainA.SendMsgs(msgTransfer)
	suite.Require().NoError(err)

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)
	err = path.RelayPacket(packet)
	suite.Require().NoError(err)

	// create new path with 07-tendermint-attestor light client
	path.EndpointA.ClientConfig = ibctesting.NewTendermintAttestorConfig(0, 0)
	path.EndpointA.CreateClient()

	suite.Require().Equal(path.EndpointA.ClientID, ibctesting.SecondAttestorClientID)
	suite.Require().Equal(path.EndpointB.ClientID, ibctesting.FirstClientID)
	err = suite.coordinator.CreateConnections(path)
	suite.Require().NoError(err)
	suite.Require().Equal(path.EndpointA.ConnectionID, ibctesting.SecondConnectionID)
	suite.Require().Equal(path.EndpointB.ConnectionID, ibctesting.SecondConnectionID)

	// apply old client config temporarily to upgrade channel
	path.EndpointA.ClientConfig = ibctesting.NewTendermintConfig()
	path.EndpointA.ClientID = ibctesting.FirstClientID
	path.EndpointB.ClientID = ibctesting.FirstClientID
	path.EndpointA.ConnectionID = ibctesting.FirstConnectionID
	path.EndpointB.ConnectionID = ibctesting.FirstConnectionID
	path.EndpointA.UpgradeChannel(
		[]string{ibctesting.SecondConnectionID},
		[]string{ibctesting.SecondConnectionID},
	)
	path.EndpointA.ClientConfig = ibctesting.NewTendermintAttestorConfig(0, 0)

	// should be equal to original channel
	suite.Require().Equal(path.EndpointA.ChannelID, ibctesting.FirstChannelID)
	suite.Require().Equal(path.EndpointB.ChannelID, ibctesting.FirstChannelID)

	_, err = suite.chainA.SendMsgs(msgTransfer)
	suite.Require().NoError(err)
}
