package lanes

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/block"
	blockbase "github.com/skip-mev/block-sdk/v2/block/base"
)

func RejectMatchHandler() blockbase.MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		return false
	}
}

const (
	// SystemLaneName defines the name of the system lane.
	SystemLaneName = "system"
)

// NewSystemLane returns a new system lane.
func NewSystemLane(
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
		SystemLaneName,
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
