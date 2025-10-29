package tx

import (
	"context"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/tx/signing/direct"

	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
)

const SignModeAccountAbstraction int32 = 0xAA // 170
const Signing_SignMode_ACCOUNT_ABSTRACTION = txsigning.SignMode(SignModeAccountAbstraction)
const Signingv1beta1_SignMode_ACCOUNT_ABSTRACTION = signingv1beta1.SignMode(SignModeAccountAbstraction)

type SignModeAccountAbstractionHandler struct {
	*direct.SignModeHandler
}

func NewSignModeAccountAbstractionHandler() *SignModeAccountAbstractionHandler {
	return &SignModeAccountAbstractionHandler{
		SignModeHandler: &direct.SignModeHandler{},
	}
}

var _ signing.SignModeHandler = SignModeAccountAbstractionHandler{}

// Mode implements signing.SignModeHandler.Mode.
func (SignModeAccountAbstractionHandler) Mode() signingv1beta1.SignMode {
	return signingv1beta1.SignMode(SignModeAccountAbstraction)
}

// GetSignBytes implements SignModeHandler.GetSignBytes
func (h SignModeAccountAbstractionHandler) GetSignBytes(
	ctx context.Context, data signing.SignerData, txData signing.TxData,
) ([]byte, error) {
	return h.SignModeHandler.GetSignBytes(ctx, data, txData)
}
