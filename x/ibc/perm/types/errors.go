package types

import (
	errorsmod "cosmossdk.io/errors"
)

// Move Errors
var (
	// ErrInvalidHaltState error for the invalid halt state
	ErrInvalidHaltState = errorsmod.Register(ModuleName, 2, "invalid halt state")
)
