package types

import "golang.org/x/crypto/sha3"

func ModuleBzToChecksum(moduleBz []byte) [32]byte {
	return sha3.Sum256(moduleBz)
}
