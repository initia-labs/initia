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
	ctx = ctx.WithBlockHeight(input.SlashingKeeper.SignedBlocksWindow(ctx) + 1)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	consAddr := sdk.ConsAddress(valAddr)

	info, found := input.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
	require.False(t, found)
	newInfo := types.NewValidatorSigningInfo(
		sdk.ConsAddress(valAddr),
		int64(4),
		int64(3),
		time.Unix(2, 0),
		false,
		int64(10),
	)
	input.SlashingKeeper.SetValidatorSigningInfo(ctx, consAddr, newInfo)
	info, found = input.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
	require.True(t, found)
	require.Equal(t, info.StartHeight, int64(4))
	require.Equal(t, info.IndexOffset, int64(3))
	require.Equal(t, info.JailedUntil, time.Unix(2, 0).UTC())
	require.Equal(t, info.MissedBlocksCounter, int64(10))
}

func TestGetSetValidatorMissedBlockBitArray(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	ctx = ctx.WithBlockHeight(input.SlashingKeeper.SignedBlocksWindow(ctx) + 1)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	consAddr := sdk.ConsAddress(valAddr)

	missed := input.SlashingKeeper.GetValidatorMissedBlockBitArray(ctx, consAddr, 0)
	require.False(t, missed) // treat empty key as not missed
	input.SlashingKeeper.SetValidatorMissedBlockBitArray(ctx, consAddr, 0, true)
	missed = input.SlashingKeeper.GetValidatorMissedBlockBitArray(ctx, consAddr, 0)
	require.True(t, missed) // now should be missed
}

func TestTombstoned(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	ctx = ctx.WithBlockHeight(input.SlashingKeeper.SignedBlocksWindow(ctx) + 1)

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
	ctx = ctx.WithBlockHeight(input.SlashingKeeper.SignedBlocksWindow(ctx) + 1)

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

	info, ok := input.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
	require.True(t, ok)
	require.Equal(t, time.Unix(253402300799, 0).UTC(), info.JailedUntil)
}
