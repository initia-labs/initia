package keeper

import (
	"strings"

	metrics "github.com/hashicorp/go-metrics"

	"cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	coretypes "github.com/cosmos/ibc-go/v8/modules/core/types"

	consumertypes "github.com/initia-labs/initia/x/ibc/fetchprice/consumer/types"
	"github.com/initia-labs/initia/x/ibc/fetchprice/types"
)

// sendFetchPrice handles fetchprice sending logic.
func (k Keeper) sendFetchPrice(
	ctx sdk.Context,
	sourcePort,
	sourceChannel string,
	currencyIds []string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	memo string,
) (uint64, error) {
	if len(currencyIds) == 0 {
		return 0, errors.Wrap(types.ErrInvalidCurrencyId, "empty currency ids")
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

	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePort, sourceChannel))
	if !ok {
		return 0, errors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	labels := []metrics.Label{
		telemetry.NewLabel(coretypes.LabelDestinationPort, destinationPort),
		telemetry.NewLabel(coretypes.LabelDestinationChannel, destinationChannel),
	}

	packetData := types.FetchPricePacketData{
		CurrencyIds: currencyIds,
		Memo:        memo,
	}
	if err := packetData.ValidateBasic(); err != nil {
		return 0, err
	}

	if _, err := k.ics4Wrapper.SendPacket(
		ctx, channelCap, sourcePort, sourceChannel,
		timeoutHeight, timeoutTimestamp, packetData.GetBytes(),
	); err != nil {
		return 0, err
	}

	defer func() {
		if len(currencyIds) > 0 {
			telemetry.SetGaugeWithLabels(
				[]string{"tx", "msg", "ibc", "fetchprice"},
				float32(len(currencyIds)),
				[]metrics.Label{
					telemetry.NewLabel(types.LabelCurrencyIds, strings.Join(currencyIds, ",")),
				},
			)
		}

		telemetry.IncrCounterWithLabels(
			[]string{"ibc", consumertypes.SubModuleName, "fetchprice"},
			1,
			labels,
		)
	}()

	return sequence, nil
}
