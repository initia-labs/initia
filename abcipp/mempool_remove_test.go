package abcipp

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/stretchr/testify/require"
)

// TestRemoveUnknownTxReturnsNotFound keeps error behavior explicit so
// call sites can safely differentiate no-op removals from successful removals.
func TestRemoveUnknownTxReturnsNotFound(t *testing.T) {
	mp, keeper, sdkCtx, _ := newTestMempoolWithEvents(t, 10)
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	tx := newTestTxWithPriv(priv, 0, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx))

	missing := newTestTxWithPriv(priv, 99, 1000, "default")
	err := mp.RemoveWithReason(missing, RemovalReasonAnteRejectedInPrepare)
	require.ErrorIs(t, err, sdkmempool.ErrTxNotFound)
}

func TestRemoveEntriesLockedRemovesActiveEntries(t *testing.T) {
	mp, keeper, sdkCtx, _ := newTestMempoolWithEvents(t, 32)
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 0)

	tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
	tx1 := newTestTxWithPriv(priv, 1, 1000, "default")
	require.NoError(t, mp.Insert(ctx, tx0))
	require.NoError(t, mp.Insert(ctx, tx1))

	key0, err := txKeyFromTx(tx0)
	require.NoError(t, err)
	key1, err := txKeyFromTx(tx1)
	require.NoError(t, err)

	mp.mtx.Lock()
	e0 := mp.entries[key0]
	e1 := mp.entries[key1]
	mp.removeEntryLocked(e0)
	mp.removeEntryLocked(nil)
	mp.removeEntryLocked(e1)
	mp.mtx.Unlock()

	require.Equal(t, 0, mp.CountTx())
	require.Equal(t, 0, activeCount(mp))
	require.NoError(t, mp.ValidateInvariants())
}

func TestRemoveActiveStaleLockedBranches(t *testing.T) {
	t.Run("returnsEarlyWhenOnChainSeqIsZero", func(t *testing.T) {
		mp, keeper, sdkCtx, _ := newTestMempoolWithEvents(t, 16)
		ctx := sdk.WrapSDKContext(sdkCtx)

		priv := secp256k1.GenPrivKey()
		sender := sdk.AccAddress(priv.PubKey().Address())
		keeper.SetSequence(sender, 0)
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 0, 1000, "default")))

		mp.mtx.Lock()
		ss := mp.senders[sender.String()]
		removed := mp.removeActiveStaleLocked(ss, 0)
		mp.mtx.Unlock()

		require.Len(t, removed, 0)
		require.Equal(t, 1, mp.CountTx())
	})

	t.Run("returnsEarlyWhenActiveMinIsAtOrAboveOnChainSeq", func(t *testing.T) {
		mp, keeper, sdkCtx, _ := newTestMempoolWithEvents(t, 16)
		ctx := sdk.WrapSDKContext(sdkCtx)

		priv := secp256k1.GenPrivKey()
		sender := sdk.AccAddress(priv.PubKey().Address())
		keeper.SetSequence(sender, 3)
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 3, 1000, "default")))
		require.NoError(t, mp.Insert(ctx, newTestTxWithPriv(priv, 4, 1000, "default")))

		mp.mtx.Lock()
		ss := mp.senders[sender.String()]
		require.Equal(t, uint64(3), ss.activeMin)
		removed := mp.removeActiveStaleLocked(ss, 3)
		mp.mtx.Unlock()

		require.Len(t, removed, 0)
		require.Equal(t, 2, mp.CountTx())
	})

	t.Run("removesRangeWithEndClampAndSkipsHoles", func(t *testing.T) {
		mp := NewPriorityMempool(PriorityMempoolConfig{MaxTx: 16}, testTxEncoder, newMockAccountKeeper())
		sdkCtx := testSDKContext()

		priv := secp256k1.GenPrivKey()
		sender := sdk.AccAddress(priv.PubKey().Address())

		tx0 := newTestTxWithPriv(priv, 0, 1000, "default")
		tx2 := newTestTxWithPriv(priv, 2, 1000, "default")
		bz0 := encodeTx(t, tx0)
		bz2 := encodeTx(t, tx2)

		key0, err := txKeyFromTx(tx0)
		require.NoError(t, err)
		key2, err := txKeyFromTx(tx2)
		require.NoError(t, err)

		mp.mtx.Lock()
		entry0 := &txEntry{
			tx:       tx0,
			priority: 1000,
			size:     int64(len(bz0)),
			key:      key0,
			sequence: 0,
			order:    mp.nextOrder(),
			tier:     mp.selectTier(sdkCtx, tx0),
			bytes:    bz0,
		}
		entry2 := &txEntry{
			tx:       tx2,
			priority: 1000,
			size:     int64(len(bz2)),
			key:      key2,
			sequence: 2,
			order:    mp.nextOrder(),
			tier:     mp.selectTier(sdkCtx, tx2),
			bytes:    bz2,
		}
		mp.addEntryLocked(entry0)
		mp.addEntryLocked(entry2)

		ss := mp.senders[sender.String()]
		require.NotNil(t, ss)
		require.Equal(t, uint64(0), ss.activeMin)
		require.Equal(t, uint64(2), ss.activeMax)

		// onChainSeq=5 clamps end to activeMax(2), loop must skip nonce=1 hole.
		removed := mp.removeActiveStaleLocked(ss, 5)
		mp.mtx.Unlock()

		require.Len(t, removed, 2)
		require.Equal(t, key0, removed[0].key)
		require.Equal(t, key2, removed[1].key)
		require.Equal(t, 0, mp.CountTx())
	})
}
