package hook

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

var _ ophosttypes.BridgeHook = BridgeHook{}

type BridgeHook struct {
	IBCChannelKeeper ChannelKeeper
	IBCPermKeeper    PermKeeper
}

type ChannelKeeper interface {
	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)
}

type PermKeeper interface {
	SetChannelRelayer(ctx sdk.Context, channel string, relayer sdk.AccAddress)
}

func NewBridgeHook(channelKeeper ChannelKeeper, permKeeper PermKeeper) BridgeHook {
	return BridgeHook{channelKeeper, permKeeper}
}

func (h BridgeHook) BridgeCreated(
	ctx sdk.Context,
	bridgeId uint64,
	bridgeConfig ophosttypes.BridgeConfig,
) error {
	channelID := string(bridgeConfig.Metadata)
	if channeltypes.IsValidChannelID(channelID) {
		if seq, ok := h.IBCChannelKeeper.GetNextSequenceSend(ctx, ibctransfertypes.PortID, channelID); !ok {
			return channeltypes.ErrChannelNotFound.Wrap("failed to register permissioned relayer")
		} else if seq != 1 {
			return channeltypes.ErrChannelExists.Wrap("cannot register permissioned relayer for the channel in use")
		}

		challenger, err := sdk.AccAddressFromBech32(bridgeConfig.Challenger)
		if err != nil {
			return err
		}

		// register challenger as channel relayer
		h.IBCPermKeeper.SetChannelRelayer(ctx, channelID, challenger)
	}

	return nil
}

func (h BridgeHook) BridgeChallengerUpdated(
	ctx sdk.Context,
	bridgeId uint64,
	bridgeConfig ophosttypes.BridgeConfig,
) error {
	channelID := string(bridgeConfig.Metadata)
	if channeltypes.IsValidChannelID(channelID) {
		challenger, err := sdk.AccAddressFromBech32(bridgeConfig.Challenger)
		if err != nil {
			return err
		}

		// update relayer to a new challenger
		h.IBCPermKeeper.SetChannelRelayer(ctx, channelID, challenger)
	}

	return nil
}

func (h BridgeHook) BridgeProposerUpdated(
	ctx sdk.Context,
	bridgeId uint64,
	bridgeConfig ophosttypes.BridgeConfig,
) error {
	return nil
}
