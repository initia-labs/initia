package app

import (
	"golang.org/x/exp/maps"

	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"

	packetforward "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/types"
	ratelimit "github.com/cosmos/ibc-apps/modules/rate-limiting/v8"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v8/types"
	"github.com/cosmos/ibc-go/modules/capability"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v8/modules/apps/29-fee"
	ibcfeetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"

	ibcnfttransfer "github.com/initia-labs/initia/x/ibc/nft-transfer"
	ibcnfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	ibcperm "github.com/initia-labs/initia/x/ibc/perm"
	ibcpermtypes "github.com/initia-labs/initia/x/ibc/perm/types"
	icaauth "github.com/initia-labs/initia/x/intertx"
	icaauthtypes "github.com/initia-labs/initia/x/intertx/types"

	// this line is used by starport scaffolding # stargate/app/moduleImport

	authzmodule "github.com/initia-labs/initia/x/authz/module"
	"github.com/initia-labs/initia/x/bank"
	distr "github.com/initia-labs/initia/x/distribution"
	"github.com/initia-labs/initia/x/evidence"
	"github.com/initia-labs/initia/x/genutil"
	"github.com/initia-labs/initia/x/gov"
	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	ibchookstypes "github.com/initia-labs/initia/x/ibc-hooks/types"
	"github.com/initia-labs/initia/x/move"
	movetypes "github.com/initia-labs/initia/x/move/types"
	staking "github.com/initia-labs/initia/x/mstaking"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	reward "github.com/initia-labs/initia/x/reward"
	rewardtypes "github.com/initia-labs/initia/x/reward/types"
	"github.com/initia-labs/initia/x/slashing"

	// connect oracle dependencies

	marketmap "github.com/skip-mev/connect/v2/x/marketmap"
	marketmaptypes "github.com/skip-mev/connect/v2/x/marketmap/types"
	"github.com/skip-mev/connect/v2/x/oracle"
	oracletypes "github.com/skip-mev/connect/v2/x/oracle/types"

	"github.com/initia-labs/OPinit/x/ophost"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	// noble forwarding keeper
	forwarding "github.com/noble-assets/forwarding/v2"
	forwardingtypes "github.com/noble-assets/forwarding/v2/types"

	dynamicfee "github.com/initia-labs/initia/x/dynamic-fee"
	dynamicfeetypes "github.com/initia-labs/initia/x/dynamic-fee/types"
)

var maccPerms = map[string][]string{
	authtypes.FeeCollectorName:      nil,
	distrtypes.ModuleName:           nil,
	icatypes.ModuleName:             nil,
	ibcfeetypes.ModuleName:          nil,
	rewardtypes.ModuleName:          nil,
	stakingtypes.BondedPoolName:     {authtypes.Burner, authtypes.Staking},
	stakingtypes.NotBondedPoolName:  {authtypes.Burner, authtypes.Staking},
	govtypes.ModuleName:             {authtypes.Burner},
	ibctransfertypes.ModuleName:     {authtypes.Minter, authtypes.Burner},
	movetypes.MoveStakingModuleName: nil,
	// connect oracle permissions
	oracletypes.ModuleName:    nil,
	marketmaptypes.ModuleName: nil,

	// this is only for testing
	authtypes.Minter: {authtypes.Minter},
}

