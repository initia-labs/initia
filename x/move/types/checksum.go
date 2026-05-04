package types

import "crypto/sha3"

func ModuleBzToChecksum(moduleBz []byte) [32]byte {
	return sha3.Sum256(moduleBz)
}
