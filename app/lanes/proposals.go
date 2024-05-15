package lanes

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	blockbase "github.com/skip-mev/block-sdk/v2/block/base"
	"github.com/skip-mev/block-sdk/v2/block/proposals"
)

// DefaultProposalHandler returns a default implementation of the PrepareLaneHandler and
// ProcessLaneHandler.
type DefaultProposalHandler struct {
	lane *blockbase.BaseLane
}

// NewDefaultProposalHandler returns a new default proposal handler.
func NewDefaultProposalHandler(lane *blockbase.BaseLane) *DefaultProposalHandler {
	return &DefaultProposalHandler{
		lane: lane,
	}
}

// DefaultPrepareLaneHandler returns a default implementation of the PrepareLaneHandler. It
// selects all transactions in the mempool that are valid and not already in the partial
// proposal. It will continue to reap transactions until the maximum blockspace/gas for this
// lane has been reached. Additionally, any transactions that are invalid will be returned.
func (h *DefaultProposalHandler) PrepareLaneHandler() blockbase.PrepareLaneHandler {
	return func(ctx sdk.Context, proposal proposals.Proposal, limit proposals.LaneLimits) ([]sdk.Tx, []sdk.Tx, error) {
		var (
			totalSize    int64
			totalGas     uint64
			txsToInclude []sdk.Tx
			txsToRemove  []sdk.Tx
		)

		// Select all transactions in the mempool that are valid and not already in the
		// partial proposal.
		for iterator := h.lane.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
			tx := iterator.Tx()

			txInfo, err := h.lane.GetTxInfo(ctx, tx)
			if err != nil {
				h.lane.Logger().Info("failed to get hash of tx", "err", err)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			// Double check that the transaction belongs to this lane.
			if !h.lane.Match(ctx, tx) {
				h.lane.Logger().Info(
					"failed to select tx for lane; tx does not belong to lane",
					"tx_hash", txInfo.Hash,
					"lane", h.lane.Name(),
				)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			// if the transaction is already in the (partial) block proposal, we skip it.
			if proposal.Contains(txInfo.Hash) {
				h.lane.Logger().Info(
					"failed to select tx for lane; tx is already in proposal",
					"tx_hash", txInfo.Hash,
					"lane", h.lane.Name(),
				)

				continue
			}

			// If the transaction is too large, we break and do not attempt to include more txs.
			if updatedSize := totalSize + txInfo.Size; updatedSize > limit.MaxTxBytes {
				h.lane.Logger().Info(
					"failed to select tx for lane; tx bytes above the maximum allowed",
					"lane", h.lane.Name(),
					"tx_size", txInfo.Size,
					"total_size", totalSize,
					"max_tx_bytes", limit.MaxTxBytes,
					"tx_hash", txInfo.Hash,
				)

				if txInfo.Size > limit.MaxTxBytes {
					txsToRemove = append(txsToRemove, tx)
				}

				continue
			}

			// If the gas limit of the transaction is too large, we break and do not attempt to include more txs.
			if updatedGas := totalGas + txInfo.GasLimit; updatedGas > limit.MaxGasLimit {
				h.lane.Logger().Info(
					"failed to select tx for lane; gas limit above the maximum allowed",
					"lane", h.lane.Name(),
					"tx_gas", txInfo.GasLimit,
					"total_gas", totalGas,
					"max_gas", limit.MaxGasLimit,
					"tx_hash", txInfo.Hash,
				)

				if txInfo.GasLimit > limit.MaxGasLimit {
					txsToRemove = append(txsToRemove, tx)
				}

				continue
			}

			// Verify the transaction.
			if err = h.lane.VerifyTx(ctx, tx, false); err != nil {
				h.lane.Logger().Info(
					"failed to verify tx",
					"tx_hash", txInfo.Hash,
					"err", err,
				)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			totalSize += txInfo.Size
			totalGas += txInfo.GasLimit
			txsToInclude = append(txsToInclude, tx)
		}

		return txsToInclude, txsToRemove, nil
	}
}

// DefaultProcessLaneHandler returns a default implementation of the ProcessLaneHandler. It verifies
// the following invariants:
//  1. Transactions belonging to the lane must be contiguous from the beginning of the partial proposal.
//  2. Transactions that do not belong to the lane must be contiguous from the end of the partial proposal.
//  3. Transactions must be ordered respecting the priority defined by the lane (e.g. gas price).
//  4. Transactions must be valid according to the verification logic of the lane.
func (h *DefaultProposalHandler) ProcessLaneHandler() blockbase.ProcessLaneHandler {
	return func(ctx sdk.Context, partialProposal []sdk.Tx) ([]sdk.Tx, []sdk.Tx, error) {
		if len(partialProposal) == 0 {
			return nil, nil, nil
		}

		for index, tx := range partialProposal {
			if !h.lane.Match(ctx, tx) {
				// If the transaction does not belong to this lane, we return the remaining transactions
				// iff there are no matches in the remaining transactions after this index.
				if index+1 < len(partialProposal) {
					if err := h.lane.VerifyNoMatches(ctx, partialProposal[index+1:]); err != nil {
						return nil, nil, fmt.Errorf("failed to verify no matches: %w", err)
					}
				}

				return partialProposal[:index], partialProposal[index:], nil
			}

			// If the transactions do not respect the priority defined by the mempool, we consider the proposal
			// to be invalid
			if index > 0 {
				if v, err := h.lane.Compare(ctx, partialProposal[index-1], tx); v == -1 || err != nil {
					return nil, nil, fmt.Errorf("transaction at index %d has a higher priority than %d", index, index-1)
				}
			}

			if err := h.lane.VerifyTx(ctx, tx, false); err != nil {
				return nil, nil, fmt.Errorf("failed to verify tx: %w", err)
			}
		}

		// This means we have processed all transactions in the partial proposal i.e.
		// all of the transactions belong to this lane. There are no remaining transactions.
		return partialProposal, nil, nil
	}
}
