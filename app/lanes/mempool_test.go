package lanes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/math"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsign "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	signer_extraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	blockbase "github.com/skip-mev/block-sdk/v2/block/base"

	lanes "github.com/initia-labs/initia/app/lanes"
	"github.com/initia-labs/initia/app/params"
)

func Test_MempoolInsert(t *testing.T) {
	ctx := sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger()).WithConsensusParams(cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: 1000000,
			MaxGas:   1000000,
		},
	})

	signerExtractor := signer_extraction.NewDefaultAdapter()
	encodingConfig := params.MakeEncodingConfig()
	txEncoder := encodingConfig.TxConfig.TxEncoder()

	// cannot create mempool with negative ratio
	_, err := lanes.NewMempool(
		blockbase.NewDefaultTxPriority(),
		signerExtractor,
		1, // max txs
		math.LegacyNewDecFromInt(math.NewInt(-1)), // max block space
		txEncoder,
	)
	require.Error(t, err)

	// cannot create mempool with ratio greater than 1
	_, err = lanes.NewMempool(
		blockbase.NewDefaultTxPriority(),
		signerExtractor,
		1,                                        // max txs
		math.LegacyNewDecFromInt(math.NewInt(2)), // max block space
		txEncoder,
	)
	require.Error(t, err)

	// valid creation
	mempool, err := lanes.NewMempool(
		blockbase.NewDefaultTxPriority(),
		signerExtractor,
		1,                                    // max txs
		math.LegacyMustNewDecFromStr("0.01"), // max block space
		txEncoder,
	)
	require.NoError(t, err)

	priv, _, addr := testdata.KeyTestPubAddr()
	defaultSignMode, err := authsign.APISignModeToInternal(encodingConfig.TxConfig.SignModeHandler().DefaultMode())
	require.NoError(t, err)

	// valid gas limit
	txBuilder := encodingConfig.TxConfig.NewTxBuilder()
	txBuilder.SetGasLimit(10000)
	txBuilder.SetMemo("")
	txBuilder.SetMsgs(&banktypes.MsgSend{FromAddress: addr.String(), ToAddress: addr.String(), Amount: sdk.Coins{}})
	sigV2 := signing.SignatureV2{
		PubKey: priv.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  defaultSignMode,
			Signature: nil,
		},
		Sequence: 1,
	}
	err = txBuilder.SetSignatures(sigV2)
	require.NoError(t, err)
	err = mempool.Insert(ctx, txBuilder.GetTx())
	require.NoError(t, err)

	// high gas limit than max gas
	txBuilder.SetGasLimit(10001)
	err = mempool.Insert(ctx, txBuilder.GetTx())
	require.ErrorContains(t, err, "exceeds max lane gas limit")

	// rollback gas limit and set memo to exceed max block space
	txBuilder.SetGasLimit(10000)
	txBuilder.SetMemo(string(make([]byte, 10000)))
	err = mempool.Insert(ctx, txBuilder.GetTx())
	require.ErrorContains(t, err, "exceeds max lane size")
}
