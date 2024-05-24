package params

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/tx"
)

type config struct {
	client.TxConfig
}

func NewClientTxConfig(protoCodec codec.ProtoCodecMarshaler) client.TxConfig {
	return config{tx.NewTxConfig(protoCodec, tx.DefaultSignModes)}
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
