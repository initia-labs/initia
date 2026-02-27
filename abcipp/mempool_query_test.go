package abcipp

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/stretchr/testify/require"
)

func TestGetTxInfo(t *testing.T) {
	mp, keeper, sdkCtx, _ := newTestMempoolWithEvents(t, 64)
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default") // active
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default") // queued
	require.NoError(t, mp.Insert(ctx, tx0))
	require.NoError(t, mp.Insert(ctx, tx2))

	t.Run("active", func(t *testing.T) {
		info, err := mp.GetTxInfo(sdkCtx, tx0)
		require.NoError(t, err)
		require.Equal(t, sender.String(), info.Sender)
		require.Equal(t, uint64(0), info.Sequence)
		require.Equal(t, "default", info.Tier)
		require.Equal(t, encodeTx(t, tx0), info.TxBytes)
	})

	t.Run("queued", func(t *testing.T) {
		info, err := mp.GetTxInfo(sdkCtx, tx2)
		require.NoError(t, err)
		require.Equal(t, sender.String(), info.Sender)
		require.Equal(t, uint64(2), info.Sequence)
		require.Equal(t, "queued", info.Tier)
		require.Equal(t, encodeTx(t, tx2), info.TxBytes)
	})

	t.Run("not-found", func(t *testing.T) {
		missing := newTestTxWithPriv(priv, 9, 1000, "default")
		_, err := mp.GetTxInfo(sdkCtx, missing)
		require.ErrorIs(t, err, sdkmempool.ErrTxNotFound)
	})
}

func TestIteratePendingTxsOrdersAndStopsEarly(t *testing.T) {
	mp, keeper, sdkCtx, _ := newTestMempoolWithEvents(t, 64)
	ctx := sdk.WrapSDKContext(sdkCtx)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	senderB := sdk.AccAddress(privB.PubKey().Address())
	keeper.SetSequence(senderA, 0)
	keeper.SetSequence(senderB, 0)

	// A: active 0,1 and queued 3
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privA, 1, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privA, 3, 1000, "default")))
	// B: active 0 and queued 2
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privB, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privB, 2, 1000, "default")))

	type pair struct {
		sender string
		nonce  uint64
	}
	got := make([]pair, 0, 3)
	mp.IteratePendingTxs(func(sender string, nonce uint64, tx sdk.Tx) bool {
		got = append(got, pair{sender: sender, nonce: nonce})
		return true
	})

	// Pending iterator must include only active txs and be sorted by sender, nonce.
	if senderA.String() < senderB.String() {
		require.Equal(t, []pair{
			{sender: senderA.String(), nonce: 0},
			{sender: senderA.String(), nonce: 1},
			{sender: senderB.String(), nonce: 0},
		}, got)
	} else {
		require.Equal(t, []pair{
			{sender: senderB.String(), nonce: 0},
			{sender: senderA.String(), nonce: 0},
			{sender: senderA.String(), nonce: 1},
		}, got)
	}

	// Early stop should stop iteration immediately after first callback.
	calls := 0
	mp.IteratePendingTxs(func(sender string, nonce uint64, tx sdk.Tx) bool {
		calls++
		return false
	})
	require.Equal(t, 1, calls)
}

func TestIterateQueuedTxsOrdersAndStopsEarly(t *testing.T) {
	mp, keeper, sdkCtx, _ := newTestMempoolWithEvents(t, 64)
	ctx := sdk.WrapSDKContext(sdkCtx)

	privA := secp256k1.GenPrivKey()
	privB := secp256k1.GenPrivKey()
	senderA := sdk.AccAddress(privA.PubKey().Address())
	senderB := sdk.AccAddress(privB.PubKey().Address())
	keeper.SetSequence(senderA, 0)
	keeper.SetSequence(senderB, 0)

	// A: active 0 and queued 2,3
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privA, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privA, 2, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privA, 3, 1000, "default")))
	// B: active 0 and queued 1,4
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privB, 0, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privB, 1, 1000, "default")))
	require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(privB, 4, 1000, "default")))

	type pair struct {
		sender string
		nonce  uint64
	}
	got := make([]pair, 0, 3)
	mp.IterateQueuedTxs(func(sender string, nonce uint64, tx sdk.Tx) bool {
		got = append(got, pair{sender: sender, nonce: nonce})
		return true
	})

	// Queued iterator must include only queued txs and be sorted by sender, nonce.
	// B:1 is active (next expected), so queued set is A:2,3 and B:4.
	if senderA.String() < senderB.String() {
		require.Equal(t, []pair{
			{sender: senderA.String(), nonce: 2},
			{sender: senderA.String(), nonce: 3},
			{sender: senderB.String(), nonce: 4},
		}, got)
	} else {
		require.Equal(t, []pair{
			{sender: senderB.String(), nonce: 4},
			{sender: senderA.String(), nonce: 2},
			{sender: senderA.String(), nonce: 3},
		}, got)
	}

	calls := 0
	mp.IterateQueuedTxs(func(sender string, nonce uint64, tx sdk.Tx) bool {
		calls++
		return false
	})
	require.Equal(t, 1, calls)
}
