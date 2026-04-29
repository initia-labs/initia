package types

import (
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

func GetReceivedTokenDenom(packet channeltypes.Packet, data transfertypes.FungibleTokenPacketData) string {
	denom := transfertypes.ExtractDenomFromPath(data.Denom)
	if denom.HasPrefix(packet.GetSourcePort(), packet.GetSourceChannel()) {
		// receiver chain is the source: strip the prefix added by the sender.
		denom.Trace = denom.Trace[1:]
	} else {
		// sender chain is the source: prepend the destination hop.
		denom.Trace = append([]transfertypes.Hop{transfertypes.NewHop(packet.GetDestPort(), packet.GetDestChannel())}, denom.Trace...)
	}

	return denom.IBCDenom()
}
