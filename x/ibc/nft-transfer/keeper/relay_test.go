package keeper_test

import (
	"context"
	"strings"

	"github.com/initia-labs/initia/v1/x/ibc/nft-transfer/keeper"
	"github.com/initia-labs/initia/v1/x/ibc/nft-transfer/types"
	movekeeper "github.com/initia-labs/initia/v1/x/move/keeper"
	movetypes "github.com/initia-labs/initia/v1/x/move/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

func (suite *KeeperTestSuite) GetNFTOwner(ctx context.Context, k *keeper.Keeper, moveKeeper *movekeeper.Keeper, nftKeeper *movekeeper.NftKeeper, classId, className, nftId string) sdk.AccAddress {
	ac := k.Codec().InterfaceRegistry().SigningContext().AddressCodec()
	moduleAddr, err := movetypes.AccAddressFromString(ac, "0x1")
	suite.Require().NoError(err)

	typeTags, err := movetypes.TypeTagsFromTypeArgs([]string{"0x1::nft::Nft"})
	suite.Require().NoError(err)

	collectionAddr, err := movetypes.CollectionAddressFromClassId(classId)
	suite.Require().NoError(err)

	collectionCreator, _, _, _, err := nftKeeper.CollectionInfo(ctx, collectionAddr)
	suite.Require().NoError(err)

	tokenAddr, err := movetypes.TokenAddressFromTokenId(collectionCreator, className, nftId)
	suite.Require().NoError(err)

	ta, err := tokenAddr.BcsSerialize()
	suite.Require().NoError(err)

	res, _, err := moveKeeper.ExecuteViewFunction(ctx, moduleAddr, "object", "owner", typeTags, [][]byte{ta})
	suite.Require().NoError(err)

	strAddr := strings.Trim(res.Ret, "\"")
	addr, err := movetypes.AccAddressFromString(ac, strAddr)
	suite.Require().NoError(err)

	return movetypes.ConvertVMAddressToSDKAddress(addr)
}

func (suite *KeeperTestSuite) TestOnAcknowledgementPacket() {
	suite.SetupTest()
	k := suite.chainA.GetInitiaApp().NftTransferKeeper
	moveKeeper := suite.chainA.GetInitiaApp().MoveKeeper
	nftKeeper := movekeeper.NewNftKeeper(moveKeeper)

	ctx := suite.chainA.GetContext()

	var classId string
	classUri := "uri"
	className := "name"
	classSymbol := "symbol"
	nftId := "kitty"
	nftUri := "kitty_uri"
	nftData := "kitty_data"

	pathA2B := NewTransferPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(pathA2B)
	suite.coordinator.CreateChannels(pathA2B)

	sender := pathA2B.EndpointA.Chain.SenderAccount.GetAddress()
	receiver := pathA2B.EndpointB.Chain.SenderAccount.GetAddress()

	classId = suite.CreateNftClass(pathA2B.EndpointA, className, classUri, classSymbol)
	suite.MintNft(pathA2B.EndpointA, sender, classId, className, nftId, nftUri, nftData)

	owner := suite.GetNFTOwner(ctx, k, moveKeeper, &nftKeeper, classId, className, nftId)
	suite.Require().Equal(sender, owner)

	packet := suite.transferNft(
		pathA2B.EndpointA,
		pathA2B.EndpointB,
		classId,
		nftId,
		sender.String(),
		receiver.String(),
	)

	escrowAddress := types.GetEscrowAddress(pathA2B.EndpointA.ChannelConfig.PortID, pathA2B.EndpointA.ChannelID)

	owner = suite.GetNFTOwner(ctx, k, moveKeeper, &nftKeeper, classId, className, nftId)
	suite.Require().Equal(escrowAddress, owner)

	ack := channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Result{},
	}

	var data types.NonFungibleTokenPacketData
	err := k.Codec().UnmarshalJSON(packet.GetData(), &data)
	suite.Require().NoError(err)

	// no refund
	err = k.OnAcknowledgementPacket(ctx, packet, data, ack)
	suite.Require().NoError(err)

	owner = suite.GetNFTOwner(ctx, k, moveKeeper, &nftKeeper, classId, className, nftId)
	suite.Require().Equal(escrowAddress, owner)

	ack = channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{},
	}

	// should refund
	err = k.OnAcknowledgementPacket(ctx, packet, data, ack)
	suite.Require().NoError(err)

	owner = suite.GetNFTOwner(ctx, k, moveKeeper, &nftKeeper, classId, className, nftId)
	suite.Require().Equal(sender, owner)
}

func (suite *KeeperTestSuite) TestOnTimeoutPacket() {
	suite.SetupTest()
	k := suite.chainA.GetInitiaApp().NftTransferKeeper
	moveKeeper := suite.chainA.GetInitiaApp().MoveKeeper
	nftKeeper := movekeeper.NewNftKeeper(moveKeeper)

	ctx := suite.chainA.GetContext()

	var classId string
	classUri := "uri"
	className := "name"
	classSymbol := "symbol"
	nftId := "kitty"
	nftUri := "kitty_uri"
	nftData := "kitty_data"

	pathA2B := NewTransferPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(pathA2B)
	suite.coordinator.CreateChannels(pathA2B)

	sender := pathA2B.EndpointA.Chain.SenderAccount.GetAddress()
	receiver := pathA2B.EndpointB.Chain.SenderAccount.GetAddress()

	classId = suite.CreateNftClass(pathA2B.EndpointA, className, classUri, classSymbol)
	suite.MintNft(pathA2B.EndpointA, sender, classId, className, nftId, nftUri, nftData)

	owner := suite.GetNFTOwner(ctx, k, moveKeeper, &nftKeeper, classId, className, nftId)
	suite.Require().Equal(sender, owner)

	packet := suite.transferNft(
		pathA2B.EndpointA,
		pathA2B.EndpointB,
		classId,
		nftId,
		sender.String(),
		receiver.String(),
	)

	escrowAddress := types.GetEscrowAddress(pathA2B.EndpointA.ChannelConfig.PortID, pathA2B.EndpointA.ChannelID)

	owner = suite.GetNFTOwner(ctx, k, moveKeeper, &nftKeeper, classId, className, nftId)
	suite.Require().Equal(escrowAddress, owner)

	var data types.NonFungibleTokenPacketData
	err := k.Codec().UnmarshalJSON(packet.GetData(), &data)
	suite.Require().NoError(err)

	// should refund
	err = k.OnTimeoutPacket(ctx, packet, data)
	suite.Require().NoError(err)

	owner = suite.GetNFTOwner(ctx, k, moveKeeper, &nftKeeper, classId, className, nftId)
	suite.Require().Equal(sender, owner)
}

func (suite *KeeperTestSuite) TestClassIdPathFromHash() {
	ctx, k := suite.SetupKeeperTest()

	path := "nft-transfer/channel-1/gaeguri"
	classTrace := types.ParseClassTrace(path)
	hash := classTrace.Hash()
	err := k.ClassTraces.Set(ctx, hash, classTrace)
	suite.Require().NoError(err)

	fullClassIdPath, err := k.ClassIdPathFromHash(sdk.UnwrapSDKContext(ctx), "ibc/"+hash.String())
	suite.Require().NoError(err)
	suite.Require().Equal(path, fullClassIdPath)
}
