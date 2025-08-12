package v2

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

// v2
type Hooks interface{}

// SendPacket Hooks
type OnSendPacketOverrideHooks interface {
	OnSendPacketOverride(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress) error
}
type OnSendPacketBeforeHooks interface {
	OnSendPacketBeforeHook(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress)
}
type OnSendPacketAfterHooks interface {
	OnSendPacketAfterHook(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress, err error)
}

// OnRecvPacket Hooks
type OnRecvPacketOverrideHooks interface {
	OnRecvPacketOverride(im IBCMiddleware, ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult
}
type OnRecvPacketBeforeHooks interface {
	OnRecvPacketBeforeHook(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress)
}
type OnRecvPacketAfterHooks interface {
	OnRecvPacketAfterHook(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress, result channeltypesv2.RecvPacketResult)
}

// OnTimeoutPacket Hooks
type OnTimeoutPacketOverrideHooks interface {
	OnTimeoutPacketOverride(im IBCMiddleware, ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) error
}
type OnTimeoutPacketBeforeHooks interface {
	OnTimeoutPacketBeforeHook(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress)
}
type OnTimeoutPacketAfterHooks interface {
	OnTimeoutPacketAfterHook(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress, err error)
}

// OnAcknowledgementPacket Hooks
type OnAcknowledgementPacketOverrideHooks interface {
	OnAcknowledgementPacketOverride(im IBCMiddleware, ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, acknowledgement []byte, payload channeltypesv2.Payload, relayer sdk.AccAddress) error
}
type OnAcknowledgementPacketBeforeHooks interface {
	OnAcknowledgementPacketBeforeHook(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, acknowledgement []byte, payload channeltypesv2.Payload, relayer sdk.AccAddress)
}
type OnAcknowledgementPacketAfterHooks interface {
	OnAcknowledgementPacketAfterHook(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, acknowledgement []byte, payload channeltypesv2.Payload, relayer sdk.AccAddress, err error)
}

// WriteAcknowledgement Hooks
type WriteAcknowledgementOverrideHooks interface {
	WriteAcknowledgementOverride(ctx sdk.Context, srcClientID string, sequence uint64, ack channeltypesv2.Acknowledgement) error
}
type WriteAcknowledgementBeforeHooks interface {
	WriteAcknowledgementBeforeHook(ctx sdk.Context, srcClientID string, sequence uint64, ack channeltypesv2.Acknowledgement)
}
type WriteAcknowledgementAfterHooks interface {
	WriteAcknowledgementAfterHook(ctx sdk.Context, srcClientID string, sequence uint64, ack channeltypesv2.Acknowledgement, err error)
}
