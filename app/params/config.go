package params

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	blockproposalstypes "github.com/skip-mev/block-sdk/block/proposals/types"
)

type config struct {
	client.TxConfig
}

func NewTxConfig(protoCodec codec.ProtoCodecMarshaler, enabledSignModes []signingtypes.SignMode) client.TxConfig {
	return config{authtx.NewTxConfig(protoCodec, enabledSignModes)}
}

func (g config) TxDecoder() sdk.TxDecoder {
	return func(txBytes []byte) (sdk.Tx, error) {
		if tx, err := g.TxConfig.TxDecoder()(txBytes); err != nil {

			// convert skip's custom message to empty tx
			var metaData blockproposalstypes.ProposalInfo
			if err := metaData.Unmarshal(txBytes); err == nil {
				txbuilder := g.NewTxBuilder()
				txbuilder.SetMemo("Tx is for BlockSDK")

				return txbuilder.GetTx(), err
			}

			return tx, err
		} else {
			return tx, err
		}
	}
}
