package app

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/testutil/mock"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/upgrade"

	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	ibctransfer "github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	ibc "github.com/cosmos/ibc-go/v7/modules/core"
	"github.com/strangelove-ventures/packet-forward-middleware/v7/router"

	"github.com/initia-labs/initia/x/bank"
	"github.com/initia-labs/initia/x/distribution"
	"github.com/initia-labs/initia/x/evidence"
	"github.com/initia-labs/initia/x/genutil"
	"github.com/initia-labs/initia/x/gov"

	nfttransfer "github.com/initia-labs/initia/x/ibc/nft-transfer"
	ibcperm "github.com/initia-labs/initia/x/ibc/perm"
	"github.com/initia-labs/initia/x/move"
	moveconfig "github.com/initia-labs/initia/x/move/config"
	staking "github.com/initia-labs/initia/x/mstaking"
	"github.com/initia-labs/initia/x/reward"
)

func TestSimAppExportAndBlockedAddrs(t *testing.T) {
	app := SetupWithGenesisAccounts(nil, nil)

	// BlockedAddresses returns a map of addresses in app v1 and a map of modules name in app v2.
	for acc := range app.ModuleAccountAddrs() {
		var addr sdk.AccAddress
		if modAddr, err := sdk.AccAddressFromBech32(acc); err == nil {
			addr = modAddr
		} else {
			addr = app.AccountKeeper.GetModuleAddress(acc)
		}

		require.True(
			t,
			app.BankKeeper.BlockedAddr(addr),
			fmt.Sprintf("ensure that blocked addresses are properly set in bank keeper: %s should be blocked", acc),
		)
	}
}

func TestGetMaccPerms(t *testing.T) {
	dup := GetMaccPerms()
	require.Equal(t, maccPerms, dup, "duplicated module account permissions differed from actual module account permissions")
}

func TestInitGenesisOnMigration(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	app := NewInitiaApp(
		logger, db, nil, true, moveconfig.DefaultMoveConfig(), simtestutil.EmptyAppOptions{})
	ctx := app.NewContext(true, tmproto.Header{Height: app.LastBlockHeight()})

	// Create a mock module. This module will serve as the new module we're
	// adding during a migration.
	mockCtrl := gomock.NewController(t)
	t.Cleanup(mockCtrl.Finish)
	mockModule := mock.NewMockAppModuleWithAllExtensions(mockCtrl)
	mockDefaultGenesis := json.RawMessage(`{"key": "value"}`)
	mockModule.EXPECT().DefaultGenesis(gomock.Eq(app.appCodec)).Times(1).Return(mockDefaultGenesis)
	mockModule.EXPECT().InitGenesis(gomock.Eq(ctx), gomock.Eq(app.appCodec), gomock.Eq(mockDefaultGenesis)).Times(1).Return(nil)
	mockModule.EXPECT().ConsensusVersion().Times(1).Return(uint64(0))

	app.mm.Modules["mock"] = mockModule

	// Run migrations only for "mock" module. We exclude it from
	// the VersionMap to simulate upgrading with a new module.
	_, err := app.mm.RunMigrations(ctx, app.configurator,
		module.VersionMap{
			"bank":                       bank.AppModule{}.ConsensusVersion(),
			"auth":                       auth.AppModule{}.ConsensusVersion(),
			"authz":                      authzmodule.AppModule{}.ConsensusVersion(),
			"mstaking":                   staking.AppModule{}.ConsensusVersion(),
			"reward":                     reward.AppModule{}.ConsensusVersion(),
			"distribution":               distribution.AppModule{}.ConsensusVersion(),
			"slashing":                   slashing.AppModule{}.ConsensusVersion(),
			"gov":                        gov.AppModule{}.ConsensusVersion(),
			"params":                     params.AppModule{}.ConsensusVersion(),
			"upgrade":                    upgrade.AppModule{}.ConsensusVersion(),
			"feegrant":                   feegrantmodule.AppModule{}.ConsensusVersion(),
			"evidence":                   evidence.AppModule{}.ConsensusVersion(),
			"crisis":                     crisis.AppModule{}.ConsensusVersion(),
			"genutil":                    genutil.AppModule{}.ConsensusVersion(),
			"capability":                 capability.AppModule{}.ConsensusVersion(),
			"group":                      groupmodule.AppModule{}.ConsensusVersion(),
			"consensus":                  consensus.AppModule{}.ConsensusVersion(),
			"ibc":                        ibc.AppModule{}.ConsensusVersion(),
			"transfer":                   ibctransfer.AppModule{}.ConsensusVersion(),
			"nonfungibletokentransfer":   nfttransfer.AppModule{}.ConsensusVersion(),
			"interchainaccounts":         ica.AppModule{}.ConsensusVersion(),
			"packetfowardmiddleware":     router.AppModule{}.ConsensusVersion(),
			"permissionedchannelrelayer": ibcperm.AppModule{}.ConsensusVersion(),
			"move":                       move.AppModule{}.ConsensusVersion(),
		},
	)
	require.NoError(t, err)
}

func TestUpgradeStateOnGenesis(t *testing.T) {
	app := SetupWithGenesisAccounts(nil, nil)

	// make sure the upgrade keeper has version map in state
	ctx := app.NewContext(false, tmproto.Header{})
	vm := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	for v, i := range app.mm.Modules {
		if i, ok := i.(module.HasConsensusVersion); ok {
			require.Equal(t, vm[v], i.ConsensusVersion())
		}
	}
}

func TestGetKey(t *testing.T) {
	db := dbm.NewMemDB()
	app := NewInitiaApp(
		log.NewTMLogger(log.NewSyncWriter(os.Stdout)),
		db, nil, true, moveconfig.DefaultMoveConfig(), simtestutil.EmptyAppOptions{})

	require.NotEmpty(t, app.GetKey(banktypes.StoreKey))
	require.NotEmpty(t, app.GetTKey(paramstypes.TStoreKey))
	require.NotEmpty(t, app.GetMemKey(capabilitytypes.MemStoreKey))
}
