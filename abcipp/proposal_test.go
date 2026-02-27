package abcipp

import (
	"sync"
	"testing"
	"time"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestPriorityMempoolConcurrentTierDistribution(t *testing.T) {
	t.Parallel()

	tiers := []Tier{
		testTierMatcher("high"),
		testTierMatcher("low"),
	}
	mp := newTestPriorityMempool(t, tiers)
	ctx := sdk.WrapSDKContext(testSDKContext())

	var wg sync.WaitGroup
	start := make(chan struct{})
	worker := func(id int) {
		defer wg.Done()
		priv := secp256k1.GenPrivKey()
		<-start
		for j := 0; j < 200; j++ {
			tier := "high"
			if j%2 == 1 {
				tier = "low"
			}
			tx := newTestTxWithPriv(priv, uint64(j), 1000, tier)
			_ = mp.Insert(ctx, tx)
		}
	}

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go worker(i)
	}

	close(start)
	wg.Wait()

	dist := mp.GetTxDistribution()
	if dist["high"] == 0 || dist["low"] == 0 {
		t.Fatalf("expected both tiers to have entries, got %v", dist)
	}
}

func TestProposalHandlerWithConcurrentMempool(t *testing.T) {
	tiers := []Tier{
		testTierMatcher("high"),
		testTierMatcher("low"),
	}
	mp := newTestPriorityMempool(t, tiers)

	sdkCtx := testSDKContext()
	wrappedCtx := sdk.WrapSDKContext(sdkCtx)

	for i := 0; i < 10; i++ {
		tier := "high"
		if i%2 == 1 {
			tier = "low"
		}
		tx := newTestTx(testAddress(i), 0, 1000, tier)
		if err := mp.Insert(wrappedCtx, tx); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	ante := func(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	}

	handler := NewProposalHandler(log.NewNopLogger(), testTxDecoder, testTxEncoder, mp, ante)

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var idx uint64
		for {
			select {
			case <-done:
				return
			default:
				tier := "high"
				if idx%2 == 1 {
					tier = "low"
				}
				tx := newTestTx(testAddress(int(idx%5)), 1000+idx, 1500, tier)
				_ = mp.Insert(wrappedCtx, tx)
				idx++
				time.Sleep(time.Millisecond)
			}
		}
	}()

	req := &abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: 1 << 20,
	}

	resp, err := handler.PrepareProposalHandler()(sdkCtx, req)
	close(done)
	wg.Wait()
	if err != nil {
		t.Fatalf("prepare proposal: %v", err)
	}

	processReq := &abci.RequestProcessProposal{
		Height: 2,
		Txs:    resp.Txs,
	}
	if _, err := handler.ProcessProposalHandler()(sdkCtx, processReq); err != nil {
		t.Fatalf("process proposal: %v", err)
	}
}

