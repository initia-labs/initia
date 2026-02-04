package abcipp

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/core/address"
	txsigning "cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/tx/signing/direct"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func TestPriorityMempoolRejectsOutOfOrderSequences(t *testing.T) {
	keeper := newMockAccountKeeper()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	keeper.SetSequence(sender, 5)

	mp := NewPriorityMempool(PriorityMempoolConfig{
		MaxTx: 10,
	}, testTxEncoder)

	sdkCtx := testSDKContext()
	if _, _, err := mp.NextExpectedSequence(sdkCtx, sender.String()); err != nil {
		t.Fatalf("expected to fetch initial account sequence: %v", err)
	}
	ctx := sdk.WrapSDKContext(sdkCtx)

	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 5, 1000, "default")); err != nil {
		t.Fatalf("failed to insert initial tx: %v", err)
	}

	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 7, 1000, "default")); err == nil {
		t.Fatalf("expected sequence gap to be rejected")
	}

	if err := mp.Insert(ctx, newTestTxWithPriv(priv, 6, 1000, "default")); err != nil {
		t.Fatalf("failed to insert sequential tx: %v", err)
	}
}

func TestPriorityMempoolNextExpectedSequenceLifecycle(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	priv := secp256k1.GenPrivKey()
	tx := newTestTxWithPriv(priv, 5, 1000, "default")

	if _, ok, err := mp.NextExpectedSequence(sdkCtx, tx.sender.String()); err != nil {
		t.Fatalf("fetch initial sequence: %v", err)
	} else if ok {
		t.Fatalf("expected no entry for sender before insert")
	}

	if err := mp.Insert(ctx, tx); err != nil {
		t.Fatalf("insert tx: %v", err)
	}

	if seq, ok, err := mp.NextExpectedSequence(sdkCtx, tx.sender.String()); err != nil {
		t.Fatalf("fetch expected sequence: %v", err)
	} else if !ok || seq != 6 {
		t.Fatalf("expected next sequence 6, got %d (ok=%t)", seq, ok)
	}

	if !mp.Contains(tx) {
		t.Fatalf("expected tx to be tracked")
	}

	info, err := mp.GetTxInfo(sdkCtx, tx)
	if err != nil {
		t.Fatalf("get tx info: %v", err)
	}
	if info.Sequence != 5 || info.Tier != "default" {
		t.Fatalf("unexpected tx info %+v", info)
	}

	if hash, ok := mp.Lookup(tx.sender.String(), tx.sequence); !ok || hash == "" {
		t.Fatalf("lookup failed")
	}

	if err := mp.Remove(tx); err != nil {
		t.Fatalf("remove tx: %v", err)
	}

	if _, ok, err := mp.NextExpectedSequence(sdkCtx, tx.sender.String()); err != nil {
		t.Fatalf("fetch after removal: %v", err)
	} else if ok {
		t.Fatalf("expected sender to reset after removal")
	}

	if _, err := mp.GetTxInfo(sdkCtx, tx); err != sdkmempool.ErrTxNotFound {
		t.Fatalf("expected ErrTxNotFound after removal, got %v", err)
	}
}

func TestPriorityMempoolSelectOrdersByTierAndTracksDistribution(t *testing.T) {
	tiers := []Tier{
		testTierMatcher("high"),
		testTierMatcher("low"),
	}
	mp := newTestPriorityMempool(t, tiers)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	lowTx := newTestTx(testAddress(1), 1, 1000, "low")
	highTx := newTestTx(testAddress(2), 1, 1000, "high")

	if err := mp.Insert(ctx, lowTx); err != nil {
		t.Fatalf("insert low priority: %v", err)
	}
	if err := mp.Insert(ctx, highTx); err != nil {
		t.Fatalf("insert high priority: %v", err)
	}

	order := make([]string, 0, 2)
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		tt, ok := it.Tx().(*testTx)
		if !ok {
			t.Fatalf("unexpected tx type %T", it.Tx())
		}
		order = append(order, tt.tier)
	}

	if len(order) != 2 {
		t.Fatalf("expected two entries, got %d", len(order))
	}
	if order[0] == order[1] {
		t.Fatalf("expected each tier once, got %v", order)
	}

	dist := mp.GetTxDistribution()
	if dist["high"] != 1 || dist["low"] != 1 {
		t.Fatalf("unexpected distribution %v", dist)
	}
}

func TestPriorityMempoolCleanUpEntries(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)
	baseApp := testBaseApp{ctx: sdkCtx}

	priv := secp256k1.GenPrivKey()
	tx1 := newTestTxWithPriv(priv, 1, 1000, "default")
	tx2 := newTestTxWithPriv(priv, 2, 1000, "default")

	if err := mp.Insert(ctx, tx1); err != nil {
		t.Fatalf("insert first tx: %v", err)
	}
	if err := mp.Insert(ctx, tx2); err != nil {
		t.Fatalf("insert second tx: %v", err)
	}

	keeper := newMockAccountKeeper()
	keeper.SetSequence(tx1.sender, 3)

	mp.cleanUpEntries(baseApp, keeper)

	if mp.CountTx() != 0 {
		t.Fatalf("expected stale entries removed, still have %d", mp.CountTx())
	}

	if mp.Contains(tx1) || mp.Contains(tx2) {
		t.Fatalf("stale entries should be gone")
	}

	if _, ok, err := mp.NextExpectedSequence(sdkCtx, tx1.sender.String()); err != nil {
		t.Fatalf("fetch after cleanup: %v", err)
	} else if ok {
		t.Fatalf("expected sender reset after cleanup")
	}
}

