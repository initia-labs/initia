package types

import (
	"bytes"
	context "context"
	"encoding/binary"

	"github.com/cometbft/cometbft/crypto/tmhash"

	sdk "github.com/cosmos/cosmos-sdk/types"

	vmtypes "github.com/initia-labs/movevm/types"
)

func NewEnv(ctx context.Context, nextAccountNumber uint64, executionCounter uint64) vmtypes.Env {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	txBytes := sdkCtx.TxBytes()
	if len(txBytes) == 0 {
		txBytes = bytes.Repeat([]byte{0}, 32)
	}

	var txHash [32]uint8
	copy(txHash[:], tmhash.Sum(txBytes))

	var sessionID [32]uint8
	counterBz := binary.BigEndian.AppendUint64([]byte{}, executionCounter)
	copy(sessionID[:], tmhash.Sum(append(txBytes, counterBz[:]...)))

	return vmtypes.Env{
		ChainId:             sdkCtx.ChainID(),
		BlockHeight:         uint64(sdkCtx.BlockHeader().Height),          //nolint: gosec
		BlockTimestampNanos: uint64(sdkCtx.BlockHeader().Time.UnixNano()), //nolint: gosec
		NextAccountNumber:   nextAccountNumber,
		TxHash:              txHash,
		SessionId:           sessionID,
	}
}

// NOTE: This is hack of store operation.
// We do not want to increase account number, so make cache context
// and drop write() to fetch next account number without update.
func NextAccountNumber(ctx context.Context, accKeeper AccountKeeper) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	tmpCtx, _ := sdkCtx.CacheContext()
	return accKeeper.NextAccountNumber(tmpCtx)
}
