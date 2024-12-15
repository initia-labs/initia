package lanes

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	signer_extraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	blockbase "github.com/skip-mev/block-sdk/v2/block/base"
	"github.com/skip-mev/block-sdk/v2/block/proposals"
)

type (
	txKey struct {
		nonce  uint64
		sender string
	}

	// Mempool manages transaction priority, provides helper functions, and wraps the SDK's Priority Nonce mempool.
	Mempool[C comparable] struct {
		index     sdkmempool.Mempool        // Priority nonce-based mempool.
		extractor signer_extraction.Adapter // Adapter to extract signer information.
		txCache   map[txKey]struct{}        // Cache for quick lookup of transactions in the mempool.
		ratio     math.LegacyDec            // Block space ratio allowed for this lane.
		txEncoder sdk.TxEncoder             // Transaction encoder.
	}
)

// NewMempool creates a new instance of Mempool.
func NewMempool[C comparable](
	txPriority blockbase.TxPriority[C], extractor signer_extraction.Adapter,
	maxTx int, ratio math.LegacyDec, txEncoder sdk.TxEncoder,
) (*Mempool[C], error) {
	// Validate inputs.
	if err := validateMempoolConfig(ratio, txEncoder); err != nil {
		return nil, err
	}

	return &Mempool[C]{
		index: blockbase.NewPriorityMempool(
			blockbase.PriorityNonceMempoolConfig[C]{
				TxPriority: txPriority,
				MaxTx:      maxTx,
			},
			extractor,
		),
		extractor: extractor,
		txCache:   make(map[txKey]struct{}),
		ratio:     ratio,
		txEncoder: txEncoder,
	}, nil
}

// validateMempoolConfig validates the configuration parameters for creating a Mempool.
func validateMempoolConfig(ratio math.LegacyDec, txEncoder sdk.TxEncoder) error {
	if !ratio.IsPositive() {
		return errors.New("mempool creation: ratio must be positive")
	}
	if ratio.GT(math.LegacyOneDec()) {
		return errors.New("mempool creation: ratio must be less than or equal to 1")
	}
	if txEncoder == nil {
		return errors.New("mempool creation: tx encoder is nil")
	}
	return nil
}

// Priority returns the transaction priority.
func (cm *Mempool[C]) Priority(ctx sdk.Context, tx sdk.Tx) any {
	return 1 // Fixed priority for now; extend as needed.
}

// CountTx returns the total number of transactions in the mempool.
func (cm *Mempool[C]) CountTx() int {
	return cm.index.CountTx()
}

// Select provides an iterator over all transactions in the mempool.
func (cm *Mempool[C]) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return cm.index.Select(ctx, txs)
}

// Compare ignores priority check and returns a constant value for ProcessLaneHandler.
func (cm *Mempool[C]) Compare(ctx sdk.Context, this sdk.Tx, other sdk.Tx) (int, error) {
	return 0, nil
}

// Contains checks whether a transaction exists in the mempool.
func (cm *Mempool[C]) Contains(tx sdk.Tx) bool {
	key, err := cm.getTxKey(tx)
	if err != nil {
		return false
	}
	_, exists := cm.txCache[key]
	return exists
}

// Insert adds a transaction to the mempool after validating lane limits.
func (cm *Mempool[C]) Insert(ctx context.Context, tx sdk.Tx) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate lane limits.
	if err := cm.AssertLaneLimits(sdkCtx, tx); err != nil {
		return err
	}

	// Insert into the underlying priority mempool.
	if err := cm.index.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	// Cache the transaction.
	key, err := cm.getTxKey(tx)
	if err != nil {
		return err
	}
	cm.txCache[key] = struct{}{}

	return nil
}

// Remove deletes a transaction from the mempool and its cache.
func (cm *Mempool[C]) Remove(tx sdk.Tx) error {
	// Remove from the priority mempool.
	if err := cm.index.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		return fmt.Errorf("failed to remove transaction from the mempool: %w", err)
	}

	// Remove from the cache.
	key, err := cm.getTxKey(tx)
	if err != nil {
		return err
	}
	delete(cm.txCache, key)

	return nil
}

// getTxKey generates a unique key for a transaction based on its nonce and sender.
func (cm *Mempool[C]) getTxKey(tx sdk.Tx) (txKey, error) {
	signers, err := cm.extractor.GetSigners(tx)
	if err != nil || len(signers) == 0 {
		return txKey{}, fmt.Errorf("failed to extract signer from transaction: %w", err)
	}

	// Use the first signer for indexing.
	signer := signers[0]
	return txKey{nonce: signer.Sequence, sender: signer.Signer.String()}, nil
}

// AssertLaneLimits ensures the transaction does not exceed the lane's size or gas limits.
func (cm *Mempool[C]) AssertLaneLimits(ctx sdk.Context, tx sdk.Tx) error {
	maxBlockSize, maxGasLimit := proposals.GetBlockLimits(ctx)
	maxLaneTxSize := cm.ratio.MulInt64(maxBlockSize).TruncateInt64()
	maxLaneGasLimit := cm.ratio.MulInt(math.NewIntFromUint64(maxGasLimit)).TruncateInt().Uint64()

	txBytes, err := cm.txEncoder(tx)
	if err != nil {
		return fmt.Errorf("failed to encode transaction: %w", err)
	}

	gasTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return errors.New("transaction does not implement FeeTx interface")
	}

	// Validate size and gas limits.
	txSize := int64(len(txBytes))
	txGasLimit := gasTx.GetGas()

	if txSize > maxLaneTxSize {
		return fmt.Errorf("transaction size %d exceeds max lane size %d", txSize, maxLaneTxSize)
	}
	if txGasLimit > maxLaneGasLimit {
		return fmt.Errorf("transaction gas limit %d exceeds max lane gas limit %d", txGasLimit, maxLaneGasLimit)
	}

	return nil
}
