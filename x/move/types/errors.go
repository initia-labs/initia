package types

import (
	errorsmod "cosmossdk.io/errors"
)

// Move Errors
var (
	// ErrEmpty error for the empty content
	ErrEmpty = errorsmod.Register(ModuleName, 2, "empty")

	// ErrLimit error for the content that exceeds a limit
	ErrLimit = errorsmod.Register(ModuleName, 3, "exceeds limit")

	// ErrMalformedDenom error for the invalid denom format
	ErrMalformedDenom = errorsmod.Register(ModuleName, 4, "malformed denom")

	// ErrMalformedClassId error for the invalid denom format
	ErrMalformedClassId = errorsmod.Register(ModuleName, 5, "malformed class id")

	// ErrMalformedStructTag error for the invalid denom format
	ErrMalformedStructTag = errorsmod.Register(ModuleName, 6, "malformed struct tag")

	// ErrInvalidDexConfig error for invalid dex config
	ErrInvalidDexConfig = errorsmod.Register(ModuleName, 7, "invalid dex config value")

	// ErrUnauthorized error raised when wrong admin try to upgrade module
	ErrUnauthorized = errorsmod.Register(ModuleName, 8, "unauthorized")

	// ErrNotSupportedCosmosMessage error raised when the returned cosmos messages are not supported
	ErrNotSupportedCosmosMessage = errorsmod.Register(ModuleName, 9, "malformed cosmos message")

	// ErrInvalidRequest error raised when the request is invalid
	ErrInvalidRequest = errorsmod.Register(ModuleName, 10, "invalid request")
)