func TestPrepareProposalRemovesTxWhenConsensusParamsShrink(t *testing.T) {
	cases := []struct {
		name          string
		maxBytes      int64
		maxGas        int64
		txGas         uint64
		expectRemoved string
	}{
		{
			name:          "gas-shrink",
			maxBytes:      1 << 20,
			maxGas:        50,
			txGas:         80,
			expectRemoved: "expected tx to be removed after gas limit shrink",
		},
		{
			name:          "bytes-shrink",
			maxBytes:      0, // filled after encoding
			maxGas:        1 << 20,
			txGas:         1,
			expectRemoved: "expected tx to be removed after max bytes shrink",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mp := newTestPriorityMempool(t, nil)

			tx := newTestTx(testAddress(1), 0, tc.txGas, "default")
			txBytes, err := testTxEncoder(tx)
			if err != nil {
				t.Fatalf("encode tx: %v", err)
			}

			insertMaxBytes := tc.maxBytes
			if tc.name == "bytes-shrink" {
				insertMaxBytes = int64(len(txBytes)) + 10
				tc.maxBytes = int64(len(txBytes)) - 1
			}

			highLimitCtx := testSDKContextWithParams(insertMaxBytes, 1<<20)
			insertCtx := sdk.WrapSDKContext(highLimitCtx)
			if err := mp.Insert(insertCtx, tx); err != nil {
				t.Fatalf("insert: %v", err)
			}

			ante := func(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
				return ctx, nil
			}
			handler := NewProposalHandler(log.NewNopLogger(), testTxDecoder, testTxEncoder, mp, ante)

			lowLimitCtx := testSDKContextWithParams(tc.maxBytes, tc.maxGas)
			req := &abci.RequestPrepareProposal{
				Height:     2,
				MaxTxBytes: 1 << 20,
			}
			resp, err := handler.PrepareProposalHandler()(lowLimitCtx, req)
			if err != nil {
				t.Fatalf("prepare proposal: %v", err)
			}
			if len(resp.Txs) != 0 {
				t.Fatalf("expected no txs included, got %d", len(resp.Txs))
			}
			if mp.Contains(tx) {
				t.Fatalf("%s", tc.expectRemoved)
			}
		})
	}
}

func TestProcessProposalRejectsWhenConsensusParamsShrink(t *testing.T) {
	cases := []struct {
		name     string
		maxBytes int64
		maxGas   int64
		txGas    uint64
	}{
		{
			name:     "gas-shrink",
			maxBytes: 1 << 20,
			maxGas:   50,
			txGas:    80,
		},
		{
			name:     "bytes-shrink",
			maxBytes: 0, // filled after encoding
			maxGas:   1 << 20,
			txGas:    1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mp := newTestPriorityMempool(t, nil)
			ante := func(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
				return ctx, nil
			}
			handler := NewProposalHandler(log.NewNopLogger(), testTxDecoder, testTxEncoder, mp, ante)

			tx := newTestTx(testAddress(2), 1, tc.txGas, "default")
			txBytes, err := testTxEncoder(tx)
			if err != nil {
				t.Fatalf("encode tx: %v", err)
			}

			if tc.name == "bytes-shrink" {
				tc.maxBytes = int64(len(txBytes)) - 1
			}

			lowLimitCtx := testSDKContextWithParams(tc.maxBytes, tc.maxGas)
			req := &abci.RequestProcessProposal{
				Height: 2,
				Txs:    [][]byte{txBytes},
			}

			_, err = handler.ProcessProposalHandler()(lowLimitCtx, req)
			if err == nil {
				t.Fatalf("expected process proposal to error after params shrink")
			}
		})
	}
}

func TestPrepareProposalSkipsTxThatWouldOverflowButKeepsLaterTx(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	ctx := testSDKContextWithParams(1<<20, 10)
	wrappedCtx := sdk.WrapSDKContext(ctx)

	tx1 := newTestTx(testAddress(1), 0, 6, "default")
	tx2 := newTestTx(testAddress(2), 0, 6, "default")
	tx3 := newTestTx(testAddress(3), 0, 4, "default")

	if err := mp.Insert(wrappedCtx, tx1); err != nil {
		t.Fatalf("insert tx1: %v", err)
	}
	if err := mp.Insert(wrappedCtx, tx2); err != nil {
		t.Fatalf("insert tx2: %v", err)
	}
	if err := mp.Insert(wrappedCtx, tx3); err != nil {
		t.Fatalf("insert tx3: %v", err)
	}

	ante := func(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	}
	handler := NewProposalHandler(log.NewNopLogger(), testTxDecoder, testTxEncoder, mp, ante)

	req := &abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: 1 << 20,
	}
	resp, err := handler.PrepareProposalHandler()(ctx, req)
	if err != nil {
		t.Fatalf("prepare proposal: %v", err)
	}

	if len(resp.Txs) != 2 {
		t.Fatalf("expected 2 txs included, got %d", len(resp.Txs))
	}

	decoded, err := GetDecodedTxs(testTxDecoder, resp.Txs)
	if err != nil {
		t.Fatalf("decode txs: %v", err)
	}
	got := make([]sdk.AccAddress, 0, len(decoded))
	for _, tx := range decoded {
		tt, ok := tx.(*testTx)
		if !ok {
			t.Fatalf("unexpected tx type %T", tx)
		}
		got = append(got, tt.sender)
	}

	if len(got) != 2 || !got[0].Equals(tx1.sender) || !got[1].Equals(tx3.sender) {
		t.Fatalf("expected tx1 then tx3, got %v", got)
	}
	if !mp.Contains(tx2) {
		t.Fatalf("expected tx2 to remain in mempool after being skipped")
	}
}

