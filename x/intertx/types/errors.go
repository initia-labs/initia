package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrIBCAccountAlreadyExist = errorsmod.Register(ModuleName, 2, "interchain account already registered")
	ErrIBCAccountNotExist     = errorsmod.Register(ModuleName, 3, "interchain account not exist")
)