func TestPriorityMempoolCleanUpAnteErrors(t *testing.T) {
	accountKeeper := newTestAccountKeeper(authcodec.NewBech32Codec("initia"))
	bankKeeper := newTestBankKeeper()
	signModeHandler := txsigning.NewHandlerMap(direct.SignModeHandler{})

	anteHandler, err := authante.NewAnteHandler(authante.HandlerOptions{
		AccountKeeper:   accountKeeper,
		BankKeeper:      bankKeeper,
		SignModeHandler: signModeHandler,
	})
	if err != nil {
		t.Fatalf("failed to build ante handler: %v", err)
	}

	mp := NewPriorityMempool(PriorityMempoolConfig{
		MaxTx:       10,
		AnteHandler: anteHandler,
	}, testTxEncoder)

	sdkCtx := testSDKContext()
	priv := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(priv.PubKey().Address())
	accountKeeper.SetAccount(sdkCtx, authtypes.NewBaseAccountWithAddress(sender))
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx := newTestTxWithPriv(priv, 0, 1000, "default")
	txBytes, err := testTxEncoder(tx)
	if err != nil {
		t.Fatalf("encode tx: %v", err)
	}
	recheckCtx := sdkCtx.WithTxBytes(txBytes).WithIsReCheckTx(true)
	if _, err := anteHandler(recheckCtx, tx, false); err == nil {
		t.Fatalf("expected ante to reject tx without funds")
	}
	if err := mp.Insert(ctx, tx); err != nil {
		t.Fatalf("insert tx: %v", err)
	}

	senderKey := tx.sender.String()
	buckets := mp.snapshotBuckets()
	bucket, ok := buckets[senderKey]
	if !ok {
		t.Fatalf("bucket not found for sender")
	}
	invalid := bucket.collectInvalid(sdkCtx, mp.cfg.AnteHandler, 0)
	if len(invalid) == 0 {
		t.Fatalf("expected collectInvalid to detect the tx as invalid")
	}

	baseApp := testBaseApp{ctx: sdkCtx}
	mp.cleanUpEntries(baseApp, accountKeeper)

	if mp.CountTx() != 0 {
		t.Fatalf("expected ante failure to remove tx, still have %d", mp.CountTx())
	}

	if mp.Contains(tx) {
		t.Fatalf("expected tx removed after ante failure")
	}
}

func TestPriorityMempoolPreservesInsertionOrder(t *testing.T) {
	mp := newTestPriorityMempool(t, nil)
	sdkCtx := testSDKContext()
	ctx := sdk.WrapSDKContext(sdkCtx)

	tx1 := newTestTx(testAddress(1), 1, 1000, "default")
	tx2 := newTestTx(testAddress(2), 1, 1000, "default")

	if err := mp.Insert(ctx, tx1); err != nil {
		t.Fatalf("insert first tx: %v", err)
	}
	if err := mp.Insert(ctx, tx2); err != nil {
		t.Fatalf("insert second tx: %v", err)
	}

	var order []sdk.AccAddress
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		tt, ok := it.Tx().(*testTx)
		if !ok {
			t.Fatalf("unexpected tx type %T", it.Tx())
		}
		order = append(order, tt.sender)
	}

	if len(order) != 2 {
		t.Fatalf("expected two entries, got %d", len(order))
	}
	if !order[0].Equals(tx1.sender) || !order[1].Equals(tx2.sender) {
		t.Fatalf("expected FIFO insertion order, got %v", order)
	}
}

func TestPriorityMempoolOrdersByTierPriorityAndOrder(t *testing.T) {
	tiers := []Tier{
		testTierMatcher("high"),
		testTierMatcher("low"),
	}
	mp := newTestPriorityMempool(t, tiers)
	sdkCtx := testSDKContext()

	ctxPriority5 := sdk.WrapSDKContext(sdkCtx.WithPriority(5))
	ctxPriority10 := sdk.WrapSDKContext(sdkCtx.WithPriority(10))
	ctxPriority100 := sdk.WrapSDKContext(sdkCtx.WithPriority(100))

	highLowPriority1 := newTestTx(testAddress(1), 1, 1000, "high")
	highHighPriority := newTestTx(testAddress(2), 1, 1000, "high")
	highLowPriority2 := newTestTx(testAddress(3), 1, 1000, "high")
	lowHighPriority := newTestTx(testAddress(4), 1, 1000, "low")

	if err := mp.Insert(ctxPriority100, lowHighPriority); err != nil {
		t.Fatalf("insert low tier high priority: %v", err)
	}
	if err := mp.Insert(ctxPriority5, highLowPriority1); err != nil {
		t.Fatalf("insert high tier low priority #1: %v", err)
	}
	if err := mp.Insert(ctxPriority10, highHighPriority); err != nil {
		t.Fatalf("insert high tier high priority: %v", err)
	}
	if err := mp.Insert(ctxPriority5, highLowPriority2); err != nil {
		t.Fatalf("insert high tier low priority #2: %v", err)
	}

	var order []sdk.AccAddress
	for it := mp.Select(context.Background(), nil); it != nil; it = it.Next() {
		tt, ok := it.Tx().(*testTx)
		if !ok {
			t.Fatalf("unexpected tx type %T", it.Tx())
		}
		order = append(order, tt.sender)
	}

	expected := []sdk.AccAddress{
		highHighPriority.sender,
		highLowPriority1.sender,
		highLowPriority2.sender,
		lowHighPriority.sender,
	}
	if len(order) != len(expected) {
		t.Fatalf("expected %d entries, got %d", len(expected), len(order))
	}
	for idx, sender := range expected {
		if !order[idx].Equals(sender) {
			t.Fatalf("unexpected order at %d: got %v expected %v", idx, order[idx], sender)
		}
	}
}

