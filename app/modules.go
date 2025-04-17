package app

import (
	"context"

	"golang.org/x/exp/maps"

	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
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

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
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

	// block-sdk dependencies

	"github.com/skip-mev/block-sdk/v2/x/auction"
	auctiontypes "github.com/skip-mev/block-sdk/v2/x/auction/types"

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

	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	bankkeeper "github.com/initia-labs/initia/x/bank/keeper"

	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"

	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"

	ratelimitkeeper "github.com/cosmos/ibc-apps/modules/rate-limiting/v8/keeper"
	ibcfeekeeper "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/keeper"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"

	ibcnfttransferkeeper "github.com/initia-labs/initia/x/ibc/nft-transfer/keeper"
	ibcpermkeeper "github.com/initia-labs/initia/x/ibc/perm/keeper"
	icaauthkeeper "github.com/initia-labs/initia/x/intertx/keeper"

	// this line is used by starport scaffolding # stargate/app/moduleImport

	distrkeeper "github.com/initia-labs/initia/x/distribution/keeper"
	dynamicfeekeeper "github.com/initia-labs/initia/x/dynamic-fee/keeper"
	evidencekeeper "github.com/initia-labs/initia/x/evidence/keeper"
	ibchookskeeper "github.com/initia-labs/initia/x/ibc-hooks/keeper"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"
	stakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"
	rewardkeeper "github.com/initia-labs/initia/x/reward/keeper"
	slashingkeeper "github.com/initia-labs/initia/x/slashing/keeper"

	// block-sdk dependencies

	auctionkeeper "github.com/skip-mev/block-sdk/v2/x/auction/keeper"

	// connect oracle dependencies

	oraclekeeper "github.com/skip-mev/connect/v2/x/oracle/keeper"

	ophostkeeper "github.com/initia-labs/OPinit/x/ophost/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/core/address"
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
	// x/auction's module account must be instantiated upon genesis to accrue auction rewards not
	// distributed to proposers
	auctiontypes.ModuleName: nil,
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
		authzmodule.NewAppModule(app.appCodec, *app.AuthzKeeper, app.interfaceRegistry),
		groupmodule.NewAppModule(app.appCodec, *app.GroupKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		consensus.NewAppModule(app.appCodec, *app.ConsensusParamsKeeper),
		move.NewAppModule(app.appCodec, *app.MoveKeeper, app.vc, maps.Keys(maccPerms)),
		auction.NewAppModule(app.appCodec, *app.AuctionKeeper),
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

// modulesForAutoCli returns a list of modules for auto-cli
func modulesForAutoCli(appCodec codec.Codec, txConfig client.TxConfig, interfaceRegistry cdctypes.InterfaceRegistry, ac, vc address.Codec) []module.AppModule {
	return []module.AppModule{
		genutil.NewAppModule(nil, nil, nil, txConfig),
		auth.NewAppModule(appCodec, accountkeeper.AccountKeeper{}, nil, nil),
		bank.NewAppModule(appCodec, bankkeeper.BaseKeeper{}, accountkeeper.AccountKeeper{}),
		capability.NewAppModule(appCodec, capabilitykeeper.Keeper{}, false),
		crisis.NewAppModule(nil, false, nil),
		feegrantmodule.NewAppModule(appCodec, mockAccountKeeper{addressCodec: ac}, nil, feegrantkeeper.Keeper{}, interfaceRegistry),
		gov.NewAppModule(appCodec, nil, nil, nil),
		reward.NewAppModule(appCodec, rewardkeeper.Keeper{}),
		slashing.NewAppModule(appCodec, slashingkeeper.Keeper{}),
		distr.NewAppModule(appCodec, distrkeeper.Keeper{}),
		staking.NewAppModule(appCodec, stakingkeeper.Keeper{}),
		upgrade.NewAppModule(nil, ac),
		evidence.NewAppModule(evidencekeeper.Keeper{}),
		authzmodule.NewAppModule(appCodec, authzkeeper.Keeper{}, interfaceRegistry),
		groupmodule.NewAppModule(appCodec, groupkeeper.Keeper{}, mockAccountKeeper{addressCodec: ac}, nil, interfaceRegistry),
		consensus.NewAppModule(appCodec, consensusparamkeeper.Keeper{}),
		move.NewAppModule(appCodec, movekeeper.Keeper{}, vc, maps.Keys(maccPerms)),
		auction.NewAppModule(appCodec, auctionkeeper.Keeper{}),
		ophost.NewAppModule(appCodec, ophostkeeper.Keeper{}),
		// connect modules
		oracle.NewAppModule(appCodec, oraclekeeper.Keeper{}),
		marketmap.NewAppModule(appCodec, nil),
		// ibc modules
		ibc.NewAppModule(nil),
		ibctransfer.NewAppModule(ibctransferkeeper.Keeper{}),
		ibcnfttransfer.NewAppModule(appCodec, ibcnfttransferkeeper.Keeper{}),
		ica.NewAppModule(nil, nil),
		icaauth.NewAppModule(appCodec, icaauthkeeper.Keeper{}),
		ibcfee.NewAppModule(ibcfeekeeper.Keeper{}),
		ibcperm.NewAppModule(appCodec, ibcpermkeeper.Keeper{}),
		ibctm.NewAppModule(),
		solomachine.NewAppModule(),
		packetforward.NewAppModule(nil, nil),
		ibchooks.NewAppModule(appCodec, ibchookskeeper.Keeper{}),
		forwarding.NewAppModule(nil),
		ratelimit.NewAppModule(appCodec, ratelimitkeeper.Keeper{}),
		dynamicfee.NewAppModule(appCodec, dynamicfeekeeper.Keeper{}),
	}
}

func NewBasicManager() module.BasicManager {
	return module.NewBasicManager(
		genutil.AppModuleBasic{},
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		crisis.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		gov.AppModuleBasic{},
		reward.AppModuleBasic{},
		slashing.AppModuleBasic{},
		distr.AppModuleBasic{},
		staking.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		groupmodule.AppModuleBasic{},
		consensus.AppModuleBasic{},
		move.AppModuleBasic{},
		auction.AppModuleBasic{},
		ophost.AppModuleBasic{},
		// connect modules
		oracle.AppModuleBasic{},
		marketmap.AppModuleBasic{},
		// ibc modules
		ibc.AppModuleBasic{},
		ibctransfer.AppModuleBasic{},
		ibcnfttransfer.AppModuleBasic{},
		ica.AppModuleBasic{},
		icaauth.AppModuleBasic{},
		ibcfee.AppModuleBasic{},
		ibcperm.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		solomachine.AppModuleBasic{},
		packetforward.AppModuleBasic{},
		ibchooks.AppModuleBasic{},
		forwarding.AppModuleBasic{},
		ratelimit.AppModuleBasic{},
		dynamicfee.AppModuleBasic{},
	)
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
		ibcpermtypes.ModuleName, auctiontypes.ModuleName, ophosttypes.ModuleName,
		oracletypes.ModuleName, marketmaptypes.ModuleName, packetforwardtypes.ModuleName, ibchookstypes.ModuleName,
		forwardingtypes.ModuleName, ratelimittypes.ModuleName,
	}
}

// mockAccountKeeper is a mock implementation of the account keeper interface
// it is used to pass the address codec to the modules for auto-cli
type mockAccountKeeper struct {
	addressCodec address.Codec
}

func (ak mockAccountKeeper) AddressCodec() address.Codec { return ak.addressCodec }
func (ak mockAccountKeeper) NewAccount(ctx context.Context, acc sdk.AccountI) sdk.AccountI {
	return nil
}
func (ak mockAccountKeeper) RemoveAccount(ctx context.Context, acc sdk.AccountI)             {}
func (ak mockAccountKeeper) IterateAccounts(ctx context.Context, cb func(sdk.AccountI) bool) {}
func (ak mockAccountKeeper) GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return nil
}
func (ak mockAccountKeeper) GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI {
	return nil
}
func (ak mockAccountKeeper) GetModuleAddress(moduleName string) sdk.AccAddress { return nil }
func (ak mockAccountKeeper) NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return nil
}
func (ak mockAccountKeeper) SetAccount(ctx context.Context, acc sdk.AccountI) {}
