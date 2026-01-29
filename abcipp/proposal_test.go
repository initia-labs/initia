package abcipp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdksigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	protov2 "google.golang.org/protobuf/proto"
)

type testTx struct {
	sender   sdk.AccAddress
	sequence uint64
	gas      uint64
	tier     string
	sig      sdksigning.SignatureV2
}

var _ authsigning.SigVerifiableTx = (*testTx)(nil)

func newTestTx(sender sdk.AccAddress, sequence uint64, gas uint64, tier string) *testTx {
	priv := secp256k1.GenPrivKey()
	sig := sdksigning.SignatureV2{
		PubKey: priv.PubKey(),
		Data: &sdksigning.SingleSignatureData{
			SignMode:  sdksigning.SignMode_SIGN_MODE_DIRECT,
			Signature: []byte{0x1},
		},
		Sequence: sequence,
	}
	return &testTx{
		sender:   sender,
		sequence: sequence,
		gas:      gas,
		tier:     tier,
		sig:      sig,
	}
}

func newTestTxWithPriv(priv cryptotypes.PrivKey, sequence uint64, gas uint64, tier string) *testTx {
	return &testTx{
		sender:   sdk.AccAddress(priv.PubKey().Address()),
		sequence: sequence,
		gas:      gas,
		tier:     tier,
		sig: sdksigning.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &sdksigning.SingleSignatureData{
				SignMode:  sdksigning.SignMode_SIGN_MODE_DIRECT,
				Signature: []byte{0x1},
			},
			Sequence: sequence,
		},
	}
}

func (tx *testTx) GetMsgs() []sdk.Msg {
	return nil
}

func (tx *testTx) GetMsgsV2() ([]protov2.Message, error) {
	return nil, nil
}

func (tx *testTx) ValidateBasic() error {
	return nil
}

func (tx *testTx) GetSigners() ([][]byte, error) {
	return [][]byte{tx.sender.Bytes()}, nil
}

func (tx *testTx) GetPubKeys() ([]cryptotypes.PubKey, error) {
	return []cryptotypes.PubKey{tx.sig.PubKey}, nil
}

func (tx *testTx) GetSignaturesV2() ([]sdksigning.SignatureV2, error) {
	return []sdksigning.SignatureV2{{
		PubKey:   tx.sig.PubKey,
		Data:     tx.sig.Data,
		Sequence: tx.sequence,
	}}, nil
}

func (tx *testTx) SetSignaturesV2(signatures []sdksigning.SignatureV2) error {
	if len(signatures) == 0 {
		return fmt.Errorf("no signatures provided")
	}
	tx.sig = signatures[0]
	return nil
}

func (tx *testTx) GetFee() sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1)))
}

func (tx *testTx) GetGas() uint64 {
	return tx.gas
}

func (tx *testTx) FeePayer() []byte {
	return tx.sender.Bytes()
}

func (tx *testTx) FeeGranter() []byte {
	return nil
}

func (tx *testTx) GetMemo() string {
	return ""
}

func (tx *testTx) GetTimeoutHeight() uint64 {
	return 0
}

type encodedTestTx struct {
	Sender   string `json:"sender"`
	Gas      uint64 `json:"gas"`
	Sequence uint64 `json:"sequence"`
	Tier     string `json:"tier"`
}

func testTxEncoder(tx sdk.Tx) ([]byte, error) {
	tt, ok := tx.(*testTx)
	if !ok {
		return nil, fmt.Errorf("unexpected tx type %T", tx)
	}
	enc := encodedTestTx{
		Sender:   hex.EncodeToString(tt.sender.Bytes()),
		Gas:      tt.gas,
		Sequence: tt.sequence,
		Tier:     tt.tier,
	}
	return json.Marshal(enc)
}

func testTxDecoder(bz []byte) (sdk.Tx, error) {
	var enc encodedTestTx
	if err := json.Unmarshal(bz, &enc); err != nil {
		return nil, err
	}
	raw, err := hex.DecodeString(enc.Sender)
	if err != nil {
		return nil, err
	}
	sender := sdk.AccAddress(raw)
	return newTestTx(sender, enc.Sequence, enc.Gas, enc.Tier), nil
}

func testSDKContext() sdk.Context {
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	if err := cms.LoadLatestVersion(); err != nil {
		panic(err)
	}

	return sdk.NewContext(cms, tmproto.Header{}, false, log.NewNopLogger()).
		WithConsensusParams(tmproto.ConsensusParams{
			Block: &tmproto.BlockParams{
				MaxBytes: 1 << 20,
				MaxGas:   1 << 20,
			},
		})
}

func testAddress(id int) sdk.AccAddress {
	sum := sha256.Sum256([]byte(fmt.Sprintf("addr-%d", id)))
	return sdk.AccAddress(sum[:20])
}

func testTierMatcher(name string) Tier {
	return Tier{
		Name: name,
		Matcher: func(_ sdk.Context, tx sdk.Tx) bool {
			tt, ok := tx.(*testTx)
			return ok && tt.tier == name
		},
	}
}

func newTestPriorityMempool(t *testing.T, tiers []Tier) *PriorityMempool {
	t.Helper()
	if len(tiers) == 0 {
		tiers = []Tier{{Name: "default", Matcher: func(sdk.Context, sdk.Tx) bool { return true }}}
	}
	return NewPriorityMempool(PriorityMempoolConfig{
		MaxTx: 1000,
		Tiers: tiers,
	}, testTxEncoder)
}

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
		<-start
		for j := 0; j < 200; j++ {
			tier := "high"
			if j%2 == 1 {
				tier = "low"
			}
			tx := newTestTx(testAddress(id), uint64(id*1000+j), 1000, tier)
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
		tx := newTestTx(testAddress(i), uint64(i), 1000, tier)
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
