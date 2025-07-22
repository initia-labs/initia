package keeper_test

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	storetypes "cosmossdk.io/store/types"

	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func TestVerifyAccountAbstractionSignature(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	moveKeeper := input.MoveKeeper

	moduleAddress, err := vmtypes.NewAccountAddress("0xcafe")
	require.NoError(t, err)

	// register authenticator
	err = moveKeeper.PublishModuleBundle(ctx, moduleAddress, vmtypes.NewModuleBundle([]vmtypes.Module{
		{Code: publicKeyAuthenticator},
	}...), movetypes.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	signerPub, signerPriv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// add authentication function
	_, _, originAddr := keyPubAddr()

	err = moveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(originAddr),
		vmtypes.StdAddress,
		"account_abstraction",
		"add_authentication_function",
		[]vmtypes.TypeTag{},
		[]string{"\"0xcafe\"", "\"public_key_authenticator\"", "\"authenticate\""},
	)
	require.NoError(t, err)

	// permit public key
	err = moveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		movetypes.ConvertSDKAddressToVMAddress(originAddr),
		moduleAddress,
		"public_key_authenticator",
		"permit_public_key",
		[]vmtypes.TypeTag{},
		[]string{fmt.Sprintf("\"%s\"", hex.EncodeToString(signerPub))},
	)
	require.NoError(t, err)

	// prepare authentication data
	hasher := sha3.New256()
	hasher.Write(publicKeyAuthenticator)
	digest := hasher.Sum(nil)
	digestBytes := digest[:]

	signature := ed25519.Sign(signerPriv, digestBytes)

	var authenticator []byte
	bcsEncodedPubBz, err := vmtypes.SerializeBytes([]byte(signerPub))
	require.NoError(t, err)

	bcsEncodedSignatureBz, err := vmtypes.SerializeBytes(signature)
	require.NoError(t, err)

	authenticator = append(authenticator, bcsEncodedPubBz...)
	authenticator = append(authenticator, bcsEncodedSignatureBz...)

	abstractionData := vmtypes.AbstractionData{
		FunctionInfo: vmtypes.FunctionInfo{
			ModuleAddress: moduleAddress,
			ModuleName:    "public_key_authenticator",
			FunctionName:  "authenticate",
		},
		AuthData: &vmtypes.AbstractionAuthData__V1{
			SigningMessageDigest: digestBytes,
			Authenticator:        authenticator,
		},
	}

	// execute authentication
	signer, err := moveKeeper.VerifyAccountAbstractionSignature(ctx, originAddr.String(), abstractionData)
	require.NoError(t, err)
	require.NotNil(t, signer)
	require.Equal(t, movetypes.ConvertSDKAddressToVMAddress(originAddr), *signer)

	// set gas limit to 10000 and execute again
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000))
	signer, err = moveKeeper.VerifyAccountAbstractionSignature(ctx, originAddr.String(), abstractionData)
	require.Error(t, err)
	require.Nil(t, signer)
}
