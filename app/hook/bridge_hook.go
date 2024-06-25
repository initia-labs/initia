package hook

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	permtypes "github.com/initia-labs/initia/x/ibc/perm/types"
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
	HasPermission(ctx context.Context, portID, channelID string, relayer sdk.AccAddress) (bool, error)
	SetPermissionedRelayers(ctx context.Context, portID, channelID string, relayers []sdk.AccAddress) error
	GetPermissionedRelayers(ctx context.Context, portID, channelID string) ([]sdk.AccAddress, error)
}

func NewBridgeHook(channelKeeper ChannelKeeper, permKeeper PermKeeper, ac address.Codec) BridgeHook {
	return BridgeHook{channelKeeper, permKeeper, ac}
}

func (h BridgeHook) BridgeCreated(
	ctx context.Context,
	bridgeId uint64,
	bridgeConfig ophosttypes.BridgeConfig,
) error {
	hasPermChannels, metadata := hasPermChannels(bridgeConfig.Metadata)
	if !hasPermChannels {
		return nil
	}

	challenger, err := h.ac.StringToBytes(bridgeConfig.Challenger)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, permChannel := range metadata.PermChannels {
		portID, channelID := permChannel.PortID, permChannel.ChannelID
		if seq, ok := h.IBCChannelKeeper.GetNextSequenceSend(sdkCtx, portID, channelID); !ok {
			return channeltypes.ErrChannelNotFound.Wrap("failed to register permissioned relayer")
		} else if seq != 1 {
			return channeltypes.ErrChannelExists.Wrap("cannot register permissioned relayer for the channel in use")
		}

		// check if the channel has a permissioned relayer
		if _, err := h.IBCPermKeeper.GetPermissionedRelayers(ctx, portID, channelID); err == nil {
			return permtypes.ErrAlreadyTaken.Wrap("failed to claim permissioned relayer")
		} else if !errors.Is(err, collections.ErrNotFound) {
			return err
		}

		// register challenger as channel relayer
		if err = h.IBCPermKeeper.SetPermissionedRelayers(sdkCtx, portID, channelID, []sdk.AccAddress{challenger}); err != nil {
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
	hasPermChannels, metadata := hasPermChannels(bridgeConfig.Metadata)
	if !hasPermChannels {
		return nil
	}

	challenger, err := h.ac.StringToBytes(bridgeConfig.Challenger)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, permChannel := range metadata.PermChannels {
		portID, channelID := permChannel.PortID, permChannel.ChannelID

		// update relayer to a new challenger
		if err = h.IBCPermKeeper.SetPermissionedRelayers(sdkCtx, portID, channelID, []sdk.AccAddress{challenger}); err != nil {
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

// BridgeBatchInfoUpdated implements types.BridgeHook.
func (h BridgeHook) BridgeBatchInfoUpdated(
	ctx context.Context,
	bridgeId uint64,
	bridgeConfig ophosttypes.BridgeConfig,
) error {
	return nil
}

func (h BridgeHook) BridgeMetadataUpdated(
	ctx context.Context,
	bridgeId uint64,
	bridgeConfig ophosttypes.BridgeConfig,
) error {
	hasPermChannels, metadata := hasPermChannels(bridgeConfig.Metadata)
	if !hasPermChannels {
		return nil
	}

	challenger, err := h.ac.StringToBytes(bridgeConfig.Challenger)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, permChannel := range metadata.PermChannels {
		portID, channelID := permChannel.PortID, permChannel.ChannelID

		// check if the relayer is already registered as a permissioned relayer
		if hasPermission, err := h.IBCPermKeeper.HasPermission(ctx, portID, channelID, challenger); err != nil {
			return err
		} else if hasPermission {
			continue
		}

		if seq, ok := h.IBCChannelKeeper.GetNextSequenceSend(sdkCtx, portID, channelID); !ok {
			return channeltypes.ErrChannelNotFound.Wrap("failed to register permissioned relayer")
		} else if seq != 1 {
			return channeltypes.ErrChannelExists.Wrap("cannot register permissioned relayer for the channel in use")
		}

		// register challenger as channel relayer
		if err = h.IBCPermKeeper.SetPermissionedRelayers(sdkCtx, portID, channelID, []sdk.AccAddress{challenger}); err != nil {
			return err
		}
	}

	return nil
}
