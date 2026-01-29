package abcipp

import (
	"context"

	cometabci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

// Mempool defines the interface a mempool should implement.
type Mempool interface {
	sdkmempool.Mempool

	// Contains returns true if the transaction is in the mempool.
	Contains(tx sdk.Tx) bool

	// Lookup returns the txHash from the mempool if it exists.
	Lookup(sender string, nonce uint64) (string, bool)

	// GetTxDistribution returns a map of tier to the number of transactions
	GetTxDistribution() map[string]uint64

	// GetTxInfo returns information about a transaction in the mempool.
	GetTxInfo(ctx sdk.Context, tx sdk.Tx) (TxInfo, error)

	// NextExpectedSequence returns the next expected sequence for a sender
	NextExpectedSequence(ctx sdk.Context, sender string) (uint64, bool, error)
}

type AccountKeeper interface {
	GetSequence(context.Context, sdk.AccAddress) (uint64, error)
}

// TxInfo contains information about a transaction in the mempool.
type TxInfo struct {
	Sender   string
	Sequence uint64
	Size     int64
	GasLimit uint64
	Tier     string
	TxBytes  []byte
}

// CheckTx is baseapp's CheckTx method that checks the validity of a transaction.
type CheckTx func(req *cometabci.RequestCheckTx) (*cometabci.ResponseCheckTx, error)

// BaseApp is an interface that allows us to call baseapp's CheckTx method
// as well as retrieve the latest committed state.
type BaseApp interface {
	GetContextForCheckTx(txBytes []byte) sdk.Context
	IsSealed() bool
}
