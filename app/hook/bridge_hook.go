package hook

import (
	"context"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

var _ ophosttypes.BridgeHook = BridgeHook{}

type BridgeHook struct {
	IBCChannelKeeper ChannelKeeper
	IBCPermKeeper    PermKeeper
	ac               address.Codec
}

type ChannelKeeper interface {
	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)
}

type PermKeeper interface {
	SetChannelRelayer(ctx context.Context, channel string, relayer sdk.AccAddress) error
}

func NewBridgeHook(channelKeeper ChannelKeeper, permKeeper PermKeeper, ac address.Codec) BridgeHook {
	return BridgeHook{channelKeeper, permKeeper, ac}
}

func (h BridgeHook) BridgeCreated(
	ctx context.Context,
	bridgeId uint64,
	bridgeConfig ophosttypes.BridgeConfig,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	channelID := string(bridgeConfig.Metadata)
	if channeltypes.IsValidChannelID(channelID) {
		if seq, ok := h.IBCChannelKeeper.GetNextSequenceSend(sdkCtx, ibctransfertypes.PortID, channelID); !ok {
			return channeltypes.ErrChannelNotFound.Wrap("failed to register permissioned relayer")
		} else if seq != 1 {
			return channeltypes.ErrChannelExists.Wrap("cannot register permissioned relayer for the channel in use")
		}

		challenger, err := h.ac.StringToBytes(bridgeConfig.Challenger)
		if err != nil {
			return err
		}

		// register challenger as channel relayer
		if err = h.IBCPermKeeper.SetChannelRelayer(sdkCtx, channelID, challenger); err != nil {
			return err
		}
	}

	return nil
}

func (h BridgeHook) BridgeChallengerUpdated(
	ctx context.Context,
	bridgeId uint64,
	bridgeConfig ophosttypes.BridgeConfig,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	channelID := string(bridgeConfig.Metadata)
	if channeltypes.IsValidChannelID(channelID) {
		challenger, err := h.ac.StringToBytes(bridgeConfig.Challenger)
		if err != nil {
			return err
		}

		// update relayer to a new challenger
		if err = h.IBCPermKeeper.SetChannelRelayer(sdkCtx, channelID, challenger); err != nil {
			return err
		}
	}

	return nil
}

func (h BridgeHook) BridgeProposerUpdated(
	ctx context.Context,
	bridgeId uint64,
	bridgeConfig ophosttypes.BridgeConfig,
) error {
	return nil
}
