package types

import (
	errorsmod "cosmossdk.io/errors"
)

// IBC nft transfer sentinel errors
var (
	ErrInvalidPacketTimeout         = errorsmod.Register(ModuleName, 2, "invalid packet timeout")
	ErrInvalidClassIdForNftTransfer = errorsmod.Register(ModuleName, 3, "invalid class id for cross-chain nft transfer")
	ErrInvalidVersion               = errorsmod.Register(ModuleName, 4, "invalid ICS721 version")
	ErrInvalidClassId               = errorsmod.Register(ModuleName, 5, "invalid class id")
	ErrInvalidTokenIds              = errorsmod.Register(ModuleName, 6, "invalid token ids")
	ErrInvalidPacket                = errorsmod.Register(ModuleName, 7, "invalid non-fungible token packet")
	ErrTraceNotFound                = errorsmod.Register(ModuleName, 8, "class trace not found")
	ErrSendDisabled                 = errorsmod.Register(ModuleName, 9, "non-fungible token transfers from this chain are disabled")
	ErrReceiveDisabled              = errorsmod.Register(ModuleName, 10, "non-fungible token transfers to this chain are disabled")
	ErrMaxNftTransferChannels       = errorsmod.Register(ModuleName, 11, "max non-fungible token transfer channels")
)
