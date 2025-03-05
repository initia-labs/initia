package params

import (
	"cosmossdk.io/x/tx/signing/aminojson"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	tx "github.com/initia-labs/initia/v1/tx"
)

type config struct {
	client.TxConfig
}

func NewClientTxConfig(protoCodec codec.ProtoCodecMarshaler) client.TxConfig {
	signingOptions, err := authtx.NewDefaultSigningOptions()
	if err != nil {
		panic(err)
	}

	return config{
		authtx.NewTxConfig(
			protoCodec,
			authtx.DefaultSignModes,
			tx.NewSignModeEIP191Handler(aminojson.SignModeHandlerOptions{
				FileResolver: signingOptions.FileResolver,
				TypeResolver: signingOptions.TypeResolver,
			}),
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
