package abcipp

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/abcipp/types"
)

func TestQueryTxDistribution(t *testing.T) {
	tiers := []Tier{
		testTierMatcher("high"),
		testTierMatcher("low"),
	}
	mp := newTestPriorityMempool(t, tiers)
	ctx := sdk.WrapSDKContext(testSDKContext())

	txHigh := newTestTx(testAddress(1), 0, 1000, "high")
	txLow := newTestTx(testAddress(2), 0, 1000, "low")

	if err := mp.Insert(ctx, txHigh); err != nil {
		t.Fatalf("insert high tx: %v", err)
	}
	if err := mp.Insert(ctx, txLow); err != nil {
		t.Fatalf("insert low tx: %v", err)
	}

	server := &MempoolQueryServer{mempool: mp}
	resp, err := server.QueryTxDistribution(context.Background(), &types.QueryTxDistributionRequest{})
	if err != nil {
		t.Fatalf("query distribution: %v", err)
	}

	if resp.Distribution["high"] != 1 || resp.Distribution["low"] != 1 {
		t.Fatalf("unexpected distribution: %v", resp.Distribution)
	}
}

func TestQueryTxHashHexSender(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	ctx := sdk.WrapSDKContext(testSDKContext())

	priv := secp256k1.GenPrivKey()
	tx := newTestTxWithPriv(priv, 5, 1000, "default")
	if err := mp.Insert(ctx, tx); err != nil {
		t.Fatalf("insert tx: %v", err)
	}

	txBytes, err := testTxEncoder(tx)
	if err != nil {
		t.Fatalf("encode tx: %v", err)
	}

	senderHex := "0x" + hex.EncodeToString(sdk.AccAddress(priv.PubKey().Address()).Bytes())
	server := &MempoolQueryServer{mempool: mp}
	resp, err := server.QueryTxHash(context.Background(), &types.QueryTxHashRequest{
		Sender:   senderHex,
		Sequence: "5",
	})
	if err != nil {
		t.Fatalf("query tx hash: %v", err)
	}

	expected := TxHash(txBytes)
	if resp.TxHash != expected {
		t.Fatalf("unexpected hash: got %s expected %s", resp.TxHash, expected)
	}
}

func TestQueryTxHashNotFoundReturnsEmpty(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	server := &MempoolQueryServer{mempool: mp}

	resp, err := server.QueryTxHash(context.Background(), &types.QueryTxHashRequest{
		Sender:   "0x" + hex.EncodeToString(testAddress(10).Bytes()),
		Sequence: "1",
	})
	if err != nil {
		t.Fatalf("query tx hash: %v", err)
	}
	if resp.TxHash != "" {
		t.Fatalf("expected empty hash for missing tx, got %s", resp.TxHash)
	}
}

func TestQueryTxHashInvalidSender(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	server := &MempoolQueryServer{mempool: mp}

	_, err := server.QueryTxHash(context.Background(), &types.QueryTxHashRequest{
		Sender:   "0xzz",
		Sequence: "1",
	})
	if err == nil {
		t.Fatalf("expected error for invalid sender")
	}
}

func TestQueryTxHashInvalidSequence(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	server := &MempoolQueryServer{mempool: mp}

	_, err := server.QueryTxHash(context.Background(), &types.QueryTxHashRequest{
		Sender:   "0x" + hex.EncodeToString(testAddress(11).Bytes()),
		Sequence: "not-a-number",
	})
	if err == nil {
		t.Fatalf("expected error for invalid sequence")
	}
}
