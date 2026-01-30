package app

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	cosmosante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/initia-labs/initia/abcipp"
	appante "github.com/initia-labs/initia/app/ante"
	dynamicfeeante "github.com/initia-labs/initia/x/dynamic-fee/ante"
	dynamicfeekeeper "github.com/initia-labs/initia/x/dynamic-fee/keeper"
)

func (app *InitiaApp) setupABCIPP(mempoolMaxTxs int) (
	sdkmempool.Mempool,
	sdk.AnteHandler,
	sdk.PrepareProposalHandler,
	sdk.ProcessProposalHandler,
	abcipp.CheckTx,
	error,
) {

	feeChecker := dynamicfeeante.NewMempoolFeeChecker(dynamicfeekeeper.NewAnteKeeper(app.DynamicFeeKeeper)).CheckTxFeeWithMinGasPrices
	feeCheckerWrapper := func(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
		freeFeeChecker := func() bool {
			for _, msg := range tx.GetMsgs() {
				switch msg.(type) {
				case *clienttypes.MsgUpdateClient:
				case *channeltypes.MsgTimeout:
				case *channeltypes.MsgAcknowledgement:
				default:
					return false
				}
			}
			return true
		}

		if !freeFeeChecker() {
			return feeChecker(ctx, tx)
		}

		// return fee without fee check
		feeTx, ok := tx.(sdk.FeeTx)
		if !ok {
			return nil, 0, errors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
		}

		return feeTx.GetFee(), 1 /* FIFO */, nil
	}

	anteHandler, err := appante.NewAnteHandler(
		appante.HandlerOptions{
			HandlerOptions: cosmosante.HandlerOptions{
				AccountKeeper:   app.AccountKeeper,
				BankKeeper:      app.BankKeeper,
				FeegrantKeeper:  app.FeeGrantKeeper,
				SignModeHandler: app.txConfig.SignModeHandler(),
				TxFeeChecker:    feeCheckerWrapper,
			},
			Codec:     app.appCodec,
			TxEncoder: app.txConfig.TxEncoder(),

			IBCkeeper:                app.IBCKeeper,
			DynamicFeeKeeper:         dynamicfeekeeper.NewAnteKeeper(app.DynamicFeeKeeper),
			AccountAbstractionKeeper: app.MoveKeeper,
		},
	)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	mempool := abcipp.NewPriorityMempool(
		abcipp.PriorityMempoolConfig{
			MaxTx:       mempoolMaxTxs,
			AnteHandler: anteHandler,
		}, app.TxEncode,
	)

	// start mempool cleaning worker
	mempool.StartCleaningWorker(app.BaseApp, app.AccountKeeper, abcipp.DefaultMempoolCleaningInterval)

	proposalHandler := abcipp.NewProposalHandler(
		app.Logger(),
		app.txConfig.TxDecoder(),
		app.txConfig.TxEncoder(),
		mempool,
		anteHandler,
	)

	checkTxHandler := abcipp.NewCheckTxHandler(
		app.Logger(),
		app.BaseApp,
		mempool,
		app.txConfig.TxDecoder(),
		app.BaseApp.CheckTx,
		feeCheckerWrapper,
	)

	return mempool, anteHandler, proposalHandler.PrepareProposalHandler(), proposalHandler.ProcessProposalHandler(), checkTxHandler.CheckTx, nil
}
