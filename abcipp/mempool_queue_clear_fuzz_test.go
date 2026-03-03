//go:build fuzz

package abcipp

import (
	"context"
	"math/rand"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

type queueOp struct {
	sender string
	seq    uint64
}

func queueClearPattern(base uint64, count int) []uint64 {
	if count <= 3 {
		seqs := []uint64{base + 2, base, base + 1}
		return seqs[:count]
	}

	seqs := []uint64{base + 2, base, base + 1}
	for i := 3; i < count; i++ {
		seqs = append(seqs, base+uint64(i))
	}
	return seqs
}

func runQueueClearScenario(t *testing.T, senderCount int, txPerSender int, seed int64) {
	t.Helper()

	mp, keeper, sdkCtx, eventCh := newTestMempoolWithEvents(t, senderCount*txPerSender+16)
	mp.SetMaxQueuedPerSender(txPerSender + 4)
	drainEvents(eventCh)

	rng := rand.New(rand.NewSource(seed))

	privBySender := make(map[string]*secp256k1.PrivKey, senderCount)
	ops := make([]queueOp, 0, senderCount*txPerSender)
	for i := 0; i < senderCount; i++ {
		priv := secp256k1.GenPrivKey()
		sender := sdk.AccAddress(priv.PubKey().Address())
		keeper.SetSequence(sender, 0)
		privBySender[sender.String()] = priv

		for _, seq := range queueClearPattern(0, txPerSender) {
			ops = append(ops, queueOp{sender: sender.String(), seq: seq})
		}
	}

	rng.Shuffle(len(ops), func(i, j int) {
		ops[i], ops[j] = ops[j], ops[i]
	})

	for _, op := range ops {
		priority := int64(rng.Intn(1000) + 1)
		ctx := sdk.WrapSDKContext(sdkCtx.WithPriority(priority))
		tx := newTestTxWithPriv(privBySender[op.sender], op.seq, 1000, "default")
		require.NoError(t, mp.Insert(ctx, tx))
	}

	committed := make(map[string][]uint64, senderCount)
	maxSteps := senderCount*txPerSender*6 + 32

	for step := 0; step < maxSteps; step++ {
		if mp.CountTx() == 0 {
			break
		}

		it := mp.Select(context.Background(), nil)
		if it == nil {
			before := mp.CountTx()
			mp.PromoteQueued(sdk.WrapSDKContext(sdkCtx))
			after := mp.CountTx()
			if after == before && mp.Select(context.Background(), nil) == nil {
				t.Fatalf("queue-clear stuck: no active tx and no promotion progress (remaining=%d)", after)
			}
			continue
		}

		tx := it.Tx()
		info := it.(TxInfoIterator).TxInfo()
		key, err := txKeyFromTx(tx)
		require.NoError(t, err)
		if info.Sender != key.sender || info.Sequence != key.nonce {
			t.Fatalf(
				"iterator mismatch: info=(%s,%d) txKey=(%s,%d)",
				info.Sender, info.Sequence, key.sender, key.nonce,
			)
		}
		require.NoError(t, mp.Remove(tx))
		committed[key.sender] = append(committed[key.sender], key.nonce)
		addr, err := sdk.AccAddressFromBech32(key.sender)
		require.NoError(t, err)
		keeper.SetSequence(addr, key.nonce+1)

		// Mimic periodic queue promotion in runtime.
		mp.PromoteQueued(sdk.WrapSDKContext(sdkCtx))
	}

	require.Equal(t, 0, mp.CountTx(), "mempool should be fully drained")

	for sender, priv := range privBySender {
		got := committed[sender]
		require.Len(t, got, txPerSender, "sender %s committed count mismatch, got=%v", sender, got)

		for i, seq := range got {
			require.Equal(t, uint64(i), seq, "sender %s committed out of order", sdk.AccAddress(priv.PubKey().Address()).String())
		}
	}
}

func FuzzQueueClearOrderingScenario(f *testing.F) {
	f.Add(uint8(5), uint8(10), int64(1))
	f.Add(uint8(3), uint8(7), int64(42))
	f.Add(uint8(2), uint8(9), int64(99))

	f.Fuzz(func(t *testing.T, senderCountRaw uint8, txPerSenderRaw uint8, seed int64) {
		senderCount := int(senderCountRaw%6) + 1
		// Queue-clear scenario requires predecessor nonces to exist; enforce
		// at least three txs per sender so pattern always includes 2,0,1.
		txPerSender := int(txPerSenderRaw%10) + 3
		runQueueClearScenario(t, senderCount, txPerSender, seed)
	})
}
