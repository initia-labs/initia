package abcipp

import (
	"fmt"

	cometabci "github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
)

// CheckTxHandler defines a CheckTx handler for ABCI++ CheckTx processing.
type CheckTxHandler struct {
	logger     log.Logger
	baseApp    BaseApp
	mempool    Mempool
	txDecoder  sdk.TxDecoder
	checkTx    CheckTx
	feeChecker ante.TxFeeChecker
}

// NewCheckTxHandler returns a new CheckTxHandler.
func NewCheckTxHandler(
	logger log.Logger,
	baseApp BaseApp,
	mempool Mempool,
	txDecoder sdk.TxDecoder,
	checkTx CheckTx,
	feeChecker ante.TxFeeChecker,
) *CheckTxHandler {
	return &CheckTxHandler{
		logger:     logger.With("module", "abcipp-checktx"),
		baseApp:    baseApp,
		mempool:    mempool,
		txDecoder:  txDecoder,
		checkTx:    checkTx,
		feeChecker: feeChecker,
	}
}

// CheckTx processes a CheckTx request from CometBFT.
func (h CheckTxHandler) CheckTx(req *cometabci.RequestCheckTx) (resp *cometabci.ResponseCheckTx, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			h.logger.Error("failed to check tx", "err", err)

			resp = sdkerrors.ResponseCheckTxWithEvents(
				fmt.Errorf("failed to check tx: %v", rec),
				0,
				0,
				nil,
				false,
			)
			err = fmt.Errorf("failed to check tx: %v", rec)
		}

	}()

	tx, err := h.txDecoder(req.Tx)
	if err != nil {
		return sdkerrors.ResponseCheckTxWithEvents(
			fmt.Errorf("failed to decode tx: %w", err),
			0,
			0,
			nil,
			false,
		), nil
	}

	isRecheck := req.Type == cometabci.CheckTxType_Recheck
	txInMempool := h.mempool.Contains(tx)

	// if the mode is ReCheck and the app's mempool does not contain the given tx, we fail
	// immediately, to purge the tx from the comet mempool.
	if isRecheck && !txInMempool {
		h.logger.Debug(
			"tx from comet mempool not found in app-side mempool",
			"tx", tx,
		)

		return sdkerrors.ResponseCheckTxWithEvents(
			fmt.Errorf("tx from comet mempool not found in app-side mempool"),
			0,
			0,
			nil,
			false,
		), nil
	}

	// baseApp.CheckTx will insert the tx into the mempool if valid
	resp, err = h.checkTx(req)

	// if re-check fails for a transaction, we'll need to explicitly purge the tx from
	// the app-side mempool
	if isInvalidCheckTxExecution(resp, err) && isRecheck && txInMempool {
		// remove the tx
		if err := h.mempool.Remove(tx); err != nil {
			h.logger.Debug(
				"failed to remove tx from app-side mempool when purging for re-check failure",
				"removal-err", err,
			)
		}
	}

	return
}

func isInvalidCheckTxExecution(resp *cometabci.ResponseCheckTx, checkTxErr error) bool {
	return resp == nil || resp.Code != 0 || checkTxErr != nil
}
