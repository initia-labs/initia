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

	// ErrMalformedSenderCosmosMessage error raised when sender data is not signer address
	ErrMalformedSenderCosmosMessage = errorsmod.Register(ModuleName, 10, "malformed sender")

	// ErrInvalidRequest error raised when the request is invalid
	ErrInvalidRequest = errorsmod.Register(ModuleName, 11, "invalid request")

	// ErrInvalidQueryRequest error raised when the query request is invalid
	ErrInvalidQueryRequest = errorsmod.Register(ModuleName, 12, "invalid query request")

	// ErrNotSupportedCustomQuery error raised when the custom request is not supported
	ErrNotSupportedCustomQuery = errorsmod.Register(ModuleName, 13, "not supported custom query")

	// ErrNotSupportedStargateQuery error raised when the stargate request is not supported or accepted
	ErrNotSupportedStargateQuery = errorsmod.Register(ModuleName, 14, "not supported stargate query")

	// ErrAddressAlreadyTaken error raised when the address is already taken
	ErrAddressAlreadyTaken = errorsmod.Register(ModuleName, 15, "address already taken")

	// ErrScriptDisabled error raised when the script execution is disabled
	ErrScriptDisabled = errorsmod.Register(ModuleName, 16, "script execution disabled")

	// ErrVMQueryFailed error raised when the query execution failed
	ErrVMQueryFailed = errorsmod.Register(ModuleName, 17, "vm query failed")
)
