package lanes_test

import (
	"testing"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	lanes "github.com/initia-labs/initia/v1/app/lanes"
)

func Test_FreeLaneMatchHandler(t *testing.T) {
	ctx := sdk.NewContext(nil, types.Header{}, false, log.NewNopLogger())

	handler := lanes.FreeLaneMatchHandler()
	require.True(t, handler(ctx, MockTx{
		msgs: []sdk.Msg{
			&clienttypes.MsgUpdateClient{},
			&channeltypes.MsgTimeout{},
			&channeltypes.MsgAcknowledgement{},
		},
	}))

	require.False(t, handler(ctx, MockTx{
		msgs: []sdk.Msg{
			&clienttypes.MsgUpdateClient{},
			&banktypes.MsgSend{},
		},
	}))
}

var _ sdk.Tx = MockTx{}
var _ sdk.FeeTx = &MockTx{}

type MockTx struct {
	msgs     []sdk.Msg
	gasLimit uint64
}

func (tx MockTx) GetMsgsV2() ([]protov2.Message, error) {
	return nil, nil
}

func (tx MockTx) GetMsgs() []sdk.Msg {
	return tx.msgs
}

func (tx MockTx) GetGas() uint64 {
	return tx.gasLimit
}

func (tx MockTx) GetFee() sdk.Coins {
	return nil
}

func (tx MockTx) FeePayer() []byte {
	return nil
}
func (tx MockTx) FeeGranter() []byte {
	return nil
}
