package types

import (
	"bytes"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/tmhash"
	sdk "github.com/cosmos/cosmos-sdk/types"

	vmtypes "github.com/initia-labs/initiavm/types"
)

func NewEnv(ctx sdk.Context, nextAccountNumber uint64, executionCounter math.Int) vmtypes.Env {
	txBytes := ctx.TxBytes()
	if len(txBytes) == 0 {
		txBytes = bytes.Repeat([]byte{0}, 32)
	}

	counterBz, err := executionCounter.Marshal()
	if err != nil {
		panic(err)
	}

	var txHash [32]uint8
	copy(txHash[:], tmhash.Sum(ctx.TxBytes()))

	var sessionID [32]uint8
	copy(sessionID[:], tmhash.Sum(append(txBytes, counterBz...)))

	return vmtypes.Env{
		BlockHeight:       uint64(ctx.BlockHeader().Height),
		BlockTimestamp:    uint64(ctx.BlockHeader().Time.Unix()),
		NextAccountNumber: nextAccountNumber,
		TxHash:            txHash,
		SessionId:         sessionID,
	}
}

// NOTE: This is hack of store operation.
// We do not want to increase account number, so make cache context
// and drop write() to fetch next account number without update.
func NextAccountNumber(ctx sdk.Context, accKeeper AccountKeeper) uint64 {
	tmpCtx, _ := ctx.CacheContext()
	return accKeeper.NextAccountNumber(tmpCtx)
}
