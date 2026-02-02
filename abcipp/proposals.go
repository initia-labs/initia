package abcipp

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	// ProposalHandler is a wrapper around the ABCI++ PrepareProposal and ProcessProposal
	// handlers.
	ProposalHandler struct {
		logger      log.Logger
		txDecoder   sdk.TxDecoder
		txEncoder   sdk.TxEncoder
		mempool     Mempool
		anteHandler sdk.AnteHandler
	}
)

// NewProposalHandler returns a new ABCI++ proposal handler with the ability to use custom process proposal logic.
func NewProposalHandler(
	logger log.Logger,
	txDecoder sdk.TxDecoder,
	txEncoder sdk.TxEncoder,
	mempool Mempool,
	anteHandler sdk.AnteHandler,
) *ProposalHandler {
	return &ProposalHandler{
		logger:      logger,
		txDecoder:   txDecoder,
		txEncoder:   txEncoder,
		mempool:     mempool,
		anteHandler: anteHandler,
	}
}

// PrepareProposalHandler only runs on the block proposer. It selects transactions from the mempool,
// enforces gas/byte limits, removes submissions that exceed the block limits, and drops entries from
// the mempool whenever the ante handler rejects them (including txs that individually exceed the
// configured max gas or size).
func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (resp *abci.ResponsePrepareProposal, err error) {
		if req.Height <= 1 {
			return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
		}

		// In the case where there is a panic, we recover here and return an empty proposal.
		defer func() {
			if rec := recover(); rec != nil {
				h.logger.Error("failed to prepare proposal", "err", rec)

				// TODO: Should we attempt to return a empty proposal here with empty proposal info?
				resp = &abci.ResponsePrepareProposal{Txs: make([][]byte, 0)}
				err = fmt.Errorf("failed to prepare proposal: %v", rec)
			}
		}()

		h.logger.Info(
			"mempool distribution before proposal creation",
			"distribution", h.mempool.GetTxDistribution(),
			"height", req.Height,
		)

		// Get the max gas limit and max block size for the proposal.
		maxGasLimit := uint64(ctx.ConsensusParams().Block.MaxGas) //nolint: gosec
		maxBlockSize := ctx.ConsensusParams().Block.MaxBytes

		// Fill the proposal with transactions from each lane.
		var (
			totalSize    int64
			totalGas     uint64
			txsToInclude [][]byte
			txsToRemove  []sdk.Tx
		)

		for iterator := h.mempool.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
			tx := iterator.Tx()

			txInfo, err := h.mempool.GetTxInfo(ctx, tx)
			if err != nil {
				h.logger.Info("failed to get hash of tx", "err", err)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			// If the transaction is too large, we skip it.
			if updatedSize := totalSize + txInfo.Size; updatedSize > maxBlockSize {
				h.logger.Debug(
					"failed to select tx for block limit; tx bytes above the maximum allowed",
					"tx_size", txInfo.Size,
					"total_size", totalSize,
					"max_block_size", maxBlockSize,
					"tier", txInfo.Tier,
					"sender", txInfo.Sender,
					"sequence", txInfo.Sequence,
					"tx_hash", TxHash(txInfo.TxBytes),
				)

				if txInfo.Size > maxBlockSize {
					txsToRemove = append(txsToRemove, tx)
				}

				continue
			}

			// If the gas limit of the transaction is too large, we skip it.
			if updatedGas := totalGas + txInfo.GasLimit; updatedGas > maxGasLimit {
				h.logger.Debug(
					"failed to select tx for block limit; gas limit above the maximum allowed",
					"tx_gas", txInfo.GasLimit,
					"total_gas", totalGas,
					"max_gas", maxGasLimit,
					"tier", txInfo.Tier,
					"sender", txInfo.Sender,
					"sequence", txInfo.Sequence,
					"tx_hash", TxHash(txInfo.TxBytes),
				)

				if txInfo.GasLimit > maxGasLimit {
					txsToRemove = append(txsToRemove, tx)
				}

				continue
			}

			// Verify the transaction.
			catchCtx, write := ctx.CacheContext()
			if _, err := h.anteHandler(catchCtx, tx, false); err != nil {
				h.logger.Info(
					"failed to verify tx",
					"err", err,
					"tier", txInfo.Tier,
					"sender", txInfo.Sender,
					"sequence", txInfo.Sequence,
					"tx_hash", TxHash(txInfo.TxBytes),
				)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			write()

			totalSize += txInfo.Size
			totalGas += txInfo.GasLimit
			txsToInclude = append(txsToInclude, txInfo.TxBytes)
		}

		// remove the invalid transactions from the mempool.
		for _, tx := range txsToRemove {
			err := h.mempool.Remove(tx)
			if err != nil {
				h.logger.Error("failed to remove tx from mempool", "err", err)
			}
		}

		h.logger.Info(
			"prepared proposal",
			"num_txs", len(txsToInclude),
			"total_tx_bytes", totalSize,
			"max_tx_bytes", maxBlockSize,
			"total_gas_limit", totalGas,
			"max_gas_limit", maxGasLimit,
			"height", req.Height,
		)

		h.logger.Info(
			"mempool distribution after proposal creation",
			"distribution", h.mempool.GetTxDistribution(),
			"height", req.Height,
		)

		return &abci.ResponsePrepareProposal{
			Txs: txsToInclude,
		}, nil
	}
}

