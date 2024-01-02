package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
)

func TestGetSetValidatorSigningInfo(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	signedBlock, err := input.SlashingKeeper.SignedBlocksWindow(ctx)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(signedBlock + 1)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	consAddr := sdk.ConsAddress(valAddr)

	info, err := input.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
	require.NoError(t, err)
	newInfo := types.NewValidatorSigningInfo(
		sdk.ConsAddress(valAddr),
		int64(4),
		int64(3),
		time.Unix(2, 0),
		false,
		int64(10),
	)
	input.SlashingKeeper.SetValidatorSigningInfo(ctx, consAddr, newInfo)
	info, err = input.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
	require.NoError(t, err)
	require.Equal(t, info.StartHeight, int64(4))
	require.Equal(t, info.IndexOffset, int64(3))
	require.Equal(t, info.JailedUntil, time.Unix(2, 0).UTC())
	require.Equal(t, info.MissedBlocksCounter, int64(10))
}

func TestGetSetValidatorMissedBlockBitmap(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	signedBlock, err := input.SlashingKeeper.SignedBlocksWindow(ctx)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(signedBlock + 1)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	consAddr := sdk.ConsAddress(valAddr)

	missed, err := input.SlashingKeeper.GetMissedBlockBitmapValue(ctx, consAddr, 0)
	require.NoError(t, err)
	require.False(t, missed) // treat empty key as not missed

	err = input.SlashingKeeper.SetMissedBlockBitmapValue(ctx, consAddr, 0, true)
	require.NoError(t, err)

	missed, err = input.SlashingKeeper.GetMissedBlockBitmapValue(ctx, consAddr, 0)
	require.NoError(t, err)
	require.True(t, missed) // now should be missed
}

func TestValidatorMissedBlockBitmap_SmallWindow(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	consAddr := sdk.ConsAddress(valAddr)

	for _, window := range []int64{100, 32_000} {
		params, err := input.SlashingKeeper.GetParams(ctx)
		require.NoError(t, err)

		params.SignedBlocksWindow = window
		require.NoError(t, input.SlashingKeeper.SetParams(ctx, params))

		// validator misses all blocks in the window
		var valIdxOffset int64
		for valIdxOffset < params.SignedBlocksWindow {
			idx := valIdxOffset % params.SignedBlocksWindow
			err := input.SlashingKeeper.SetMissedBlockBitmapValue(ctx, consAddr, idx, true)
			require.NoError(t, err)

			missed, err := input.SlashingKeeper.GetMissedBlockBitmapValue(ctx, consAddr, idx)
			require.NoError(t, err)
			require.True(t, missed)

			valIdxOffset++
		}

		// validator should have missed all blocks
		missedBlocks, err := input.SlashingKeeper.GetValidatorMissedBlocks(ctx, consAddr)
		require.NoError(t, err)
		require.Len(t, missedBlocks, int(params.SignedBlocksWindow))

		// sign next block, which rolls the missed block bitmap
		idx := valIdxOffset % params.SignedBlocksWindow
		err = input.SlashingKeeper.SetMissedBlockBitmapValue(ctx, consAddr, idx, false)
		require.NoError(t, err)

		missed, err := input.SlashingKeeper.GetMissedBlockBitmapValue(ctx, consAddr, idx)
		require.NoError(t, err)
		require.False(t, missed)

		// validator should have missed all blocks except the last one
		missedBlocks, err = input.SlashingKeeper.GetValidatorMissedBlocks(ctx, consAddr)
		require.NoError(t, err)
		require.Len(t, missedBlocks, int(params.SignedBlocksWindow)-1)
	}
}

func TestTombstoned(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	signedBlock, err := input.SlashingKeeper.SignedBlocksWindow(ctx)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(signedBlock + 1)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	consAddr := sdk.ConsAddress(valAddr)

	require.Panics(t, func() { input.SlashingKeeper.Tombstone(ctx, consAddr) })
	require.False(t, input.SlashingKeeper.IsTombstoned(ctx, consAddr))

	newInfo := types.NewValidatorSigningInfo(
		consAddr,
		int64(4),
		int64(3),
		time.Unix(2, 0),
		false,
		int64(10),
	)
	input.SlashingKeeper.SetValidatorSigningInfo(ctx, consAddr, newInfo)

	require.False(t, input.SlashingKeeper.IsTombstoned(ctx, consAddr))
	input.SlashingKeeper.Tombstone(ctx, consAddr)
	require.True(t, input.SlashingKeeper.IsTombstoned(ctx, consAddr))
	require.Panics(t, func() { input.SlashingKeeper.Tombstone(ctx, consAddr) })
}

func TestJailUntil(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	signedBlock, err := input.SlashingKeeper.SignedBlocksWindow(ctx)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(signedBlock + 1)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	consAddr := sdk.ConsAddress(valAddr)

	require.Panics(t, func() { input.SlashingKeeper.JailUntil(ctx, consAddr, time.Now()) })

	newInfo := types.NewValidatorSigningInfo(
		consAddr,
		int64(4),
		int64(3),
		time.Unix(2, 0),
		false,
		int64(10),
	)
	input.SlashingKeeper.SetValidatorSigningInfo(ctx, consAddr, newInfo)
	input.SlashingKeeper.JailUntil(ctx, consAddr, time.Unix(253402300799, 0).UTC())

	info, err := input.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
	require.NoError(t, err)
	require.Equal(t, time.Unix(253402300799, 0).UTC(), info.JailedUntil)
}
