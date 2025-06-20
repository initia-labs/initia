package tx

import (
	"context"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/tx/signing/aminojson"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
)

const SignModeAccountAbstraction int32 = 900
const Signing_SignMode_ACCOUNT_ABSTRACTION = txsigning.SignMode(SignModeAccountAbstraction)
const Signingv1beta1_SignMode_ACCOUNT_ABSTRACTION = signingv1beta1.SignMode(SignModeAccountAbstraction)

type SignModeAccountAbstractionHandler struct {
	*aminojson.SignModeHandler
}

func NewSignModeAccountAbstractionHandler(options aminojson.SignModeHandlerOptions) *SignModeAccountAbstractionHandler {
	return &SignModeAccountAbstractionHandler{
		SignModeHandler: aminojson.NewSignModeHandler(options),
	}
}

var _ signing.SignModeHandler = SignModeAccountAbstractionHandler{}

// Mode implements signing.SignModeHandler.Mode.
func (SignModeAccountAbstractionHandler) Mode() signingv1beta1.SignMode {
	return signingv1beta1.SignMode(SignModeAccountAbstraction) //nolint
}

// GetSignBytes implements SignModeHandler.GetSignBytes
func (h SignModeAccountAbstractionHandler) GetSignBytes(
	ctx context.Context, data signing.SignerData, txData signing.TxData,
) ([]byte, error) {
	return h.SignModeHandler.GetSignBytes(ctx, data, txData)
}
