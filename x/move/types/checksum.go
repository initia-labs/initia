package types

import (
	"crypto/sha256"
)

func ModuleBzToChecksum(moduleBz []byte) [32]byte {
	return sha256.Sum256(moduleBz)
}
