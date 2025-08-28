package tendermintattestor

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrInvalidAttestation      = errorsmod.Register(ModuleName, 1, "invalid attestation")
	ErrUnauthorizedAttestation = errorsmod.Register(ModuleName, 2, "unauthorized attestation")
)