type mockAccountKeeper struct {
	sequences map[string]uint64
}

func newMockAccountKeeper() *mockAccountKeeper {
	return &mockAccountKeeper{
		sequences: make(map[string]uint64),
	}
}

func (m *mockAccountKeeper) SetSequence(addr sdk.AccAddress, seq uint64) {
	m.sequences[string(addr.Bytes())] = seq
}

func (m *mockAccountKeeper) GetSequence(_ context.Context, addr sdk.AccAddress) (uint64, error) {
	key := string(addr.Bytes())
	seq, ok := m.sequences[key]
	if !ok {
		return 0, fmt.Errorf("sequence not found for %s", addr)
	}
	return seq, nil
}

type testBaseApp struct {
	ctx sdk.Context
}

func (b testBaseApp) GetContextForSimulate(_ []byte) sdk.Context {
	return b.ctx
}

type testAccountKeeper struct {
	accounts     map[string]sdk.AccountI
	params       authtypes.Params
	moduleAddrs  map[string]sdk.AccAddress
	addressCodec address.Codec
}

func newTestAccountKeeper(codec address.Codec) *testAccountKeeper {
	moduleAddr := sdk.AccAddress(bytes.Repeat([]byte{0xA}, 20))
	return &testAccountKeeper{
		accounts: make(map[string]sdk.AccountI),
		params:   authtypes.DefaultParams(),
		moduleAddrs: map[string]sdk.AccAddress{
			authtypes.FeeCollectorName: moduleAddr,
		},
		addressCodec: codec,
	}
}

func (k *testAccountKeeper) GetParams(ctx context.Context) authtypes.Params {
	return k.params
}

func (k *testAccountKeeper) GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return k.accounts[addr.String()]
}

func (k *testAccountKeeper) SetAccount(ctx context.Context, acc sdk.AccountI) {
	k.accounts[acc.GetAddress().String()] = acc
}

func (k *testAccountKeeper) GetModuleAddress(name string) sdk.AccAddress {
	return k.moduleAddrs[name]
}

func (k *testAccountKeeper) AddressCodec() address.Codec {
	return k.addressCodec
}

func (k *testAccountKeeper) GetSequence(ctx context.Context, addr sdk.AccAddress) (uint64, error) {
	account := k.GetAccount(ctx, addr)
	if account == nil {
		return 0, fmt.Errorf("account not found for %s", addr)
	}
	return account.GetSequence(), nil
}

type testBankKeeper struct {
	balances    map[string]sdk.Coins
	moduleCoins map[string]sdk.Coins
}

func newTestBankKeeper() *testBankKeeper {
	return &testBankKeeper{
		balances:    make(map[string]sdk.Coins),
		moduleCoins: make(map[string]sdk.Coins),
	}
}

func (b *testBankKeeper) IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error {
	return nil
}

func (b *testBankKeeper) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	if !b.hasBalance(from, amt) {
		return sdkerrors.ErrInsufficientFunds
	}
	b.debit(from, amt)
	b.balances[to.String()] = b.balance(to).Add(amt...)
	return nil
}

func (b *testBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	if !b.hasBalance(senderAddr, amt) {
		return sdkerrors.ErrInsufficientFunds
	}
	b.debit(senderAddr, amt)
	b.moduleCoins[recipientModule] = b.moduleCoins[recipientModule].Add(amt...)
	return nil
}

func (b *testBankKeeper) balance(addr sdk.AccAddress) sdk.Coins {
	coins := b.balances[addr.String()]
	if coins == nil {
		return sdk.NewCoins()
	}
	return coins
}

func (b *testBankKeeper) hasBalance(addr sdk.AccAddress, amt sdk.Coins) bool {
	return b.balance(addr).IsAllGTE(amt)
}

func (b *testBankKeeper) debit(addr sdk.AccAddress, amt sdk.Coins) {
	remaining := b.balance(addr).Sub(amt...)
	if remaining.IsZero() {
		delete(b.balances, addr.String())
		return
	}
	b.balances[addr.String()] = remaining
}
