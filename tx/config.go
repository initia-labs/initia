package tx

import (
	"fmt"

	txsigning "cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/tx/signing/aminojson"
	"cosmossdk.io/x/tx/signing/direct"
	"cosmossdk.io/x/tx/signing/directaux"
	"cosmossdk.io/x/tx/signing/textual"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
)

// DefaultSignModes are the default sign modes enabled for protobuf transactions.
var DefaultSignModes = []signingtypes.SignMode{
	signingtypes.SignMode_SIGN_MODE_DIRECT,
	signingtypes.SignMode_SIGN_MODE_DIRECT_AUX,
	signingtypes.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
	signingtypes.SignMode_SIGN_MODE_EIP_191,
	// signingtypes.SignMode_SIGN_MODE_TEXTUAL is not enabled by default, as it requires a x/bank keeper or gRPC connection.
}

// NewTxConfig returns a new protobuf TxConfig using the provided ProtoCodec and sign modes. The
// first enabled sign mode will become the default sign mode.
//
// NOTE: Use NewTxConfigWithOptions to provide a custom signing handler in case the sign mode
// is not supported by default (eg: SignMode_SIGN_MODE_EIP_191), or to enable SIGN_MODE_TEXTUAL.
//
// We prefer to use depinject to provide client.TxConfig, but we permit this constructor usage. Within the SDK,
// this constructor is primarily used in tests, but also sees usage in app chains like:
// https://github.com/evmos/evmos/blob/719363fbb92ff3ea9649694bd088e4c6fe9c195f/encoding/config.go#L37
func NewTxConfig(protoCodec codec.Codec, enabledSignModes []signingtypes.SignMode,
	customSignModes ...txsigning.SignModeHandler,
) client.TxConfig {
	config := authtx.ConfigOptions{
		EnabledSignModes: enabledSignModes,
		CustomSignModes:  customSignModes,
	}

	var err error
	config.SigningHandler, err = NewSigningHandlerMap(config)
	if err != nil {
		panic(err)
	}

	txConfig, err := authtx.NewTxConfigWithOptions(protoCodec, config)
	if err != nil {
		panic(err)
	}

	return txConfig
}

// NewSigningHandlerMap returns a new txsigning.HandlerMap using the provided ConfigOptions.
// It is recommended to use types.InterfaceRegistry in the field ConfigOptions.FileResolver as shown in
// NewTxConfigWithOptions but this fn does not enforce it.
func NewSigningHandlerMap(configOpts authtx.ConfigOptions) (*txsigning.HandlerMap, error) {
	var err error
	if configOpts.SigningOptions == nil {
		configOpts.SigningOptions, err = authtx.NewDefaultSigningOptions()
		if err != nil {
			return nil, err
		}
	}
	if configOpts.SigningContext == nil {
		configOpts.SigningContext, err = txsigning.NewContext(*configOpts.SigningOptions)
		if err != nil {
			return nil, err
		}
	}

	signingOpts := configOpts.SigningOptions

	if len(configOpts.EnabledSignModes) == 0 {
		configOpts.EnabledSignModes = DefaultSignModes
	}

	lenSignModes := len(configOpts.EnabledSignModes)
	handlers := make([]txsigning.SignModeHandler, lenSignModes+len(configOpts.CustomSignModes))
	for i, m := range configOpts.EnabledSignModes {
		var err error
		switch m {
		case signingtypes.SignMode_SIGN_MODE_DIRECT:
			handlers[i] = &direct.SignModeHandler{}
		case signingtypes.SignMode_SIGN_MODE_DIRECT_AUX:
			handlers[i], err = directaux.NewSignModeHandler(directaux.SignModeHandlerOptions{
				TypeResolver:   signingOpts.TypeResolver,
				SignersContext: configOpts.SigningContext,
			})
			if err != nil {
				return nil, err
			}
		case signingtypes.SignMode_SIGN_MODE_LEGACY_AMINO_JSON:
			handlers[i] = aminojson.NewSignModeHandler(aminojson.SignModeHandlerOptions{
				FileResolver: signingOpts.FileResolver,
				TypeResolver: signingOpts.TypeResolver,
			})
		case signingtypes.SignMode_SIGN_MODE_TEXTUAL:
			handlers[i], err = textual.NewSignModeHandler(textual.SignModeOptions{
				CoinMetadataQuerier: configOpts.TextualCoinMetadataQueryFn,
				FileResolver:        signingOpts.FileResolver,
				TypeResolver:        signingOpts.TypeResolver,
			})
			if configOpts.TextualCoinMetadataQueryFn == nil {
				return nil, fmt.Errorf("cannot enable SIGN_MODE_TEXTUAL without a TextualCoinMetadataQueryFn")
			}
			if err != nil {
				return nil, err
			}
		case signingtypes.SignMode_SIGN_MODE_EIP_191:
			handlers[i] = NewSignModeEIP191Handler(aminojson.SignModeHandlerOptions{
				FileResolver: signingOpts.FileResolver,
				TypeResolver: signingOpts.TypeResolver,
			})
		}
	}
	for i, m := range configOpts.CustomSignModes {
		handlers[i+lenSignModes] = m
	}

	handler := txsigning.NewHandlerMap(handlers...)
	return handler, nil
}
