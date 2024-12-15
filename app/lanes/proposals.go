package lanes

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	blockbase "github.com/skip-mev/block-sdk/v2/block/base"
	"github.com/skip-mev/block-sdk/v2/block/proposals"
)

// DefaultProposalHandler provides default implementations for the PrepareLaneHandler and ProcessLaneHandler.
type DefaultProposalHandler struct {
	lane *blockbase.BaseLane
}

// NewDefaultProposalHandler creates a new instance of DefaultProposalHandler.
func NewDefaultProposalHandler(lane *blockbase.BaseLane) *DefaultProposalHandler {
	return &DefaultProposalHandler{lane: lane}
}

// PrepareLaneHandler returns a default implementation of the PrepareLaneHandler.
// It selects transactions that meet blockspace/gas constraints and excludes invalid ones.
func (h *DefaultProposalHandler) PrepareLaneHandler() blockbase.PrepareLaneHandler {
	return func(ctx sdk.Context, proposal proposals.Proposal, limit proposals.LaneLimits) ([]sdk.Tx, []sdk.Tx, error) {
		var (
			totalSize    int64
			totalGas     uint64
			txsToInclude []sdk.Tx
			txsToRemove  []sdk.Tx
		)

		for iterator := h.lane.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
			tx := iterator.Tx()

			txInfo, err := h.lane.GetTxInfo(ctx, tx)
			if err != nil {
				h.logAndAppend("failed to get tx info", tx, err, &txsToRemove)
				continue
			}

			// Validate transaction eligibility for this lane.
			if !h.validateTxEligibility(ctx, proposal, tx, txInfo, limit, totalSize, totalGas, &txsToRemove) {
				continue
			}

			// Verify the transaction.
			if err = h.lane.VerifyTx(ctx, tx, false); err != nil {
				h.logAndAppend("failed to verify tx", tx, err, &txsToRemove)
				continue
			}

			// Update totals and include the transaction.
			totalSize += txInfo.Size
			totalGas += txInfo.GasLimit
			txsToInclude = append(txsToInclude, tx)
		}

		return txsToInclude, txsToRemove, nil
	}
}

// ProcessLaneHandler returns a default implementation of the ProcessLaneHandler.
// Ensures transactions meet lane-specific constraints and are correctly prioritized.
func (h *DefaultProposalHandler) ProcessLaneHandler() blockbase.ProcessLaneHandler {
	return func(ctx sdk.Context, partialProposal []sdk.Tx) ([]sdk.Tx, []sdk.Tx, error) {
		if len(partialProposal) == 0 {
			return nil, nil, nil
		}

		for index, tx := range partialProposal {
			// Check if the transaction belongs to this lane.
			if !h.lane.Match(ctx, tx) {
				return h.handleNonMatchingTransactions(ctx, partialProposal, index)
			}

			// Validate transaction priority.
			if err := h.validateTxPriority(ctx, partialProposal, index); err != nil {
				return nil, nil, err
			}

			// Verify the transaction.
			if err := h.lane.VerifyTx(ctx, tx, false); err != nil {
				return nil, nil, fmt.Errorf("failed to verify tx: %w", err)
			}
		}

		// All transactions are valid and belong to this lane.
		return partialProposal, nil, nil
	}
}

// validateTxEligibility checks if a transaction is eligible for inclusion in the lane.
func (h *DefaultProposalHandler) validateTxEligibility(
	ctx sdk.Context,
	proposal proposals.Proposal,
	tx sdk.Tx,
	txInfo *blockbase.TxInfo,
	limit proposals.LaneLimits,
	totalSize int64,
	totalGas uint64,
	txsToRemove *[]sdk.Tx,
) bool {
	// Check if transaction belongs to this lane.
	if !h.lane.Match(ctx, tx) {
		h.logAndAppend("tx does not belong to lane", tx, nil, txsToRemove)
		return false
	}

	// Check if the transaction is already in the proposal.
	if proposal.Contains(txInfo.Hash) {
		h.lane.Logger().Info(
			"tx already in proposal",
			"tx_hash", txInfo.Hash,
			"lane", h.lane.Name(),
		)
		return false
	}

	// Check size limits.
	if updatedSize := totalSize + txInfo.Size; updatedSize > limit.MaxTxBytes {
		h.logTxLimitExceeded("tx size exceeds max limit", txInfo, totalSize, limit.MaxTxBytes, txsToRemove)
		return false
	}

	// Check gas limits.
	if updatedGas := totalGas + txInfo.GasLimit; updatedGas > limit.MaxGasLimit {
		h.logTxLimitExceeded("tx gas exceeds max limit", txInfo, totalGas, limit.MaxGasLimit, txsToRemove)
		return false
	}

	return true
}

// validateTxPriority checks if transactions are correctly ordered based on lane priority.
func (h *DefaultProposalHandler) validateTxPriority(ctx sdk.Context, partialProposal []sdk.Tx, index int) error {
	if index > 0 {
		prevTx := partialProposal[index-1]
		currentTx := partialProposal[index]

		priority, err := h.lane.Compare(ctx, prevTx, currentTx)
		if priority == -1 || err != nil {
			return fmt.Errorf("tx at index %d has higher priority than index %d", index, index-1)
		}
	}
	return nil
}

// handleNonMatchingTransactions processes transactions that do not belong to the lane.
func (h *DefaultProposalHandler) handleNonMatchingTransactions(
	ctx sdk.Context,
	partialProposal []sdk.Tx,
	index int,
) ([]sdk.Tx, []sdk.Tx, error) {
	if index+1 < len(partialProposal) {
		if err := h.lane.VerifyNoMatches(ctx, partialProposal[index+1:]); err != nil {
			return nil, nil, fmt.Errorf("failed to verify no matches: %w", err)
		}
	}
	return partialProposal[:index], partialProposal[index:], nil
}

// logAndAppend logs an error and appends the transaction to the remove list.
func (h *DefaultProposalHandler) logAndAppend(message string, tx sdk.Tx, err error, txsToRemove *[]sdk.Tx) {
	h.lane.Logger().Info(message, "err", err)
	*txsToRemove = append(*txsToRemove, tx)
}

// logTxLimitExceeded logs and handles transactions exceeding size or gas limits.
func (h *DefaultProposalHandler) logTxLimitExceeded(
	message string,
	txInfo *blockbase.TxInfo,
	totalValue int64,
	maxValue int64,
	txsToRemove *[]sdk.Tx,
) {
	h.lane.Logger().Debug(
		message,
		"tx_hash", txInfo.Hash,
		"lane", h.lane.Name(),
		"value", txInfo.Size,
		"total_value", totalValue,
		"max_value", maxValue,
	)
	if txInfo.Size > maxValue {
		*txsToRemove = append(*txsToRemove, txInfo.Tx)
	}
}
