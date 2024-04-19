package lanes

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/block"
	blockbase "github.com/skip-mev/block-sdk/v2/block/base"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
)

// SystemLaneMatchHandler returns the default match handler for the system lane.
func SystemLaneMatchHandler() blockbase.MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		if len(tx.GetMsgs()) != 1 {
			return false
		}

		for _, msg := range tx.GetMsgs() {
			switch msg.(type) {
			case *opchildtypes.MsgUpdateOracle:
			default:
				return false
			}
		}

		return true
	}
}

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

	_lane, err := blockbase.NewBaseLane(
		cfg,
		SystemLaneName,
		blockbase.WithMatchHandler(matchFn),
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
