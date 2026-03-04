//go:build fuzz

package abcipp

import (
	"context"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/stretchr/testify/require"
)

type queueOp struct {
	sender string
	seq    uint64
}

func queueClearPattern(base uint64, count int, rng *rand.Rand) []uint64 {
	seqs := make([]uint64, count)
	for i := 0; i < count; i++ {
		seqs[i] = base + uint64(i)
	}

	rng.Shuffle(len(seqs), func(i, j int) {
		seqs[i], seqs[j] = seqs[j], seqs[i]
	})

	// Keep the scenario non-trivial even if shuffle returns sorted order by chance.
	isSorted := true
	for i := 1; i < len(seqs); i++ {
		if seqs[i-1] > seqs[i] {
			isSorted = false
			break
		}
	}
	if isSorted && len(seqs) > 1 {
		seqs[0], seqs[1] = seqs[1], seqs[0]
	}

	return seqs
}

func runValidatorConcurrentQueueClearScenario(t *testing.T, senderCount int, txPerSender int, seed int64) {
	t.Helper()

	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, senderCount*txPerSender+16)
	mp.SetMaxQueuedPerSender(txPerSender + 4)
	mp.cfg.AnteHandler = func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	}
	drainEvents(eventCh)

	rng := rand.New(rand.NewSource(seed))

	privBySender := make(map[string]*secp256k1.PrivKey, senderCount)
	seqsBySender := make(map[string][]uint64, senderCount)
	senders := make([]string, 0, senderCount)
	for i := 0; i < senderCount; i++ {
		priv := secp256k1.GenPrivKey()
		sender := sdk.AccAddress(priv.PubKey().Address())
		keeper.SetSequence(sender, 0)
		senderStr := sender.String()
		privBySender[senderStr] = priv
		senders = append(senders, senderStr)
		seqsBySender[senderStr] = queueClearPattern(0, txPerSender, rng)
	}

	doneCh := make(chan struct{})
	errCh := make(chan error, senderCount)
	var wg sync.WaitGroup
	wg.Add(len(senders))
	for i, sender := range senders {
		sender := sender
		localSeed := seed + int64(i+1)*1000003
		go func() {
			defer wg.Done()
			localRng := rand.New(rand.NewSource(localSeed))
			for _, seq := range seqsBySender[sender] {
				priority := int64(localRng.Intn(1000) + 1)
				ctx := sdk.WrapSDKContext(sdkCtx.WithPriority(priority))
				tx := newTestTxWithPriv(privBySender[sender], seq, 1000, "default")
				if err := mp.Insert(ctx, tx); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(doneCh)
		close(errCh)
	}()

	committed := make(map[string][]uint64, senderCount)
	inserterDone := false
	insertDoneAt := time.Time{}
	maxIdleLoops := senderCount*txPerSender*8 + 128
	idleLoops := 0
	reconcileStepsAfterInsert := 0

	for {
		it := mp.Select(context.Background(), nil)
		if it == nil {
			mp.PromoteQueued(sdk.WrapSDKContext(sdkCtx))

			if !inserterDone {
				select {
				case <-doneCh:
					inserterDone = true
					insertDoneAt = time.Now()
				default:
				}
			}
			if inserterDone && mp.CountTx() == 0 {
				break
			}
			if inserterDone {
				reconcileStepsAfterInsert++
			}

			idleLoops++
			if idleLoops > maxIdleLoops {
				t.Fatalf("validator scenario stuck: inserterDone=%v remaining=%d", inserterDone, mp.CountTx())
			}
			time.Sleep(time.Millisecond)
			continue
		}

		idleLoops = 0
		tx := it.Tx()
		key, err := txKeyFromTx(tx)
		require.NoError(t, err)

		err = mp.Remove(tx)
		if err != nil {
			require.ErrorIs(t, err, sdkmempool.ErrTxNotFound)
			continue
		}

		committed[key.sender] = append(committed[key.sender], key.nonce)
		addr, err := sdk.AccAddressFromBech32(key.sender)
		require.NoError(t, err)
		keeper.SetSequence(addr, key.nonce+1)
		mp.PromoteQueued(sdk.WrapSDKContext(sdkCtx))
	}

	for err := range errCh {
		require.NoError(t, err)
	}
	require.Equal(t, 0, mp.CountTx(), "mempool should be fully drained")
	require.NoError(t, mp.ValidateInvariants())
	require.False(t, insertDoneAt.IsZero(), "insert completion timestamp should be set")
	t.Logf(
		"validator clear latency after all inserts: %s (reconcile_steps=%d)",
		time.Since(insertDoneAt),
		reconcileStepsAfterInsert,
	)

	for sender, priv := range privBySender {
		got := committed[sender]
		require.Len(t, got, txPerSender, "sender %s committed count mismatch, got=%v", sender, got)
		for i, seq := range got {
			require.Equal(t, uint64(i), seq, "sender %s committed out of order", sdk.AccAddress(priv.PubKey().Address()).String())
		}
	}
}

func runNonValidatorQueueClearScenario(t *testing.T, senderCount int, txPerSender int, seed int64) {
	t.Helper()

	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, senderCount*txPerSender+16)
	mp.SetMaxQueuedPerSender(txPerSender + 4)
	mp.cfg.AnteHandler = func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	}
	drainEvents(eventCh)

	rng := rand.New(rand.NewSource(seed))

	privBySender := make(map[string]*secp256k1.PrivKey, senderCount)
	senderAddr := make(map[string]sdk.AccAddress, senderCount)
	seqsBySender := make(map[string][]uint64, senderCount)
	senders := make([]string, 0, senderCount)
	for i := 0; i < senderCount; i++ {
		priv := secp256k1.GenPrivKey()
		addr := sdk.AccAddress(priv.PubKey().Address())
		sender := addr.String()
		keeper.SetSequence(addr, 0)
		privBySender[sender] = priv
		senderAddr[sender] = addr
		senders = append(senders, sender)
		seqsBySender[sender] = queueClearPattern(0, txPerSender, rng)
	}

	errCh := make(chan error, senderCount)
	var wg sync.WaitGroup
	wg.Add(len(senders))
	doneCh := make(chan struct{})
	for i, sender := range senders {
		sender := sender
		localSeed := seed + int64(i+1)*2000003
		go func() {
			defer wg.Done()
			localRng := rand.New(rand.NewSource(localSeed))
			for _, seq := range seqsBySender[sender] {
				priority := int64(localRng.Intn(1000) + 1)
				ctx := sdk.WrapSDKContext(sdkCtx.WithPriority(priority))
				tx := newTestTxWithPriv(privBySender[sender], seq, 1000, "default")
				if err := mp.Insert(ctx, tx); err != nil {
					// In this scenario Remove(tx0) can race with late nonce-0 insertion.
					// Treat stale nonce-0 as expected under concurrent reconcile.
					if seq == 0 && strings.Contains(err.Error(), "is stale for sender") {
						continue
					}
					errCh <- err
					return
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(doneCh)
		close(errCh)
	}()

	removedByAPI := 0
	removedBySender := make(map[string]bool, senderCount)
	seqAdvancedBySender := make(map[string]bool, senderCount)
	inserterDone := false
	insertDoneAt := time.Time{}
	postInsertRemoveTried := false
	maxLoops := senderCount*txPerSender*12 + 256
	reconcileStepsAfterInsert := 0

	for step := 0; step < maxLoops; step++ {
		if !inserterDone {
			select {
			case <-doneCh:
				inserterDone = true
				insertDoneAt = time.Now()
			default:
			}
		}

		for _, sender := range senders {
			if !removedBySender[sender] {
				tx0 := newTestTxWithPriv(privBySender[sender], 0, 1000, "default")
				err := mp.Remove(tx0)
				if err == nil {
					removedBySender[sender] = true
					removedByAPI++
				} else {
					require.ErrorIs(t, err, sdkmempool.ErrTxNotFound)
				}
			}

			if inserterDone && !seqAdvancedBySender[sender] {
				keeper.SetSequence(senderAddr[sender], uint64(txPerSender))
				seqAdvancedBySender[sender] = true
			}
		}
		if inserterDone && !postInsertRemoveTried {
			postInsertRemoveTried = true
			if it := mp.Select(context.Background(), nil); it != nil {
				if err := mp.Remove(it.Tx()); err == nil {
					removedByAPI++
				}
			}
		}

		before := mp.CountTx()
		mp.PromoteQueued(sdk.WrapSDKContext(sdkCtx))
		mp.cleanUpEntries(testBaseApp{ctx: sdkCtx}, keeper)
		after := mp.CountTx()

		if inserterDone {
			reconcileStepsAfterInsert++
		}
		if inserterDone && after == 0 {
			break
		}
		if after == before {
			time.Sleep(time.Millisecond)
		}
	}

	for err := range errCh {
		require.NoError(t, err)
	}

	advancedCount := 0
	for _, sender := range senders {
		if seqAdvancedBySender[sender] {
			advancedCount++
		}
	}
	require.False(t, insertDoneAt.IsZero(), "insert completion timestamp should be set")
	require.Greater(t, removedByAPI, 0, "scenario should remove some txs via mp.Remove")
	require.Equal(t, senderCount, advancedCount, "scenario should advance account sequence for all senders")
	require.Equal(t, 0, mp.CountTx(), "non-validator reconcile should fully clear mempool")
	require.NoError(t, mp.ValidateInvariants())
	t.Logf(
		"non-validator clear latency after all inserts: %s (reconcile_steps=%d)",
		time.Since(insertDoneAt),
		reconcileStepsAfterInsert,
	)
}

func FuzzValidatorConcurrentQueueClearScenario(f *testing.F) {
	f.Add(uint8(5), uint8(10), int64(1))
	f.Add(uint8(3), uint8(7), int64(42))
	f.Add(uint8(2), uint8(9), int64(99))

	f.Fuzz(func(t *testing.T, senderCountRaw uint8, txPerSenderRaw uint8, seed int64) {
		senderCount := int(senderCountRaw%6) + 1
		txPerSender := int(txPerSenderRaw%10) + 3
		runValidatorConcurrentQueueClearScenario(t, senderCount, txPerSender, seed)
	})
}

func FuzzNonValidatorQueueClearScenario(f *testing.F) {
	f.Add(uint8(5), uint8(10), int64(7))
	f.Add(uint8(3), uint8(7), int64(17))
	f.Add(uint8(2), uint8(9), int64(777))

	f.Fuzz(func(t *testing.T, senderCountRaw uint8, txPerSenderRaw uint8, seed int64) {
		senderCount := int(senderCountRaw%6) + 1
		txPerSender := int(txPerSenderRaw%10) + 3
		runNonValidatorQueueClearScenario(t, senderCount, txPerSender, seed)
	})
}
