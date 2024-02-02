package types

import (
	errorsmod "cosmossdk.io/errors"
)

// IBC nft transfer sentinel errors
var (
	ErrInvalidPacketTimeout    = errorsmod.Register(ModuleName, 2, "invalid packet timeout")
	ErrInvalidVersion          = errorsmod.Register(ModuleName, 3, "invalid fetchprice version")
	ErrInvalidPacket           = errorsmod.Register(ModuleName, 4, "invalid fetchprice packet")
	ErrFetchDisabled           = errorsmod.Register(ModuleName, 5, "price fetching from this chain is disabled")
	ErrFetchAlreadyActivated   = errorsmod.Register(ModuleName, 6, "price fetching from this chain is already activated")
	ErrInvalidFetchPricePortID = errorsmod.Register(ModuleName, 7, "invalid fetchprice port id")
	ErrInvalidICQPortID        = errorsmod.Register(ModuleName, 8, "invalid ICQ port id")
	ErrInvalidChannelFlow      = errorsmod.Register(ModuleName, 9, "invalid message sent to channel end")
)
