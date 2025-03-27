package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrTargetGasZero = errorsmod.Register(ModuleName, 2, "target gas is zero")
)
