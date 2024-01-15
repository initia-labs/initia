package types

import (
	errorsmod "cosmossdk.io/errors"
)

// IBC transfer sentinel errors
var (
	ErrInvalidPacketTimeout = errorsmod.Register(ModuleName, 2, "invalid packet timeout")
	ErrInvalidVersion       = errorsmod.Register(ModuleName, 3, "invalid packet version")
	ErrFailedToFetchPrice   = errorsmod.Register(ModuleName, 4, "failed to fetch price")
	ErrMaxTransferChannels  = errorsmod.Register(ModuleName, 5, "max transfer channels")
	ErrInvalidMemo          = errorsmod.Register(ModuleName, 6, "invalid memo")
	ErrInvalidConsumerPort  = errorsmod.Register(ModuleName, 7, "invalid consumer port")
	ErrInvalidProviderPort  = errorsmod.Register(ModuleName, 8, "invalid provider port")
	ErrInvalidChannelFlow   = errorsmod.Register(ModuleName, 9, "invalid message sent to channel end")
	ErrInvalidCurrencyId    = errorsmod.Register(ModuleName, 10, "invalid currency id")
	ErrInvalidQuotePrice    = errorsmod.Register(ModuleName, 11, "invalid quote price")
	ErrInvalidOutgoingData  = errorsmod.Register(ModuleName, 12, "invalid outgoing packet data")
)
