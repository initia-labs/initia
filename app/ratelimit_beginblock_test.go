package app

import (
	"testing"
	"time"

	"cosmossdk.io/math"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v8/types"

	"github.com/stretchr/testify/require"
)

// rate-limiting v8.0.0 had the pre-0.50 BeginBlock signature, so the SDK never called it and no quota
// window ever reset on v1.4.x (mainnet hour epoch stuck at 4 since genesis).
func TestRateLimitWindowsResetInBeginBlock(t *testing.T) {
	app := SetupWithGenesisAccounts(nil, nil) // height 1 committed

	ctx := app.NewUncachedContext(false, cmtproto.Header{Height: 1, ChainID: app.ChainID()})
	start := time.Date(2026, 10, 23, 15, 0, 0, 0, time.UTC)
	app.RatelimitKeeper.SetHourEpoch(ctx, ratelimittypes.HourEpoch{
		EpochNumber:      4,
		Duration:         time.Hour,
		EpochStartTime:   start,
		EpochStartHeight: 1,
	})
	app.RatelimitKeeper.SetRateLimit(ctx, ratelimittypes.RateLimit{
		Path:  &ratelimittypes.Path{Denom: "uinit", ChannelId: "channel-3"},
		Quota: &ratelimittypes.Quota{MaxPercentSend: math.NewInt(100), MaxPercentRecv: math.ZeroInt(), DurationHours: 1},
		Flow:  &ratelimittypes.Flow{Inflow: math.NewInt(25), Outflow: math.NewInt(1_000), ChannelValue: math.NewInt(2_000)},
	})

	// a block inside the epoch: nothing moves
	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: start.Add(30 * time.Minute)})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)
	ctx = app.NewUncachedContext(false, cmtproto.Header{Height: 2})
	require.Equal(t, uint64(4), app.RatelimitKeeper.GetHourEpoch(ctx).EpochNumber)
	rl, found := app.RatelimitKeeper.GetRateLimit(ctx, "uinit", "channel-3")
	require.True(t, found)
	require.Equal(t, math.NewInt(1_000), rl.Flow.Outflow)

	// a block after the epoch end: epoch 5, flows reset
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 3, Time: start.Add(61 * time.Minute)})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)
	ctx = app.NewUncachedContext(false, cmtproto.Header{Height: 3})
	require.Equal(t, uint64(5), app.RatelimitKeeper.GetHourEpoch(ctx).EpochNumber)
	rl, found = app.RatelimitKeeper.GetRateLimit(ctx, "uinit", "channel-3")
	require.True(t, found)
	require.True(t, rl.Flow.Outflow.IsZero(), "outflow must reset: %s", rl.Flow.Outflow)
	require.True(t, rl.Flow.Inflow.IsZero(), "inflow must reset: %s", rl.Flow.Inflow)
}
