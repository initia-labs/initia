package app

import (
	"encoding/binary"
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/stretchr/testify/require"

	upgrades_v1_4_11 "github.com/initia-labs/initia/app/upgrades/v1_4_11"
)

// mainnet's last applied plan is v1.4.5 and this binary has no handler for it. x/upgrade refuses any
// block with that state unless our pre-block has already applied v1.4.11, so the first block at the
// restart height must pass and any earlier block must not run on this binary.
func seedLastPlan(t *testing.T, app *InitiaApp, name string, height int64) {
	t.Helper()
	key := make([]byte, 9+len(name))
	key[0] = upgradetypes.DoneByte
	binary.BigEndian.PutUint64(key[1:9], uint64(height))
	copy(key[9:], name)

	ctx := app.NewUncachedContext(false, cmtproto.Header{Height: 1, ChainID: app.ChainID()})
	ctx.KVStore(app.GetKey(upgradetypes.StoreKey)).Set(key, []byte{1})

	last, _, err := app.UpgradeKeeper.GetLastCompletedUpgrade(ctx)
	require.NoError(t, err)
	require.Equal(t, name, last)
	require.False(t, app.UpgradeKeeper.HasHandler(name))
}

func TestRestartUpgradePassesGuardAtRestartHeight(t *testing.T) {
	app := SetupWithGenesisAccounts(nil, nil) // height 1 committed

	oldChainID, oldHeight := upgrades_v1_4_11.RestartChainID, upgrades_v1_4_11.RestartHeight
	upgrades_v1_4_11.RestartChainID, upgrades_v1_4_11.RestartHeight = app.ChainID(), 2
	t.Cleanup(func() { upgrades_v1_4_11.RestartChainID, upgrades_v1_4_11.RestartHeight = oldChainID, oldHeight })

	seedLastPlan(t, app, "v1.4.5", 1)
	app.UpgradeKeeper.SetDowngradeVerified(false) // a fresh process

	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2})
	require.NoError(t, err)
}

func TestRestartBinaryRefusesBlocksBeforeRestartHeight(t *testing.T) {
	app := SetupWithGenesisAccounts(nil, nil)

	oldChainID, oldHeight := upgrades_v1_4_11.RestartChainID, upgrades_v1_4_11.RestartHeight
	upgrades_v1_4_11.RestartChainID, upgrades_v1_4_11.RestartHeight = app.ChainID(), 5
	t.Cleanup(func() { upgrades_v1_4_11.RestartChainID, upgrades_v1_4_11.RestartHeight = oldChainID, oldHeight })

	seedLastPlan(t, app, "v1.4.5", 1)
	app.UpgradeKeeper.SetDowngradeVerified(false)

	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2})
	require.ErrorContains(t, err, "upgrade handler is missing for v1.4.5")
}
