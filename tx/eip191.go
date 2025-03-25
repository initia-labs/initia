package tx

import (
	"context"
	"strconv"
	"strings"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/tx/signing/aminojson"
)

const EIP191MessagePrefix = "\x19Ethereum Signed Message:\n"

// SignModeEIP191Handler defines the SIGN_MODE_DIRECT SignModeHandler
type SignModeEIP191Handler struct {
	*aminojson.SignModeHandler
}

// NewSignModeEIP191Handler returns a new SignModeEIP191Handler.
func NewSignModeEIP191Handler(options aminojson.SignModeHandlerOptions) *SignModeEIP191Handler {
	return &SignModeEIP191Handler{
		SignModeHandler: aminojson.NewSignModeHandler(options),
	}
}

var _ signing.SignModeHandler = SignModeEIP191Handler{}

// Mode implements signing.SignModeHandler.Mode.
func (SignModeEIP191Handler) Mode() signingv1beta1.SignMode {
	return signingv1beta1.SignMode_SIGN_MODE_EIP_191 //nolint
}

// GetSignBytes implements SignModeHandler.GetSignBytes
func (h SignModeEIP191Handler) GetSignBytes(
	ctx context.Context, data signing.SignerData, txData signing.TxData,
) ([]byte, error) {
	aminoJSONBz, err := h.SignModeHandler.GetSignBytes(ctx, data, txData)
	if err != nil {
		return nil, err
	}

	return FormatEIP191Message(aminoJSONBz), nil
}

// FormatEIP191Message formats a message to be signed with EIP-191.
func FormatEIP191Message(msg []byte) []byte {
	return append(append(
		[]byte(EIP191MessagePrefix),
		[]byte(strconv.Itoa(len(msg)))...,
	), msg...)
}

// RemoveEIP191Prefix removes the EIP-191 prefix from a message.
func RemoveEIP191Prefix(msg []byte) []byte {
	idx := strings.Index(string(msg), "{")
	if idx == -1 {
		return msg
	}

	return msg[idx:]
}
