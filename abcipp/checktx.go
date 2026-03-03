package abcipp

import (
	"fmt"
	"time"

	cometabci "github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
)

const slowCheckTxThreshold = 200 * time.Millisecond

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
	start := time.Now()
	txHash := TxHash(req.Tx)
	isRecheck := req.Type == cometabci.CheckTxType_Recheck
	txInMempool := false
	sender, sequence := "", uint64(0)
	decodeDur := time.Duration(0)
	containsDur := time.Duration(0)
	baseCheckDur := time.Duration(0)
	removeDur := time.Duration(0)

	defer func() {
		if rec := recover(); rec != nil {
			h.logger.Error("failed to check tx", "err", rec)

			resp = sdkerrors.ResponseCheckTxWithEvents(
				fmt.Errorf("failed to check tx: %v", rec),
				0,
				0,
				nil,
				false,
			)
			err = nil
		}

		totalDur := time.Since(start)
		respCode := int64(0)
		if resp != nil {
			respCode = int64(resp.Code)
		}
		shouldLog := totalDur >= slowCheckTxThreshold || err != nil || respCode != 0
		if shouldLog {
			h.logger.Info(
				"checktx trace",
				"tx_hash", txHash,
				"is_recheck", isRecheck,
				"tx_in_mempool", txInMempool,
				"sender", sender,
				"sequence", sequence,
				"decode_ms", decodeDur.Milliseconds(),
				"contains_ms", containsDur.Milliseconds(),
				"base_check_ms", baseCheckDur.Milliseconds(),
				"remove_ms", removeDur.Milliseconds(),
				"total_ms", totalDur.Milliseconds(),
				"resp_code", respCode,
				"err", err,
			)
		}
	}()

	decodeStart := time.Now()
	tx, err := h.txDecoder(req.Tx)
	decodeDur = time.Since(decodeStart)
	if err != nil {
		return sdkerrors.ResponseCheckTxWithEvents(
			fmt.Errorf("failed to decode tx: %w", err),
			0,
			0,
			nil,
			false,
		), nil
	}

	if key, keyErr := txKeyFromTx(tx); keyErr == nil {
		sender, sequence = key.sender, key.nonce
	}

	containsStart := time.Now()
	txInMempool = h.mempool.Contains(tx)
	containsDur = time.Since(containsStart)

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
	baseCheckStart := time.Now()
	resp, err = h.checkTx(req)
	baseCheckDur = time.Since(baseCheckStart)

	// if re-check fails for a transaction, we'll need to explicitly purge the tx from
	// the app-side mempool
	if isInvalidCheckTxExecution(resp, err) && isRecheck && txInMempool {
		// remove the tx
		removeStart := time.Now()
		if err := h.mempool.RemoveWithReason(tx, RemovalReasonAnteRejectedInPrepare); err != nil {
			h.logger.Debug(
				"failed to remove tx from app-side mempool when purging for re-check failure",
				"removal-err", err,
			)
		}
		removeDur = time.Since(removeStart)
	}

	return
}

func isInvalidCheckTxExecution(resp *cometabci.ResponseCheckTx, checkTxErr error) bool {
	return resp == nil || resp.Code != 0 || checkTxErr != nil
}