func appModules(
	app *InitiaApp,
	skipGenesisInvariants bool,
) []module.AppModule {
	return []module.AppModule{
		genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app, app.txConfig),
		auth.NewAppModule(app.appCodec, *app.AccountKeeper, nil, nil),
		bank.NewAppModule(app.appCodec, *app.BankKeeper, app.AccountKeeper),
		capability.NewAppModule(app.appCodec, *app.CapabilityKeeper, false),
		crisis.NewAppModule(app.CrisisKeeper, skipGenesisInvariants, nil),
		feegrantmodule.NewAppModule(app.appCodec, app.AccountKeeper, app.BankKeeper, *app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(app.appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper),
		reward.NewAppModule(app.appCodec, *app.RewardKeeper),
		slashing.NewAppModule(app.appCodec, *app.SlashingKeeper),
		distr.NewAppModule(app.appCodec, *app.DistrKeeper),
		staking.NewAppModule(app.appCodec, *app.StakingKeeper),
		upgrade.NewAppModule(app.UpgradeKeeper, app.ac),
		evidence.NewAppModule(*app.EvidenceKeeper),
		authzmodule.NewAppModule(app.appCodec, *app.AuthzKeeper, *app.AccountKeeper, *app.BankKeeper, app.interfaceRegistry),
		groupmodule.NewAppModule(app.appCodec, *app.GroupKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		consensus.NewAppModule(app.appCodec, *app.ConsensusParamsKeeper),
		move.NewAppModule(app.appCodec, *app.MoveKeeper, app.vc, maps.Keys(maccPerms)),
		ophost.NewAppModule(app.appCodec, *app.OPHostKeeper),
		// connect modules
		oracle.NewAppModule(app.appCodec, *app.OracleKeeper),
		marketmap.NewAppModule(app.appCodec, app.MarketMapKeeper),
		// ibc modules
		ibc.NewAppModule(app.IBCKeeper),
		ibctransfer.NewAppModule(*app.TransferKeeper),
		ibcnfttransfer.NewAppModule(app.appCodec, *app.NftTransferKeeper),
		ica.NewAppModule(app.ICAControllerKeeper, app.ICAHostKeeper),
		icaauth.NewAppModule(app.appCodec, *app.ICAAuthKeeper),
		ibcfee.NewAppModule(*app.IBCFeeKeeper),
		ibcperm.NewAppModule(app.appCodec, *app.IBCPermKeeper),
		ibctm.NewAppModule(),
		solomachine.NewAppModule(),
		packetforward.NewAppModule(app.PacketForwardKeeper, nil),
		ibchooks.NewAppModule(app.appCodec, *app.IBCHooksKeeper),
		forwarding.NewAppModule(app.ForwardingKeeper),
		ratelimit.NewAppModule(app.appCodec, *app.RatelimitKeeper),
		dynamicfee.NewAppModule(app.appCodec, *app.DynamicFeeKeeper),
	}
}

// ModuleBasics defines the module BasicManager that is in charge of setting up basic,
// non-dependant module elements, such as codec registration
// and genesis verification.
func newBasicManagerFromManager(app *InitiaApp) module.BasicManager {
	basicManager := module.NewBasicManagerFromManager(
		app.ModuleManager,
		map[string]module.AppModuleBasic{
			genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutil.DefaultMessageValidator),
			govtypes.ModuleName:     gov.NewAppModuleBasic(app.appCodec),
		})
	basicManager.RegisterLegacyAminoCodec(app.legacyAmino)
	basicManager.RegisterInterfaces(app.interfaceRegistry)
	return basicManager
}

/*
orderBeginBlockers tells the app's module manager how to set the order of
BeginBlockers, which are run at the beginning of every block.

Interchain Security Requirements:
During begin block slashing happens after distr.BeginBlocker so that
there is nothing left over in the validator fee pool, so as to keep the
CanWithdrawInvariant invariant.
NOTE: staking module is required if HistoricalEntries param > 0
NOTE: capability module's beginblocker must come before any modules using capabilities (e.g. IBC)
*/
func orderBeginBlockers() []string {
	return []string{
		capabilitytypes.ModuleName,
		rewardtypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		authz.ModuleName,
		movetypes.ModuleName,
		ibcexported.ModuleName,
		oracletypes.ModuleName,
		marketmaptypes.ModuleName,
		ratelimittypes.ModuleName,
	}
}

/*
Interchain Security Requirements:
- provider.EndBlock gets validator updates from the staking module;
thus, staking.EndBlock must be executed before provider.EndBlock;
- creating a new consumer chain requires the following order,
CreateChildClient(), staking.EndBlock, provider.EndBlock;
thus, gov.EndBlock must be executed before staking.EndBlock
*/
func orderEndBlockers() []string {
	return []string{
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		group.ModuleName,
		oracletypes.ModuleName,
		marketmaptypes.ModuleName,
		forwardingtypes.ModuleName,
		ratelimittypes.ModuleName,
		dynamicfeetypes.ModuleName,
	}
}

/*
NOTE: The genutils module must occur after staking so that pools are
properly initialized with tokens from genesis accounts.
NOTE: The genutils module must also occur after auth so that it can access the params from auth.
NOTE: Capability module must occur first so that it can initialize any capabilities
so that other modules that want to create or claim capabilities afterwards in InitChain
can do so safely.
*/
func orderInitBlockers() []string {
	return []string{
		capabilitytypes.ModuleName, authtypes.ModuleName, movetypes.ModuleName, banktypes.ModuleName,
		distrtypes.ModuleName, stakingtypes.ModuleName, slashingtypes.ModuleName, govtypes.ModuleName,
		dynamicfeetypes.ModuleName, rewardtypes.ModuleName, crisistypes.ModuleName, genutiltypes.ModuleName, evidencetypes.ModuleName,
		authz.ModuleName, group.ModuleName, upgradetypes.ModuleName, feegrant.ModuleName,
		consensusparamtypes.ModuleName, ibcexported.ModuleName, ibctransfertypes.ModuleName,
		ibcnfttransfertypes.ModuleName, icatypes.ModuleName, icaauthtypes.ModuleName, ibcfeetypes.ModuleName,
		ibcpermtypes.ModuleName, ophosttypes.ModuleName, oracletypes.ModuleName, marketmaptypes.ModuleName,
		packetforwardtypes.ModuleName, ibchookstypes.ModuleName, forwardingtypes.ModuleName, ratelimittypes.ModuleName,
	}
}