// ProcessProposalHandler runs on the validators that did not propose the block. It mirrors the same
// limits and ante logic used during PrepareProposal, rejecting proposals if any transaction exceeds the
// configured limits or fails validation so all honest validators arrive at the same view.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (resp *abci.ResponseProcessProposal, err error) {
		if req.Height <= 1 {
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
		}

		// In the case where any of the lanes panic, we recover here and return a reject status.
		defer func() {
			if rec := recover(); rec != nil {
				h.logger.Error("failed to process proposal", "recover_err", rec)

				resp = &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
				err = fmt.Errorf("failed to process proposal: %v", rec)
			}
		}()

		// Decode the transactions in the proposal. These will be verified by each lane in a greedy fashion.
		decodedTxs, err := GetDecodedTxs(h.txDecoder, req.Txs)
		if err != nil {
			h.logger.Error("failed to decode txs", "err", err)
			return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, err
		}

		// Get the max gas limit and max block size for the proposal.
		maxGasLimit := uint64(ctx.ConsensusParams().Block.MaxGas) //nolint: gosec
		maxBlockSize := ctx.ConsensusParams().Block.MaxBytes

		// Verify the transaction.
		var totalTxBytes int64
		var totalGas uint64

		for i, tx := range decodedTxs {
			txBytes := req.Txs[i]
			if feeTx, ok := tx.(sdk.FeeTx); ok {
				gas := feeTx.GetGas()
				if totalGas+gas > maxGasLimit {
					h.logger.Error(
						"failed to process proposal; gas limit above the maximum allowed",
						"tx_gas", gas,
						"total_gas", totalGas,
						"max_gas", maxGasLimit,
						"tx_hash", TxHash(txBytes),
					)
					return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, fmt.Errorf("tx gas limit exceeds max gas limit")
				}

				totalGas += feeTx.GetGas()
			} else {
				h.logger.Error(
					"failed to get gas from tx",
					"err", "tx does not implement FeeTx",
					"tx_hash", TxHash(txBytes),
				)
				return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, fmt.Errorf("tx does not implement FeeTx")
			}

			txBz := req.Txs[i]
			size := int64(len(txBz))
			if totalTxBytes+size > maxBlockSize {
				h.logger.Error(
					"failed to process proposal; tx bytes above the maximum allowed",
					"tx_size", size,
					"total_size", totalTxBytes,
					"max_block_size", maxBlockSize,
					"tx_hash", TxHash(txBytes),
				)
				return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, fmt.Errorf("tx size exceeds max block size")
			}

			totalTxBytes += size

			// Verify the transaction.
			catchCtx, write := ctx.CacheContext()
			if _, err := h.anteHandler(catchCtx, tx, false); err != nil {
				h.logger.Error(
					"failed to validate the proposal",
					"err", err,
					"tx_hash", TxHash(txBytes),
				)
				return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}, err
			}

			write()
		}

		h.logger.Info(
			"processed proposal",
			"num_txs", len(req.Txs),
			"total_tx_bytes", totalTxBytes,
			"max_tx_bytes", maxBlockSize,
			"total_gas_limit", totalGas,
			"max_gas_limit", maxGasLimit,
			"height", req.Height,
		)

		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
	}
}
