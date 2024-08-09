package keeper_test

import (
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"

	sdktypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	ibctesting "github.com/initia-labs/initia/x/ibc/testing"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func (suite *KeeperTestSuite) CreateNftClass(
	endpoint *ibctesting.Endpoint,
	name, uri, desc string,
) string {
	nftKeeper := movekeeper.NewNftKeeper(endpoint.Chain.GetInitiaApp().MoveKeeper)

	//make_collection

	nameBz, err := vmtypes.SerializeString(name)
	suite.Require().NoError(err)

	uriBz, err := vmtypes.SerializeString(uri)
	suite.Require().NoError(err)

	descBz, err := vmtypes.SerializeString(desc)
	suite.Require().NoError(err)

	ctx := endpoint.Chain.GetContext()
	err = nftKeeper.ExecuteEntryFunction(
		ctx,
		vmtypes.TestAddress,
		vmtypes.StdAddress,
		movetypes.MoveModuleNameInitiaNft,
		movetypes.FunctionNameInitiaNftCreateCollection,
		[]vmtypes.TypeTag{},
		[][]byte{descBz, {0}, nameBz, uriBz, {0}, {0}, {0}, {0}, {0}, {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
	)
	suite.Require().NoError(err, "MakeCollection error on chain")

	collection := movetypes.NamedObjectAddress(vmtypes.TestAddress, name)
	classId, err := movetypes.ClassIdFromCollectionAddress(endpoint.Chain.GetContext(), nftKeeper, collection)
	suite.Require().NoError(err)

	return classId
}

func (suite *KeeperTestSuite) MintNft(
	endpoint *ibctesting.Endpoint,
	receiver sdktypes.AccAddress,
	classId, className, tokenId, tokenUri, tokenData string,
) {
	classNameBz, err := vmtypes.SerializeString(className)
	suite.Require().NoError(err)

	idBz, err := vmtypes.SerializeString(tokenId)
	suite.Require().NoError(err)

	uriBz, err := vmtypes.SerializeString(tokenUri)
	suite.Require().NoError(err)

	dataBz, err := vmtypes.SerializeString(tokenData)
	suite.Require().NoError(err)

	nftKeeper := movekeeper.NewNftKeeper(endpoint.Chain.GetInitiaApp().MoveKeeper)

	receiverAddr, err := vmtypes.NewAccountAddressFromBytes(receiver[:])
	suite.Require().NoError(err)

	err = nftKeeper.ExecuteEntryFunction(
		endpoint.Chain.GetContext(),
		vmtypes.TestAddress,
		vmtypes.StdAddress,
		movetypes.MoveModuleNameInitiaNft,
		movetypes.FunctionNameInitiaNftMint,
		[]vmtypes.TypeTag{},
		[][]byte{classNameBz, dataBz, idBz, uriBz, {1}, append([]byte{1}, receiverAddr[:]...)},
	)
	suite.Require().NoError(err, "MakeCollection error on chain")
}

func (suite *KeeperTestSuite) ConfirmClassId(endpoint *ibctesting.Endpoint, classId, targetClassId string) {
	if classId == targetClassId {
		return
	}
	classIdPath, err := endpoint.Chain.GetInitiaApp().NftTransferKeeper.ClassIdPathFromHash(endpoint.Chain.GetContext(), targetClassId)
	suite.Require().NoError(err, "ClassIdPathFromHash error on chain %s", endpoint.Chain.ChainID)

	baseClassId := types.ParseClassTrace(classIdPath).BaseClassId
	suite.Equal(classId, baseClassId, "wrong classId on chain %s", endpoint.Chain.ChainID)
}

// The following test describes the entire cross-chain process of nft-transfer.
// The execution sequence of the cross-chain process is:
// A -> B -> C -> B ->A
func (suite *KeeperTestSuite) TestSendAndReceive() {
	suite.SetupTest()

	var classId string
	classUri := "uri"
	className := "name"
	classSymbol := "symbol"
	nftId := "kitty"
	nftUri := "kitty_uri"
	nftData := "kitty_data"

	var targetClassId string
	var packet channeltypes.Packet

	// WARNING : be careful not to be confused with endpoint names
	// pathB2C.EndpointA is ChainB endpoint (source of path)`
	// pathB2C.EndpointB is ChainC endpoint (destination of path)
	// pathA2B.EndpointB.Chain.SenderAccount is same with receiver account of pathA2B before testing`
	pathA2B := NewTransferPath(suite.chainA, suite.chainB)
	suite.Run("transfer forward A->B", func() {
		{
			suite.coordinator.SetupConnections(pathA2B)
			suite.coordinator.CreateChannels(pathA2B)

			sender := pathA2B.EndpointA.Chain.SenderAccount.GetAddress()
			receiver := pathA2B.EndpointB.Chain.SenderAccount.GetAddress()

			classId = suite.CreateNftClass(pathA2B.EndpointA, className, classUri, classSymbol)
			suite.MintNft(pathA2B.EndpointA, sender, classId, className, nftId, nftUri, nftData)

			packet = suite.transferNft(
				pathA2B.EndpointA,
				pathA2B.EndpointB,
				classId,
				nftId,
				sender.String(),
				receiver.String(),
			)

			targetClassId = suite.receiverNft(
				pathA2B.EndpointA,
				pathA2B.EndpointB,
				packet,
			)

			suite.ConfirmClassId(pathA2B.EndpointB, classId, targetClassId)
		}
	})

	// transfer from chainB to chainC
	pathB2C := NewTransferPath(suite.chainB, suite.chainC)
	suite.Run("transfer forward B->C", func() {
		{
			suite.coordinator.SetupConnections(pathB2C)
			suite.coordinator.CreateChannels(pathB2C)

			sender := pathA2B.EndpointB.Chain.SenderAccount.GetAddress()
			receiver := pathB2C.EndpointB.Chain.SenderAccount.GetAddress()

			packet = suite.transferNft(
				pathB2C.EndpointA,
				pathB2C.EndpointB,
				targetClassId,
				nftId,
				sender.String(),
				receiver.String(),
			)

			targetClassId = suite.receiverNft(
				pathB2C.EndpointA,
				pathB2C.EndpointB,
				packet,
			)

			suite.ConfirmClassId(pathB2C.EndpointB, classId, targetClassId)
		}
	})

	// transfer from chainC to chainB
	suite.Run("transfer back C->B", func() {
		{
			sender := pathB2C.EndpointB.Chain.SenderAccount.GetAddress()
			receiver := pathB2C.EndpointA.Chain.SenderAccount.GetAddress()

			packet = suite.transferNft(
				pathB2C.EndpointB,
				pathB2C.EndpointA,
				targetClassId,
				nftId,
				sender.String(),
				receiver.String(),
			)

			targetClassId = suite.receiverNft(
				pathB2C.EndpointB,
				pathB2C.EndpointA,
				packet,
			)

			suite.ConfirmClassId(pathB2C.EndpointA, classId, targetClassId)
		}
	})

	// transfer from chainB to chainA
	suite.Run("transfer back B->A", func() {
		{
			sender := pathA2B.EndpointB.Chain.SenderAccount.GetAddress()
			receiver := pathA2B.EndpointA.Chain.SenderAccount.GetAddress()

			packet = suite.transferNft(
				pathA2B.EndpointB,
				pathA2B.EndpointA,
				targetClassId,
				nftId,
				sender.String(),
				receiver.String(),
			)

			targetClassId = suite.receiverNft(
				pathA2B.EndpointB,
				pathA2B.EndpointA,
				packet,
			)

			suite.ConfirmClassId(pathA2B.EndpointA, classId, targetClassId)
		}
	})
}

func (suite *KeeperTestSuite) transferNft(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	classId, nftId string,
	sender, receiver string,
) channeltypes.Packet {
	msgTransfer := types.NewMsgTransfer(
		fromEndpoint.ChannelConfig.PortID,
		fromEndpoint.ChannelID,
		classId,
		[]string{nftId},
		sender,
		receiver,
		toEndpoint.Chain.GetTimeoutHeight(),
		0,
		"",
	)

	res, err := fromEndpoint.Chain.SendMsgs(msgTransfer)
	suite.Require().NoError(err)

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	var data types.NonFungibleTokenPacketData
	err = suite.chainA.Codec.UnmarshalJSON(packet.GetData(), &data)
	suite.Require().NoError(err)

	return packet
}

func (suite *KeeperTestSuite) receiverNft(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	packet channeltypes.Packet,
) string {
	var data types.NonFungibleTokenPacketData
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

	var classId string
	var className string

	isAwayFromOrigin := types.SenderChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), data.GetClassId())
	if isAwayFromOrigin {
		prefixedClassId := types.GetClassIdPrefix(toEndpoint.ChannelConfig.PortID, toEndpoint.ChannelID) + data.GetClassId()
		trace := types.ParseClassTrace(prefixedClassId)
		classId = trace.IBCClassId()
	} else {
		unprefixedClassId, err := types.RemoveClassPrefix(packet.GetSourcePort(), packet.GetSourceChannel(), data.GetClassId())
		suite.Require().NoError(err)

		classId = unprefixedClassId
		classTrace := types.ParseClassTrace(unprefixedClassId)
		if classTrace.Path != "" {
			classId = classTrace.IBCClassId()
		} else {
			className, data.ClassData, err = types.ConvertClassDataFromICS721(data.ClassData)
			suite.Require().NoError(err, "ConvertTokenDataFromICS721 error on chain %s", toEndpoint.Chain.ChainID)
		}
	}
	toNftKeeper := movekeeper.NewNftKeeper(toEndpoint.Chain.GetInitiaApp().MoveKeeper)
	_className, classUri, classData, err := toNftKeeper.GetClassInfo(toEndpoint.Chain.GetContext(), classId)
	suite.Require().NoError(err, "not found class")
	suite.Require().Equal(classUri, data.GetClassUri(), "class uri not equal")
	suite.Require().Equal(classData, data.GetClassData(), "class data not equal")
	if className != "" {
		suite.Require().Equal(_className, className, "class name not equal")
	}
	return classId
}
