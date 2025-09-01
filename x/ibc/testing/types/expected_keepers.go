package types

import (
	"context"
	"time"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// StakingKeeper defines the expected staking keeper interface used in the
// IBC testing package
type StakingKeeper interface {
	GetHistoricalInfo(ctx context.Context, height int64) (stakingtypes.HistoricalInfo, error)
	DeleteHistoricalInfo(ctx context.Context, height int64) error
	TrackHistoricalInfo(ctx context.Context) error
	UnbondingTime(ctx context.Context) (time.Duration, error)
}
