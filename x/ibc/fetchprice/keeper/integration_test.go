package keeper_test

import (
	"math/rand"
	"time"

	"cosmossdk.io/math"
	icqtypes "github.com/cosmos/ibc-apps/modules/async-icq/v8/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"

	"github.com/initia-labs/initia/x/ibc/fetchprice/types"
	ibctesting "github.com/initia-labs/initia/x/ibc/testing"

	oracletypes "github.com/skip-mev/slinky/x/oracle/types"
)

func (suite *KeeperTestSuite) EnableFetchPrice(
	endpoint *ibctesting.Endpoint,
) {
	ctx := endpoint.Chain.GetContext()

	fetchPriceKeeper := endpoint.Chain.GetInitiaApp().FetchPriceKeeper
	params, err := fetchPriceKeeper.Params.Get(ctx)
	suite.NoError(err)
	params.FetchEnabled = true
	err = fetchPriceKeeper.Params.Set(ctx, params)
	suite.NoError(err)
}

func (suite *KeeperTestSuite) CreateOracleCurrencyPair(
	endpoint *ibctesting.Endpoint,
	currencyPair string,
) {
	ctx := endpoint.Chain.GetContext()

	cp, err := oracletypes.CurrencyPairFromString(currencyPair)
	suite.NoError(err)

	oracleKeeper := endpoint.Chain.GetInitiaApp().OracleKeeper
	err = oracleKeeper.CreateCurrencyPair(ctx, cp)
	suite.NoError(err)
}

func (suite *KeeperTestSuite) RegisterOracleCurrencyPrice(
	endpoint *ibctesting.Endpoint,
	currencyPair string,
	price math.Int,
) {
	ctx := endpoint.Chain.GetContext()

	cp, err := oracletypes.CurrencyPairFromString(currencyPair)
	suite.NoError(err)

	oracleKeeper := endpoint.Chain.GetInitiaApp().OracleKeeper
	err = oracleKeeper.SetPriceForCurrencyPair(ctx, cp, oracletypes.QuotePrice{
		Price:          price,
		BlockTimestamp: ctx.BlockTime(),
		BlockHeight:    uint64(ctx.BlockHeight()),
	})
	suite.NoError(err)
}

// The following test describes the entire cross-chain process of fetch price.
func (suite *KeeperTestSuite) TestFetchPrice() {
	suite.SetupTest()

	pairBTC := "BTC/USD"
	pairETH := "ETH/USD"

	priceBTC := math.NewInt(rand.Int63())
	priceETH := math.NewInt(rand.Int63())

	var packet channeltypes.Packet

	pathA2B := NewFetchPricePath(suite.chainA, suite.chainB)
	suite.Run("fetch price A->B", func() {
		{
			// enable fetchprice
			suite.EnableFetchPrice(pathA2B.EndpointA)

			suite.coordinator.SetupConnections(pathA2B)
			suite.coordinator.CreateChannels(pathA2B)

			// activate is authority message
			sender := pathA2B.EndpointA.Chain.SenderAccount.GetAddress()

			suite.CreateOracleCurrencyPair(pathA2B.EndpointA, pairBTC)
			suite.CreateOracleCurrencyPair(pathA2B.EndpointA, pairETH)
			suite.CreateOracleCurrencyPair(pathA2B.EndpointB, pairBTC)
			suite.CreateOracleCurrencyPair(pathA2B.EndpointB, pairETH)
			suite.RegisterOracleCurrencyPrice(pathA2B.EndpointB, pairBTC, priceBTC)
			suite.RegisterOracleCurrencyPrice(pathA2B.EndpointB, pairETH, priceETH)

			packet = suite.sendFetchPrice(
				pathA2B.EndpointA,
				pathA2B.EndpointB,
				sender.String(),
				[]string{
					pairBTC, pairETH,
				},
			)

			ackBz := suite.receiveFetchPrice(
				pathA2B.EndpointA,
				pathA2B.EndpointB,
				packet,
			)

			err := pathA2B.EndpointA.UpdateClient()
			suite.NoError(err)
			err = pathA2B.EndpointA.AcknowledgePacket(packet, ackBz)
			suite.NoError(err)

			suite.validateFetchedPrices(
				pathA2B.EndpointA,
				pathA2B.EndpointB,
				[]oracletypes.GetPriceResponse{
					{
						Id: 0,
						Price: &oracletypes.QuotePrice{
							Price: priceBTC,
						},
					},
					{
						Id: 1,
						Price: &oracletypes.QuotePrice{
							Price: priceETH,
						},
					},
				},
			)
		}
	})
}

func (suite *KeeperTestSuite) sendFetchPrice(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	sender string, currencyIds []string,
) channeltypes.Packet {
	msgActivate := &types.MsgActivate{
		Authority:       sender,
		SourcePort:      fromEndpoint.ChannelConfig.PortID,
		SourceChannel:   fromEndpoint.ChannelID,
		TimeoutDuration: time.Minute,
	}

	res, err := fromEndpoint.Chain.SendMsgs(msgActivate)
	suite.NoError(err)

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.NoError(err)

	var data icqtypes.InterchainQueryPacketData
	err = suite.chainA.Codec.UnmarshalJSON(packet.GetData(), &data)
	suite.NoError(err)

	return packet
}

func (suite *KeeperTestSuite) receiveFetchPrice(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	packet channeltypes.Packet,
) []byte {
	var data icqtypes.InterchainQueryPacketData
	err := suite.chainA.Codec.UnmarshalJSON(packet.GetData(), &data)
	suite.NoError(err)

	// get proof of packet commitment from chainA
	err = toEndpoint.UpdateClient()
	suite.NoError(err)

	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := fromEndpoint.QueryProof(packetKey)

	recvMsg := channeltypes.NewMsgRecvPacket(
		packet, proof, proofHeight, toEndpoint.Chain.SenderAccount.GetAddress().String())
	res, err := toEndpoint.Chain.SendMsgs(recvMsg)
	suite.NoError(err) // message committed

	ack, err := ibctesting.ParseAckFromEvents(res.GetEvents())
	suite.NoError(err)
	suite.NotNil(ack)

	return ack
}

func (suite *KeeperTestSuite) validateFetchedPrices(
	fromEndpoint, toEndpoint *ibctesting.Endpoint,
	expectedCurrencyPrices []oracletypes.GetPriceResponse,
) {
	ctx := fromEndpoint.Chain.GetContext()
	ok := fromEndpoint.Chain.GetInitiaApp().OracleKeeper

	for _, currencyPrice := range expectedCurrencyPrices {
		cp, found := ok.GetCurrencyPairFromID(ctx, currencyPrice.Id)
		suite.True(found)
		price, err := ok.GetPriceForCurrencyPair(ctx, cp)
		suite.NoError(err)
		suite.Equal(currencyPrice.Price.Price, price.Price)
	}
}
