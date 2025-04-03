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

	// Mempool defines a mempool that orders transactions based on the
	// txPriority. The mempool is a wrapper on top of the SDK's Priority Nonce mempool.
	// It include's additional helper functions that allow users to determine if a
	// transaction is already in the mempool and to compare the priority of two
	// transactions.
	Mempool[C comparable] struct {
		// index defines an index of transactions.
		index sdkmempool.Mempool

		// signerExtractor defines the signer extraction adapter that allows us to
		// extract the signer from a transaction.
		extractor signer_extraction.Adapter

		// txCache is a map of all transactions in the mempool. It is used
		// to quickly check if a transaction is already in the mempool.
		txCache map[txKey]struct{}

		// ratio defines the relative percentage of block space that can be
		// used by this lane.
		ratio math.LegacyDec

		// txEncoder defines tx encoder.
		txEncoder sdk.TxEncoder
	}
)

// NewMempool returns a new Mempool.
func NewMempool[C comparable](
	txPriority blockbase.TxPriority[C], extractor signer_extraction.Adapter,
	maxTx int, ratio math.LegacyDec, txEncoder sdk.TxEncoder,
) (*Mempool[C], error) {
	if !ratio.IsPositive() {
		return nil, errors.New("mempool creation; ratio must be positive")
	} else if ratio.GT(math.LegacyOneDec()) {
		return nil, errors.New("mempool creation; ratio must be less than or equal to 1")
	}
	if txEncoder == nil {
		return nil, errors.New("mempool creation; tx encoder is nil")
	}

	return &Mempool[C]{
		index: NewPriorityMempool(
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

// Priority returns the priority of the transaction.
func (cm *Mempool[C]) Priority(ctx sdk.Context, tx sdk.Tx) any {
	return 1
}

// CountTx returns the number of transactions in the mempool.
func (cm *Mempool[C]) CountTx() int {
	return cm.index.CountTx()
}

// Select returns an iterator of all transactions in the mempool. NOTE: If you
// remove a transaction from the mempool while iterating over the transactions,
// the iterator will not be aware of the removal and will continue to iterate
// over the removed transaction. Be sure to reset the iterator if you remove a transaction.
func (cm *Mempool[C]) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return cm.index.Select(ctx, txs)
}

// Compare return 0 to ignore priority check in ProcessLaneHandler.
func (cm *Mempool[C]) Compare(ctx sdk.Context, this sdk.Tx, other sdk.Tx) (int, error) {
	return 0, nil
}

// Contains returns true if the transaction is contained in the mempool.
func (cm *Mempool[C]) Contains(tx sdk.Tx) bool {
	if key, err := cm.getTxKey(tx); err != nil {
		return false
	} else {
		if _, ok := cm.txCache[key]; ok {
			return true
		} else {
			return false
		}
	}
}

// Insert inserts a transaction into the mempool.
func (cm *Mempool[C]) Insert(ctx context.Context, tx sdk.Tx) error {
	if err := cm.AssertLaneLimits(sdk.UnwrapSDKContext(ctx), tx); err != nil {
		return err
	}

	if err := cm.index.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	if key, err := cm.getTxKey(tx); err != nil {
		return err
	} else {
		cm.txCache[key] = struct{}{}
	}

	return nil
}

// Remove removes a transaction from the mempool.
func (cm *Mempool[C]) Remove(tx sdk.Tx) error {
	if err := cm.index.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		return fmt.Errorf("failed to remove transaction from the mempool: %w", err)
	}

	if key, err := cm.getTxKey(tx); err != nil {
		return err
	} else {
		delete(cm.txCache, key)
	}

	return nil
}

func (cm *Mempool[C]) getTxKey(tx sdk.Tx) (txKey, error) {
	signers, err := cm.extractor.GetSigners(tx)
	if err != nil {
		return txKey{}, err
	}
	if len(signers) == 0 {
		return txKey{}, fmt.Errorf("attempted to remove a tx with no signatures")
	}
	sig := signers[0]
	sender := sig.Signer.String()
	nonce := sig.Sequence
	return txKey{nonce, sender}, nil
}

// AssertLaneLimits asserts that the transaction does not exceed the lane's max size and gas limit.
func (cm *Mempool[C]) AssertLaneLimits(ctx sdk.Context, tx sdk.Tx) error {
	maxBlockSize, maxGasLimit := proposals.GetBlockLimits(ctx)
	maxLaneTxSize := cm.ratio.MulInt64(maxBlockSize).TruncateInt().Int64()
	maxLaneGasLimit := cm.ratio.MulInt(math.NewIntFromUint64(maxGasLimit)).TruncateInt().Uint64()

	txBytes, err := cm.txEncoder(tx)
	if err != nil {
		return fmt.Errorf("failed to encode transaction: %w", err)
	}

	gasTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return fmt.Errorf("failed to cast transaction to gas tx")
	}

	txSize := int64(len(txBytes))
	txGasLimit := gasTx.GetGas()

	if txSize > maxLaneTxSize {
		return fmt.Errorf("tx size %d exceeds max lane size %d", txSize, maxLaneTxSize)
	}

	if txGasLimit > maxLaneGasLimit {
		return fmt.Errorf("tx gas limit %d exceeds max lane gas limit %d", txGasLimit, maxLaneGasLimit)
	}

	return nil
}

// SkipListBufferLen returns the number of skip lists in the buffer.
//
// Only for testing.
func (cm *Mempool[C]) SkipListBufferLenForTesting() int {
	return len(cm.index.(*PriorityNonceMempool[C]).skipListBuffer)
}