func TestPrepareProposalSkipsTxWhenMaxBytesWouldOverflow(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)

	tx1 := newTestTx(testAddress(21), 0, 1, "a")
	tx2 := newTestTx(testAddress(22), 0, 1, "this-tier-is-much-longer")
	tx3 := newTestTx(testAddress(23), 0, 1, "b")

	tx1Bytes, err := testTxEncoder(tx1)
	if err != nil {
		t.Fatalf("encode tx1: %v", err)
	}
	tx2Bytes, err := testTxEncoder(tx2)
	if err != nil {
		t.Fatalf("encode tx2: %v", err)
	}
	tx3Bytes, err := testTxEncoder(tx3)
	if err != nil {
		t.Fatalf("encode tx3: %v", err)
	}

	maxBytes := int64(len(tx1Bytes)) + int64(len(tx3Bytes))
	if int64(len(tx1Bytes))+int64(len(tx2Bytes)) <= maxBytes {
		t.Fatalf("expected tx1+tx2 to overflow max bytes (sizes: tx1=%d tx2=%d tx3=%d max=%d)", len(tx1Bytes), len(tx2Bytes), len(tx3Bytes), maxBytes)
	}
	if int64(len(tx2Bytes)) > maxBytes {
		t.Fatalf("expected tx2 to fit individually under max bytes (tx2=%d max=%d)", len(tx2Bytes), maxBytes)
	}

	ctx := testSDKContextWithParams(maxBytes, 1<<20)
	insertCtx := sdk.WrapSDKContext(testSDKContextWithParams(maxBytes+100, 1<<20))

	if err := mp.Insert(insertCtx, tx1); err != nil {
		t.Fatalf("insert tx1: %v", err)
	}
	if err := mp.Insert(insertCtx, tx2); err != nil {
		t.Fatalf("insert tx2: %v", err)
	}
	if err := mp.Insert(insertCtx, tx3); err != nil {
		t.Fatalf("insert tx3: %v", err)
	}

	ante := func(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	}
	handler := NewProposalHandler(log.NewNopLogger(), testTxDecoder, testTxEncoder, mp, ante)

	req := &abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: 1 << 20,
	}
	resp, err := handler.PrepareProposalHandler()(ctx, req)
	if err != nil {
		t.Fatalf("prepare proposal: %v", err)
	}

	if len(resp.Txs) != 2 {
		t.Fatalf("expected 2 txs included, got %d", len(resp.Txs))
	}

	decoded, err := GetDecodedTxs(testTxDecoder, resp.Txs)
	if err != nil {
		t.Fatalf("decode txs: %v", err)
	}
	got := make([]sdk.AccAddress, 0, len(decoded))
	for _, tx := range decoded {
		tt, ok := tx.(*testTx)
		if !ok {
			t.Fatalf("unexpected tx type %T", tx)
		}
		got = append(got, tt.sender)
	}

	if len(got) != 2 || !got[0].Equals(tx1.sender) || !got[1].Equals(tx3.sender) {
		t.Fatalf("expected tx1 then tx3, got %v", got)
	}
	if !mp.Contains(tx2) {
		t.Fatalf("expected tx2 to remain in mempool after being skipped")
	}
}
