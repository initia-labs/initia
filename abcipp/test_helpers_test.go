package abcipp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
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

func testSDKContextWithParams(maxBytes int64, maxGas int64) sdk.Context {
	ctx := testSDKContext()
	return ctx.WithConsensusParams(tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{
			MaxBytes: maxBytes,
			MaxGas:   maxGas,
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
