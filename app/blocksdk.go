package app

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	cosmosante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	// this line is used by starport scaffolding # stargate/app/moduleImport

	appante "github.com/initia-labs/initia/app/ante"
	applanes "github.com/initia-labs/initia/app/lanes"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"

	// block-sdk dependencies

	blockabci "github.com/skip-mev/block-sdk/v2/abci"
	blockchecktx "github.com/skip-mev/block-sdk/v2/abci/checktx"
	signer_extraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block"
	blockbase "github.com/skip-mev/block-sdk/v2/block/base"
	mevlane "github.com/skip-mev/block-sdk/v2/lanes/mev"
)

func setupBlockSDK(
	app *InitiaApp,
	mempoolMaxTxs int,
) (
	mempool.Mempool,
	sdk.AnteHandler,
	blockchecktx.CheckTx,
	sdk.PrepareProposalHandler,
	sdk.ProcessProposalHandler,
	error,
) {
	// initialize and set the InitiaApp mempool. The current mempool will be the
	// x/auction module's mempool which will extract the top bid from the current block's auction
	// and insert the txs at the top of the block spots.
	signerExtractor := signer_extraction.NewDefaultAdapter()

	systemLane := applanes.NewSystemLane(blockbase.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.01"),
		MaxTxs:          1,
		SignerExtractor: signerExtractor,
	}, applanes.RejectMatchHandler())

	factory := mevlane.NewDefaultAuctionFactory(app.txConfig.TxDecoder(), signerExtractor)
	mevLane := mevlane.NewMEVLane(blockbase.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.09"),
		MaxTxs:          100,
		SignerExtractor: signerExtractor,
	}, factory, factory.MatchHandler())

	freeLane := applanes.NewFreeLane(blockbase.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.1"),
		MaxTxs:          100,
		SignerExtractor: signerExtractor,
	}, applanes.FreeLaneMatchHandler())

	defaultLane := applanes.NewDefaultLane(blockbase.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.8"),
		MaxTxs:          mempoolMaxTxs,
		SignerExtractor: signerExtractor,
	})

	lanes := []block.Lane{systemLane, mevLane, freeLane, defaultLane}
	mempool, err := block.NewLanedMempool(app.Logger(), lanes)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	anteHandler, err := appante.NewAnteHandler(
		appante.HandlerOptions{
			HandlerOptions: cosmosante.HandlerOptions{
				AccountKeeper:   app.AccountKeeper,
				BankKeeper:      app.BankKeeper,
				FeegrantKeeper:  app.FeeGrantKeeper,
				SignModeHandler: app.txConfig.SignModeHandler(),
			},
			IBCkeeper:     app.IBCKeeper,
			MoveKeeper:    movekeeper.NewDexKeeper(app.MoveKeeper),
			Codec:         app.appCodec,
			TxEncoder:     app.txConfig.TxEncoder(),
			AuctionKeeper: *app.AuctionKeeper,
			MevLane:       mevLane,
			FreeLane:      freeLane,
		},
	)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// set ante handler to lanes
	opt := []blockbase.LaneOption{
		blockbase.WithAnteHandler(anteHandler),
	}
	systemLane.(*blockbase.BaseLane).WithOptions(
		opt...,
	)
	mevLane.WithOptions(
		opt...,
	)
	freeLane.(*blockbase.BaseLane).WithOptions(
		opt...,
	)
	defaultLane.(*blockbase.BaseLane).WithOptions(
		opt...,
	)

	mevCheckTx := blockchecktx.NewMEVCheckTxHandler(
		app.BaseApp,
		app.txConfig.TxDecoder(),
		mevLane,
		anteHandler,
		app.BaseApp.CheckTx,
	)
	checkTxHandler := blockchecktx.NewMempoolParityCheckTx(
		app.Logger(), mempool,
		app.txConfig.TxDecoder(), mevCheckTx.CheckTx(),
	)
	checkTx := checkTxHandler.CheckTx()

	proposalHandler := blockabci.NewProposalHandler(
		app.Logger(),
		app.txConfig.TxDecoder(),
		app.txConfig.TxEncoder(),
		mempool,
	)

	prepareProposalHandler := proposalHandler.PrepareProposalHandler()
	processProposalHandler := proposalHandler.ProcessProposalHandler()

	return mempool, anteHandler, checkTx, prepareProposalHandler, processProposalHandler, nil
}
