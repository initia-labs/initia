package lanes

import (
	"github.com/skip-mev/block-sdk/v2/block"
	blockbase "github.com/skip-mev/block-sdk/v2/block/base"
)

const (
	// DefaultName defines the name of the priority lane.
	DefaultName = "default"
)

// NewDefaultLane defines a default lane implementation. The default lane orders
// transactions by the transaction fees. The default lane accepts any transaction
// that should not be ignored (as defined by the IgnoreList in the LaneConfig).
// The default lane builds and verifies blocks in a similar fashion to how the
// CometBFT/Tendermint consensus engine builds and verifies blocks pre SDK version
// 0.47.0.
func NewDefaultLane(cfg blockbase.LaneConfig) block.Lane {
	lane := &blockbase.BaseLane{}
	proposalHandler := NewDefaultProposalHandler(lane)

	_lane, err := blockbase.NewBaseLane(
		cfg,
		DefaultName,
		blockbase.WithMempool(NewMempool(blockbase.NewDefaultTxPriority(), cfg.SignerExtractor, cfg.MaxTxs)),
		blockbase.WithPrepareLaneHandler(proposalHandler.PrepareLaneHandler()),
		blockbase.WithProcessLaneHandler(proposalHandler.ProcessLaneHandler()),
	)
	if err != nil {
		panic(err)
	}

	*lane = *_lane
	return lane
}
