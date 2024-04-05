package keeper

import (
	"strings"

	metrics "github.com/hashicorp/go-metrics"

	"cosmossdk.io/errors"

	abcitypes "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	icqtypes "github.com/cosmos/ibc-apps/modules/async-icq/v8/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	coretypes "github.com/cosmos/ibc-go/v8/modules/core/types"

	oracletypes "github.com/skip-mev/slinky/x/oracle/types"

	"github.com/initia-labs/initia/x/ibc/fetchprice/types"
)

func (k Keeper) sendICQ(
	ctx sdk.Context,
	sourcePort,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
) (uint64, error) {
	// if fetch is not enabled, then just skip the operations.
	if ok, err := k.GetFetchEnabled(ctx); err != nil {
		return 0, err
	} else if !ok {
		return 0, nil
	}

	// if fetch is not activated, then just skip the operations.
	if ok, err := k.GetFetchActivated(ctx); err != nil {
		return 0, err
	} else if !ok {
		return 0, nil
	}

	// oracle currency pairs to query
	pairs := k.oracleKeeper.GetAllCurrencyPairs(ctx)
	if len(pairs) == 0 {
		return 0, nil
	}

	sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, errors.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", sourcePort, sourceChannel)
	}

	destinationPort := sourceChannelEnd.GetCounterparty().GetPortID()
	destinationChannel := sourceChannelEnd.GetCounterparty().GetChannelID()

	// get the next sequence
	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, errors.Wrapf(
			channeltypes.ErrSequenceSendNotFound,
			"source port: %s, source channel: %s", sourcePort, sourceChannel,
		)
	}

	// begin createOutgoingPacket logic
	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePort, sourceChannel))
	if !ok {
		return 0, errors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	labels := []metrics.Label{
		telemetry.NewLabel(coretypes.LabelDestinationPort, destinationPort),
		telemetry.NewLabel(coretypes.LabelDestinationChannel, destinationChannel),
	}

	// TODO change this to GetPrices
	pairIds := make([]string, len(pairs))
	reqs := make([]abcitypes.RequestQuery, len(pairs))
	for i, pair := range pairs {

		pairIds[i] = pair.String()

		// TODO change this to GetPrices
		reqs[i] = abcitypes.RequestQuery{
			Path: "/slinky.oracle.v1.Query/GetPrice",
			Data: k.cdc.MustMarshal(&oracletypes.GetPriceRequest{
				CurrencyPair: pair,
			}),
		}
	}

	data, err := icqtypes.SerializeCosmosQuery(reqs)
	if err != nil {
		return 0, err
	}
	packetData := icqtypes.InterchainQueryPacketData{
		Data: data,
	}

	if _, err := k.ics4Wrapper.SendPacket(
		ctx, channelCap, sourcePort, sourceChannel,
		timeoutHeight, timeoutTimestamp, packetData.GetBytes(),
	); err != nil {
		return 0, err
	}

	defer func() {
		if len(pairs) > 0 {
			telemetry.SetGaugeWithLabels(
				[]string{"tx", "msg", "ibc", "fetchprice"},
				float32(len(pairs)),
				[]metrics.Label{
					telemetry.NewLabel(types.LabelCurrencyIds, strings.Join(pairIds, ",")),
				},
			)
		}

		telemetry.IncrCounterWithLabels(
			[]string{"ibc", types.ModuleName, "send"},
			1,
			labels,
		)
	}()

	return sequence, nil
}

// If the ack is error, then deactivate the fetchprice routine.
func (k Keeper) OnAcknowledgementPacketError(ctx sdk.Context) error {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	params.FetchActivated = false
	return k.Params.Set(ctx, params)
}

// if the ack is success, then update the prices and start the next fetchprice routine.
func (k Keeper) OnAcknowledgementPacketSuccess(ctx sdk.Context, packet channeltypes.Packet, ack icqtypes.InterchainQueryPacketAck) error {
	timeout, err := k.GetTimeoutDuration(ctx)
	if err != nil {
		return err
	}

	var icqPacketData icqtypes.InterchainQueryPacketData
	if err := k.cdc.UnmarshalJSON(packet.Data, &icqPacketData); err != nil {
		return err
	}

	reqs, err := icqtypes.DeserializeCosmosQuery(icqPacketData.Data)
	if err != nil {
		return err
	}

	resps, err := icqtypes.DeserializeCosmosResponse(ack.Data)
	if err != nil {
		k.Logger(ctx).Error("failed to parse icq response to cosmos responses")
		return err
	}

	// store fetched prices to store
	for i, resp := range resps {
		if resp.Code != abcitypes.CodeTypeOK {
			continue
		}

		var oracleReq oracletypes.GetPriceRequest
		if err := k.cdc.Unmarshal(reqs[i].Data, &oracleReq); err != nil {
			k.Logger(ctx).Error("failed to parse icq request")
			return err
		}

		var oracleRes oracletypes.GetPriceResponse
		if err := k.cdc.Unmarshal(resp.Value, &oracleRes); err != nil {
			k.Logger(ctx).Error("failed to parse icq response to oracle price response")
			return err
		}

		if oracleRes.Price == nil {
			continue
		}

		cp := oracleReq.GetCurrencyPair()

		// ICQ connection is UNORDERED, so old query response can be relayed
		// later than latest one. To prevent overwrite the latest price info
		// we always check the price updated block timestamp is latest or not.
		if storedPrice, err := k.oracleKeeper.GetPriceForCurrencyPair(ctx, cp); err == nil {
			if storedPrice.BlockTimestamp.After(oracleRes.Price.BlockTimestamp) {
				continue
			}
		}

		if err := k.oracleKeeper.SetPriceForCurrencyPair(ctx, cp, *oracleRes.Price); err != nil {
			continue
		}
	}

	// send ICQ packet
	if _, err = k.sendICQ(
		ctx,
		packet.SourcePort,
		packet.SourceChannel,
		clienttypes.ZeroHeight(),
		uint64(ctx.BlockTime().Add(timeout).UnixNano()),
	); err != nil {
		return err
	}

	return nil
}

// At the timeout, just start the next fetchprice routine.
func (k Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	timeout, err := k.GetTimeoutDuration(ctx)
	if err != nil {
		return err
	}

	// send ICQ packet
	if _, err = k.sendICQ(
		ctx,
		packet.SourcePort,
		packet.SourceChannel,
		clienttypes.ZeroHeight(),
		uint64(ctx.BlockTime().Add(timeout).UnixNano()),
	); err != nil {
		return err
	}

	return nil
}
