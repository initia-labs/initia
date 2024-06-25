package types

import (
	errorsmod "cosmossdk.io/errors"
)

// IBC Perm Errors
var (
	// ErrAlreadyTaken raised if the channel relayer is already taken
	ErrAlreadyTaken = errorsmod.Register(ModuleName, 2, "already taken")
)
