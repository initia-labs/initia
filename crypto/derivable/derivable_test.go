package derivable

import (
	"testing"

	vmtypes "github.com/initia-labs/movevm/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestDerivablePubKey(t *testing.T) {
	pubKey := NewPubKey("0x1", "test_module", "test_function", []byte{0x1, 0x2, 0x3})

	pubKeyBytes := pubKey.Bytes()

	fInfo, err := vmtypes.NewFunctionInfo("0x1", "test_module", "test_function")
	require.NoError(t, err)

	fInfoBytes, err := fInfo.BcsSerialize()
	require.NoError(t, err)

	abstractPubkeyBytes, err := vmtypes.SerializeBytes([]byte{0x1, 0x2, 0x3})
	require.NoError(t, err)

	expectedBytes := make([]byte, 0, len(fInfoBytes)+len(abstractPubkeyBytes)+1)
	expectedBytes = append(expectedBytes, fInfoBytes...)
	expectedBytes = append(expectedBytes, abstractPubkeyBytes...)
	expectedBytes = append(expectedBytes, 0x5)

	require.Equal(t, expectedBytes, pubKeyBytes)

	address := pubKey.Address()
	hasher := sha3.New256()
	hasher.Write(pubKeyBytes)
	hash := hasher.Sum(nil)

	require.Equal(t, address.Bytes(), hash[:])
}
