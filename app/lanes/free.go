package lanes

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/skip-mev/block-sdk/v2/block"
	blockbase "github.com/skip-mev/block-sdk/v2/block/base"
)

// FreeLaneMatchHandler returns the default match handler for the free lane. The
// default implementation matches transactions that are ibc related. In particular,
// any transaction that is a MsgUpdateClient, MsgTimeout, MsgAcknowledgement.
func FreeLaneMatchHandler() blockbase.MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		for _, msg := range tx.GetMsgs() {
			switch msg.(type) {
			case *clienttypes.MsgUpdateClient:
			case *channeltypes.MsgTimeout:
			case *channeltypes.MsgAcknowledgement:
			default:
				return false
			}
		}

		return true
	}
}

const (
	// FreeLaneName defines the name of the free lane.
	FreeLaneName = "free"
)

// NewFreeLane returns a new free lane.
func NewFreeLane(
	cfg blockbase.LaneConfig,
	matchFn blockbase.MatchHandler,
) block.Lane {
	lane := &blockbase.BaseLane{}
	proposalHandler := NewDefaultProposalHandler(lane)

	mempool, err := NewMempool(
		blockbase.NewDefaultTxPriority(),
		cfg.SignerExtractor,
		cfg.MaxTxs,
		cfg.MaxBlockSpace,
		cfg.TxEncoder,
	)
	if err != nil {
		panic(err)
	}

	_lane, err := blockbase.NewBaseLane(
		cfg,
		FreeLaneName,
		blockbase.WithMatchHandler(matchFn),
		blockbase.WithMempool(mempool),
		blockbase.WithPrepareLaneHandler(proposalHandler.PrepareLaneHandler()),
		blockbase.WithProcessLaneHandler(proposalHandler.ProcessLaneHandler()),
	)
	if err != nil {
		panic(err)
	}

	*lane = *_lane
	return lane
}
