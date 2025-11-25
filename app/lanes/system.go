package lanes

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/block-sdk/v2/block"
	blockbase "github.com/skip-mev/block-sdk/v2/block/base"

	blocklanekeeper "github.com/skip-mev/block-sdk/v2/x/lane/keeper"
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
	laneKeeper *blocklanekeeper.Keeper,
) block.Lane {
	lane := &blockbase.BaseLane{}
	_lane, err := blockbase.NewBaseLane(
		cfg,
		SystemLaneName,
		laneKeeper,
		blockbase.WithMatchHandler(matchFn),
	)
	if err != nil {
		panic(err)
	}

	*lane = *_lane
	return lane
}
