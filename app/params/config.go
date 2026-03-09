package params

import (
	"cosmossdk.io/x/tx/signing/aminojson"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	initiatx "github.com/initia-labs/initia/tx"
)

type config struct {
	client.TxConfig
	allowQueued bool
}

type queuedTxBuilder struct {
	client.TxBuilder
	extBuilder client.ExtendedTxBuilder
}

func (b queuedTxBuilder) SetExtensionOptions(extOpts ...*codectypes.Any) {
	if len(extOpts) == 0 {
		b.extBuilder.SetExtensionOptions(&codectypes.Any{TypeUrl: initiatx.ExtensionOptionQueuedTxTypeURL})
		return
	}
	b.extBuilder.SetExtensionOptions(extOpts...)
}

func (c config) NewTxBuilder() client.TxBuilder {
	txBuilder := c.TxConfig.NewTxBuilder()
	if !c.allowQueued {
		return txBuilder
	}

	// Inject queued extension options at tx-build time, before sign bytes are generated.
	extBuilder, ok := txBuilder.(client.ExtendedTxBuilder)
	if !ok {
		return txBuilder
	}
	extBuilder.SetExtensionOptions(&codectypes.Any{TypeUrl: initiatx.ExtensionOptionQueuedTxTypeURL})
	return queuedTxBuilder{
		TxBuilder:  txBuilder,
		extBuilder: extBuilder,
	}
}

// WithAllowQueuedTxConfig toggles queued extension injection for this app's tx config.
// If a non-app tx config is provided, it is returned unchanged.
func WithAllowQueuedTxConfig(txConfig client.TxConfig, allowQueued bool) client.TxConfig {
	cfg, ok := txConfig.(config)
	if !ok {
		return txConfig
	}

	cfg.allowQueued = allowQueued
	return cfg
}

func CreateTxConfig(protoCodec codec.ProtoCodecMarshaler) client.TxConfig {
	signingOptions, err := authtx.NewDefaultSigningOptions()
	if err != nil {
		panic(err)
	}

	return config{
		TxConfig: authtx.NewTxConfig(
			protoCodec,
			authtx.DefaultSignModes,
			initiatx.NewSignModeEIP191Handler(aminojson.SignModeHandlerOptions{
				FileResolver: signingOptions.FileResolver,
				TypeResolver: signingOptions.TypeResolver,
			}),
			initiatx.NewSignModeAccountAbstractionHandler(),
		),
	}
}

func (c config) TxDecoder() sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, error) {
		if tx, err := c.TxConfig.TxDecoder()(txBytes); err != nil {
			txBuilder := c.NewTxBuilder()
			txBuilder.SetMemo("decode failed tx")

			return txBuilder.GetTx(), nil
		} else {
			return tx, err
		}
	}
}
