package app

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/stretchr/testify/require"

	vmtypes "github.com/initia-labs/movevm/types"

	upgrades_v1_4_11 "github.com/initia-labs/initia/app/upgrades/v1_4_11"
)

// The restart upgrade is applied by height, not by a stored plan. Check it runs exactly once at
// the restart height and clears forwarding allowed_denoms. Everything else is interwoven-1 only and
// must not run on any other chain.
func TestRestartUpgradeAppliesOnceAtRestartHeight(t *testing.T) {
	app := SetupWithGenesisAccounts(nil, nil) // height 1 committed

	oldChainID, oldHeight := upgrades_v1_4_11.RestartChainID, upgrades_v1_4_11.RestartHeight
	upgrades_v1_4_11.RestartChainID, upgrades_v1_4_11.RestartHeight = app.ChainID(), 3
	t.Cleanup(func() { upgrades_v1_4_11.RestartChainID, upgrades_v1_4_11.RestartHeight = oldChainID, oldHeight })

	// the halt-time state: "*" in forwarding allowed_denoms, no gate store
	ctx := app.NewUncachedContext(false, cmtproto.Header{Height: 1, ChainID: app.ChainID()})
	require.NoError(t, app.ForwardingKeeper.AllowedDenoms.Set(ctx, "*"))
	require.False(t, gateEnabled(t, app))

	// height 2 nothing happens
	finalize(t, app, 2)
	done, err := app.UpgradeKeeper.GetDoneHeight(app.NewUncachedContext(false, cmtproto.Header{Height: 2}), upgrades_v1_4_11.UpgradeName)
	require.NoError(t, err)
	require.Zero(t, done)
	require.False(t, gateEnabled(t, app))

	// height 3 the handler runs in the pre-block
	finalize(t, app, 3)
	ctx = app.NewUncachedContext(false, cmtproto.Header{Height: 3, ChainID: app.ChainID()})
	done, err = app.UpgradeKeeper.GetDoneHeight(ctx, upgrades_v1_4_11.UpgradeName)
	require.NoError(t, err)
	require.Equal(t, int64(3), done)
	require.False(t, gateEnabled(t, app), "the dex gate is interwoven-1 only")
	require.Empty(t, app.ForwardingKeeper.GetAllowedDenoms(ctx))
	govParams, err := app.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Empty(t, govParams.EmergencySubmitters, "gov params are interwoven-1 only")
	_, err = app.UpgradeKeeper.GetUpgradePlan(ctx)
	require.ErrorIs(t, err, upgradetypes.ErrNoUpgradePlanFound, "no plan may be left behind")

	// height 4 not applied again
	finalize(t, app, 4)
	done, err = app.UpgradeKeeper.GetDoneHeight(app.NewUncachedContext(false, cmtproto.Header{Height: 4}), upgrades_v1_4_11.UpgradeName)
	require.NoError(t, err)
	require.Equal(t, int64(3), done)
}

func finalize(t *testing.T, app *InitiaApp, height int64) {
	t.Helper()
	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)
}

func gateEnabled(t *testing.T, app *InitiaApp) bool {
	t.Helper()
	ctx := app.NewUncachedContext(false, cmtproto.Header{ChainID: app.ChainID()})
	out, _, err := app.MoveKeeper.ExecuteViewFunctionJSON(ctx, vmtypes.StdAddress, "dex", "is_gate_store_enabled", []vmtypes.TypeTag{}, []string{})
	if err != nil {
		// before the upgrade the on-chain dex has no such view
		return false
	}
	return out.Ret == "true"
}
