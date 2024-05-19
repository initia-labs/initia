package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	abci "github.com/cometbft/cometbft/abci/types"
	tmjson "github.com/cometbft/cometbft/libs/json"
	tmos "github.com/cometbft/cometbft/libs/os"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	cosmosante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/posthandler"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/gogoproto/proto"

	packetforward "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward"
	packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/keeper"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/types"
	"github.com/cosmos/ibc-go/modules/capability"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	icacontroller "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v8/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"

	ibcnfttransfer "github.com/initia-labs/initia/x/ibc/nft-transfer"
	ibcnfttransferkeeper "github.com/initia-labs/initia/x/ibc/nft-transfer/keeper"
	ibcnfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	ibcperm "github.com/initia-labs/initia/x/ibc/perm"
	ibcpermkeeper "github.com/initia-labs/initia/x/ibc/perm/keeper"
	ibcpermtypes "github.com/initia-labs/initia/x/ibc/perm/types"
	ibctestingtypes "github.com/initia-labs/initia/x/ibc/testing/types"
	icaauth "github.com/initia-labs/initia/x/intertx"
	icaauthkeeper "github.com/initia-labs/initia/x/intertx/keeper"
	icaauthtypes "github.com/initia-labs/initia/x/intertx/types"

	// this line is used by starport scaffolding # stargate/app/moduleImport

	appante "github.com/initia-labs/initia/app/ante"
	apphook "github.com/initia-labs/initia/app/hook"
	applanes "github.com/initia-labs/initia/app/lanes"
	apporacle "github.com/initia-labs/initia/app/oracle"
	"github.com/initia-labs/initia/app/params"
	authzmodule "github.com/initia-labs/initia/x/authz/module"
	"github.com/initia-labs/initia/x/bank"
	bankkeeper "github.com/initia-labs/initia/x/bank/keeper"
	distr "github.com/initia-labs/initia/x/distribution"
	distrkeeper "github.com/initia-labs/initia/x/distribution/keeper"
	"github.com/initia-labs/initia/x/evidence"
	evidencekeeper "github.com/initia-labs/initia/x/evidence/keeper"
	"github.com/initia-labs/initia/x/genutil"
	"github.com/initia-labs/initia/x/gov"
	govkeeper "github.com/initia-labs/initia/x/gov/keeper"
	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	ibchookskeeper "github.com/initia-labs/initia/x/ibc-hooks/keeper"
	ibcmovehooks "github.com/initia-labs/initia/x/ibc-hooks/move-hooks"
	ibchookstypes "github.com/initia-labs/initia/x/ibc-hooks/types"
	"github.com/initia-labs/initia/x/move"
	moveconfig "github.com/initia-labs/initia/x/move/config"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"
	movetypes "github.com/initia-labs/initia/x/move/types"
	staking "github.com/initia-labs/initia/x/mstaking"
	stakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	reward "github.com/initia-labs/initia/x/reward"
	rewardkeeper "github.com/initia-labs/initia/x/reward/keeper"
	rewardtypes "github.com/initia-labs/initia/x/reward/types"
	"github.com/initia-labs/initia/x/slashing"
	slashingkeeper "github.com/initia-labs/initia/x/slashing/keeper"

	// block-sdk dependencies
	blockabci "github.com/skip-mev/block-sdk/v2/abci"
	blockchecktx "github.com/skip-mev/block-sdk/v2/abci/checktx"
	signer_extraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	"github.com/skip-mev/block-sdk/v2/block"
	blockbase "github.com/skip-mev/block-sdk/v2/block/base"
	mevlane "github.com/skip-mev/block-sdk/v2/lanes/mev"
	"github.com/skip-mev/block-sdk/v2/x/auction"
	auctionante "github.com/skip-mev/block-sdk/v2/x/auction/ante"
	auctionkeeper "github.com/skip-mev/block-sdk/v2/x/auction/keeper"
	auctiontypes "github.com/skip-mev/block-sdk/v2/x/auction/types"

	// slinky oracle dependencies
	oraclepreblock "github.com/skip-mev/slinky/abci/preblock/oracle"
	oracleproposals "github.com/skip-mev/slinky/abci/proposals"
	compression "github.com/skip-mev/slinky/abci/strategies/codec"
	"github.com/skip-mev/slinky/abci/strategies/currencypair"
	"github.com/skip-mev/slinky/abci/ve"
	oracleconfig "github.com/skip-mev/slinky/oracle/config"
	"github.com/skip-mev/slinky/pkg/math/voteweighted"
	oracleclient "github.com/skip-mev/slinky/service/clients/oracle"
	servicemetrics "github.com/skip-mev/slinky/service/metrics"
	marketmap "github.com/skip-mev/slinky/x/marketmap"
	marketmapkeeper "github.com/skip-mev/slinky/x/marketmap/keeper"
	marketmaptypes "github.com/skip-mev/slinky/x/marketmap/types"
	"github.com/skip-mev/slinky/x/oracle"
	oraclekeeper "github.com/skip-mev/slinky/x/oracle/keeper"
	oracletypes "github.com/skip-mev/slinky/x/oracle/types"

	"github.com/initia-labs/OPinit/x/ophost"
	ophostkeeper "github.com/initia-labs/OPinit/x/ophost/keeper"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	// noble forwarding keeper
	"github.com/noble-assets/forwarding/x/forwarding"
	forwardingkeeper "github.com/noble-assets/forwarding/x/forwarding/keeper"
	forwardingtypes "github.com/noble-assets/forwarding/x/forwarding/types"

	// unnamed import of statik for swagger UI support
	_ "github.com/initia-labs/initia/client/docs/statik"
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// module account permissions
	maccPerms = map[string][]string{
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
		// slinky oracle permissions
		oracletypes.ModuleName:    nil,
		marketmaptypes.ModuleName: nil,

		// this is only for testing
		authtypes.Minter: {authtypes.Minter},
	}
)

var (
	_ servertypes.Application = (*InitiaApp)(nil)
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, "."+AppName)
}

const (
	maxDefaultLaneSize = 2500
)

// InitiaApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type InitiaApp struct {
	*baseapp.BaseApp

	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry types.InterfaceRegistry

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper         *authkeeper.AccountKeeper
	BankKeeper            *bankkeeper.BaseKeeper
	CapabilityKeeper      *capabilitykeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        *slashingkeeper.Keeper
	RewardKeeper          *rewardkeeper.Keeper
	DistrKeeper           *distrkeeper.Keeper
	GovKeeper             *govkeeper.Keeper
	CrisisKeeper          *crisiskeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	GroupKeeper           *groupkeeper.Keeper
	ConsensusParamsKeeper *consensusparamkeeper.Keeper
	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	EvidenceKeeper        *evidencekeeper.Keeper
	TransferKeeper        *ibctransferkeeper.Keeper
	NftTransferKeeper     *ibcnfttransferkeeper.Keeper
	AuthzKeeper           *authzkeeper.Keeper
	FeeGrantKeeper        *feegrantkeeper.Keeper
	ICAHostKeeper         *icahostkeeper.Keeper
	ICAControllerKeeper   *icacontrollerkeeper.Keeper
	ICAAuthKeeper         *icaauthkeeper.Keeper
	IBCFeeKeeper          *ibcfeekeeper.Keeper
	IBCPermKeeper         *ibcpermkeeper.Keeper
	PacketForwardKeeper   *packetforwardkeeper.Keeper
	MoveKeeper            *movekeeper.Keeper
	IBCHooksKeeper        *ibchookskeeper.Keeper
	AuctionKeeper         *auctionkeeper.Keeper // x/auction keeper used to process bids for TOB auctions
	OPHostKeeper          *ophostkeeper.Keeper
	OracleKeeper          *oraclekeeper.Keeper // x/oracle keeper used for the slinky oracle
	MarketMapKeeper       *marketmapkeeper.Keeper
	ForwardingKeeper      *forwardingkeeper.Keeper

	// other slinky oracle services
	OracleClient          oracleclient.OracleClient
	oraclePreBlockHandler *oraclepreblock.PreBlockHandler

	// make scoped keepers public for test purposes
	ScopedIBCKeeper           capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper      capabilitykeeper.ScopedKeeper
	ScopedNftTransferKeeper   capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
	ScopedICAAuthKeeper       capabilitykeeper.ScopedKeeper

	// the module manager
	ModuleManager      *module.Manager
	BasicModuleManager module.BasicManager

	// the configurator
	configurator module.Configurator

	// Override of BaseApp's CheckTx
	checkTxHandler blockchecktx.CheckTx
}

// NewInitiaApp returns a reference to an initialized Initia.
func NewInitiaApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	moveConfig moveconfig.MoveConfig,
	oracleConfig oracleconfig.AppConfig,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *InitiaApp {
	encodingConfig := params.MakeEncodingConfig()
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	appCodec := encodingConfig.Codec
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry
	txConfig := encodingConfig.TxConfig

	bApp := baseapp.NewBaseApp(AppName, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetTxEncoder(txConfig.TxEncoder())

	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey, crisistypes.StoreKey,
		rewardtypes.StoreKey, distrtypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, group.StoreKey, consensusparamtypes.StoreKey,
		ibcexported.StoreKey, upgradetypes.StoreKey, evidencetypes.StoreKey,
		ibctransfertypes.StoreKey, ibcnfttransfertypes.StoreKey, capabilitytypes.StoreKey,
		authzkeeper.StoreKey, feegrant.StoreKey, icahosttypes.StoreKey,
		icacontrollertypes.StoreKey, ibcfeetypes.StoreKey, ibcpermtypes.StoreKey,
		movetypes.StoreKey, auctiontypes.StoreKey, ophosttypes.StoreKey,
		oracletypes.StoreKey, packetforwardtypes.StoreKey, ibchookstypes.StoreKey,
		forwardingtypes.StoreKey, marketmaptypes.StoreKey,
	)
	tkeys := storetypes.NewTransientStoreKeys(forwardingtypes.TransientStoreKey)
	memKeys := storetypes.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)

	// register streaming services
	if err := bApp.RegisterStreamingServices(appOpts, keys); err != nil {
		panic(err)
	}

	app := &InitiaApp{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		txConfig:          txConfig,
		interfaceRegistry: interfaceRegistry,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	ac := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	vc := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	cc := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix())

	authorityAccAddr := authtypes.NewModuleAddress(govtypes.ModuleName)
	authorityAddr, err := ac.BytesToString(authorityAccAddr)
	if err != nil {
		panic(err)
	}

	// set the BaseApp's parameter store
	consensusParamsKeeper := consensusparamkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]), authorityAddr, runtime.EventService{})
	app.ConsensusParamsKeeper = &consensusParamsKeeper
	bApp.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	// add capability keeper and ScopeToModule for ibc module
	app.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])

	// grant capabilities for the ibc and ibc-transfer modules
	app.ScopedIBCKeeper = app.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	app.ScopedTransferKeeper = app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	app.ScopedNftTransferKeeper = app.CapabilityKeeper.ScopeToModule(ibcnfttransfertypes.ModuleName)
	app.ScopedICAHostKeeper = app.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	app.ScopedICAControllerKeeper = app.CapabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)
	app.ScopedICAAuthKeeper = app.CapabilityKeeper.ScopeToModule(icaauthtypes.ModuleName)

	app.CapabilityKeeper.Seal()

	// add keepers
	app.MoveKeeper = &movekeeper.Keeper{}

	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		ac,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authorityAddr,
	)
	app.AccountKeeper = &accountKeeper

	bankKeeper := bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		app.AccountKeeper,
		movekeeper.NewMoveBankKeeper(app.MoveKeeper),
		app.ModuleAccountAddrs(),
		authorityAddr,
	)
	app.BankKeeper = &bankKeeper

	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		movekeeper.NewVotingPowerKeeper(app.MoveKeeper),
		authorityAddr,
		vc,
		cc,
	)

	app.RewardKeeper = rewardkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[rewardtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authorityAddr,
	)

	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		movekeeper.NewDexKeeper(app.MoveKeeper),
		authtypes.FeeCollectorName,
		authorityAddr,
	)

	slashingKeeper := slashingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[slashingtypes.StoreKey]),
		app.StakingKeeper,
		authorityAddr,
	)
	app.SlashingKeeper = &slashingKeeper

	invCheckPeriod := cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod))
	app.CrisisKeeper = crisiskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[crisistypes.StoreKey]),
		invCheckPeriod,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authorityAddr,
		ac,
	)

	// get skipUpgradeHeights from the app options
	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))
	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(keys[upgradetypes.StoreKey]),
		appCodec,
		homePath,
		app.BaseApp,
		authorityAddr,
	)

	i := 0
	moduleAddrs := make([]sdk.AccAddress, len(maccPerms))
	for name := range maccPerms {
		moduleAddrs[i] = authtypes.NewModuleAddress(name)
		i += 1
	}

	feeGrantKeeper := feegrantkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys[feegrant.StoreKey]), app.AccountKeeper)
	app.FeeGrantKeeper = &feeGrantKeeper

	authzKeeper := authzkeeper.NewKeeper(runtime.NewKVStoreService(keys[authzkeeper.StoreKey]), appCodec, app.BaseApp.MsgServiceRouter(), app.AccountKeeper)
	app.AuthzKeeper = &authzKeeper

	// Create evidence Keeper for to register the IBC light client misbehaviour evidence route
	app.EvidenceKeeper = evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[evidencetypes.StoreKey]),
		app.StakingKeeper,
		app.SlashingKeeper,
		ac,
		runtime.ProvideCometInfoService(),
	)

	groupConfig := group.DefaultConfig()
	groupKeeper := groupkeeper.NewKeeper(
		keys[group.StoreKey],
		appCodec,
		app.MsgServiceRouter(),
		app.AccountKeeper,
		groupConfig,
	)
	app.GroupKeeper = &groupKeeper

	// Create IBC Keeper
	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		keys[ibcexported.StoreKey],
		nil, // we don't need migration
		app.StakingKeeper,
		app.UpgradeKeeper,
		app.ScopedIBCKeeper,
		authorityAddr,
	)

	ibcFeeKeeper := ibcfeekeeper.NewKeeper(
		appCodec,
		keys[ibcfeetypes.StoreKey],
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
	)
	app.IBCFeeKeeper = &ibcFeeKeeper

	app.IBCPermKeeper = ibcpermkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ibcpermtypes.StoreKey]),
		authorityAddr,
		ac,
	)

	marketMapKeeper := marketmapkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[marketmaptypes.StoreKey]),
		appCodec,
		authorityAccAddr,
	)
	app.MarketMapKeeper = marketMapKeeper

	oracleKeeper := oraclekeeper.NewKeeper(
		runtime.NewKVStoreService(keys[oracletypes.StoreKey]),
		appCodec,
		app.MarketMapKeeper,
		authorityAccAddr,
	)
	app.OracleKeeper = &oracleKeeper

	// Add the oracle keeper as a hook to market map keeper so new market map entries can be created
	// and propogated to the oracle keeper.
	app.MarketMapKeeper.SetHooks(app.OracleKeeper.Hooks())

	app.IBCHooksKeeper = ibchookskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ibchookstypes.StoreKey]),
		authorityAddr,
		ac,
	)

	app.ForwardingKeeper = forwardingkeeper.NewKeeper(
		appCodec,
		app.Logger(),
		runtime.NewKVStoreService(keys[forwardingtypes.StoreKey]),
		runtime.NewTransientStoreService(tkeys[forwardingtypes.TransientStoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.TransferKeeper,
	)
	app.BankKeeper.AppendSendRestriction(app.ForwardingKeeper.SendRestrictionFn)

	////////////////////////////
	// Transfer configuration //
	////////////////////////////
	// Send   : transfer -> packet forward -> fee    -> channel
	// Receive: channel  -> perm           -> fee    -> move    -> packet forward  -> transfer

	var transferStack porttypes.IBCModule
	{
		packetForwardKeeper := &packetforwardkeeper.Keeper{}

		// Create Transfer Keepers
		transferKeeper := ibctransferkeeper.NewKeeper(
			appCodec,
			keys[ibctransfertypes.StoreKey],
			nil, // we don't need migration
			// ics4wrapper: transfer -> packet forward
			packetForwardKeeper,
			app.IBCKeeper.ChannelKeeper,
			app.IBCKeeper.PortKeeper,
			app.AccountKeeper,
			app.BankKeeper,
			app.ScopedTransferKeeper,
			authorityAddr,
		)
		app.TransferKeeper = &transferKeeper
		transferStack = ibctransfer.NewIBCModule(*app.TransferKeeper)

		// forwarding middleware
		transferStack = forwarding.NewMiddleware(
			// receive: forwarding -> transfer
			transferStack,
			app.AccountKeeper,
			app.ForwardingKeeper,
		)

		// create packet forward middleware
		*packetForwardKeeper = *packetforwardkeeper.NewKeeper(
			appCodec,
			keys[packetforwardtypes.StoreKey],
			app.TransferKeeper,
			app.IBCKeeper.ChannelKeeper,
			app.DistrKeeper,
			app.BankKeeper,
			// ics4wrapper: transfer -> packet forward -> fee
			app.IBCFeeKeeper,
			authorityAddr,
		)
		app.PacketForwardKeeper = packetForwardKeeper
		transferStack = packetforward.NewIBCMiddleware(
			// receive: packet forward -> forwarding -> transfer
			transferStack,
			app.PacketForwardKeeper,
			0,
			packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp,
			packetforwardkeeper.DefaultRefundTransferPacketTimeoutTimestamp,
		)

		// create move middleware for transfer
		transferStack = ibchooks.NewIBCMiddleware(
			// receive: move -> packet forward -> forwarding -> transfer
			transferStack,
			ibchooks.NewICS4Middleware(
				nil, /* ics4wrapper: not used */
				ibcmovehooks.NewMoveHooks(appCodec, ac, app.MoveKeeper),
			),
			app.IBCHooksKeeper,
		)

		// create ibcfee middleware for transfer
		transferStack = ibcfee.NewIBCMiddleware(
			// receive: fee -> move -> packet forward -> forwarding -> transfer
			transferStack,
			*app.IBCFeeKeeper,
		)

		// create perm middleware for transfer
		transferStack = ibcperm.NewIBCMiddleware(
			// receive: perm -> fee -> move -> packet forward -> forwarding -> transfer
			transferStack,
			// ics4wrapper: not used
			nil,
			*app.IBCPermKeeper,
		)
	}

	////////////////////////////////
	// Nft Transfer configuration //
	////////////////////////////////

	var nftTransferStack porttypes.IBCModule
	{
		// Create Transfer Keepers
		app.NftTransferKeeper = ibcnfttransferkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(keys[ibcnfttransfertypes.StoreKey]),
			// ics4wrapper: nft transfer -> fee -> channel
			app.IBCFeeKeeper,
			app.IBCKeeper.ChannelKeeper,
			app.IBCKeeper.PortKeeper,
			app.AccountKeeper,
			movekeeper.NewNftKeeper(app.MoveKeeper),
			app.ScopedNftTransferKeeper,
			authorityAddr,
		)
		nftTransferIBCModule := ibcnfttransfer.NewIBCModule(*app.NftTransferKeeper)

		// create move middleware for nft-transfer
		hookMiddleware := ibchooks.NewIBCMiddleware(
			// receive: move -> nft-transfer
			nftTransferIBCModule,
			ibchooks.NewICS4Middleware(
				nil, /* ics4wrapper: not used */
				ibcmovehooks.NewMoveHooks(appCodec, ac, app.MoveKeeper),
			),
			app.IBCHooksKeeper,
		)

		nftTransferStack = ibcperm.NewIBCMiddleware(
			// receive: perm -> fee -> nft transfer
			ibcfee.NewIBCMiddleware(
				// receive: channel -> fee -> move -> nft transfer
				hookMiddleware,
				*app.IBCFeeKeeper,
			),
			// ics4wrapper: not used
			nil,
			*app.IBCPermKeeper,
		)
	}

	///////////////////////
	// ICA configuration //
	///////////////////////

	var icaHostStack porttypes.IBCModule
	var icaControllerStack porttypes.IBCModule
	{
		icaHostKeeper := icahostkeeper.NewKeeper(
			appCodec, keys[icahosttypes.StoreKey],
			nil, // we don't need migration
			app.IBCFeeKeeper,
			app.IBCKeeper.ChannelKeeper,
			app.IBCKeeper.PortKeeper,
			app.AccountKeeper,
			app.ScopedICAHostKeeper,
			app.MsgServiceRouter(),
			authorityAddr,
		)
		app.ICAHostKeeper = &icaHostKeeper

		icaControllerKeeper := icacontrollerkeeper.NewKeeper(
			appCodec, keys[icacontrollertypes.StoreKey],
			nil, // we don't need migration
			app.IBCFeeKeeper,
			app.IBCKeeper.ChannelKeeper,
			app.IBCKeeper.PortKeeper,
			app.ScopedICAControllerKeeper,
			app.MsgServiceRouter(),
			authorityAddr,
		)
		app.ICAControllerKeeper = &icaControllerKeeper

		icaAuthKeeper := icaauthkeeper.NewKeeper(
			appCodec,
			*app.ICAControllerKeeper,
			app.ScopedICAAuthKeeper,
			ac,
		)
		app.ICAAuthKeeper = &icaAuthKeeper

		icaAuthIBCModule := icaauth.NewIBCModule(*app.ICAAuthKeeper)
		icaHostIBCModule := icahost.NewIBCModule(*app.ICAHostKeeper)
		icaHostStack = ibcperm.NewIBCMiddleware(
			// receive: perm -> fee -> ica host
			ibcfee.NewIBCMiddleware(icaHostIBCModule, *app.IBCFeeKeeper),
			// ics4wrapper: not used
			nil,
			*app.IBCPermKeeper,
		)
		icaControllerIBCModule := icacontroller.NewIBCMiddleware(icaAuthIBCModule, *app.ICAControllerKeeper)
		icaControllerStack = ibcperm.NewIBCMiddleware(
			// receive: perm -> fee -> ica controller
			ibcfee.NewIBCMiddleware(icaControllerIBCModule, *app.IBCFeeKeeper),
			// ics4wrapper: not used
			nil,
			*app.IBCPermKeeper,
		)
	}

	//////////////////////////////
	// IBC router Configuration //
	//////////////////////////////

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := porttypes.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack).
		AddRoute(icahosttypes.SubModuleName, icaHostStack).
		AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
		AddRoute(icaauthtypes.ModuleName, icaControllerStack).
		AddRoute(ibcnfttransfertypes.ModuleName, nftTransferStack)
	app.IBCKeeper.SetRouter(ibcRouter)

	//////////////////////////////
	// MoveKeeper Configuration //
	//////////////////////////////

	*app.MoveKeeper = *movekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[movetypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.OracleKeeper,
		// app.NftTransferKeeper,
		app.BaseApp.MsgServiceRouter(),
		app.BaseApp.GRPCQueryRouter(),
		moveConfig,
		// staking feature
		app.DistrKeeper,
		app.StakingKeeper,
		app.RewardKeeper,
		app.DistrKeeper,
		authtypes.FeeCollectorName,
		authorityAddr,
		ac, vc,
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			app.DistrKeeper.Hooks(),
			app.SlashingKeeper.Hooks(),
		),
	)
	app.StakingKeeper.SetSlashingHooks(app.MoveKeeper.Hooks())

	// x/auction module keeper initialization

	// initialize the keeper
	auctionKeeper := auctionkeeper.NewKeeperWithRewardsAddressProvider(
		app.appCodec,
		app.keys[auctiontypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		applanes.NewRewardsAddressProvider(*app.StakingKeeper, *app.DistrKeeper),
		authorityAddr,
	)
	app.AuctionKeeper = &auctionKeeper

	app.OPHostKeeper = ophostkeeper.NewKeeper(
		app.appCodec,
		runtime.NewKVStoreService(app.keys[ophosttypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.DistrKeeper,
		ophosttypes.NewBridgeHooks(apphook.NewBridgeHook(app.IBCKeeper.ChannelKeeper, app.IBCPermKeeper, ac)),
		authorityAddr,
	)

	govConfig := govtypes.DefaultConfig()
	app.GovKeeper = govkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(keys[govtypes.StoreKey]), app.AccountKeeper, app.BankKeeper,
		app.StakingKeeper, app.DistrKeeper, app.MsgServiceRouter(), govConfig, authorityAddr,
	)

	/****  Module Options ****/

	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	skipGenesisInvariants := cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.

	app.ModuleManager = module.NewManager(
		genutil.NewAppModule(
			app.AccountKeeper, app.StakingKeeper, app, encodingConfig.TxConfig),
		auth.NewAppModule(appCodec, *app.AccountKeeper, nil, nil),
		bank.NewAppModule(appCodec, *app.BankKeeper, app.AccountKeeper),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
		crisis.NewAppModule(app.CrisisKeeper, skipGenesisInvariants, nil),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, *app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper),
		reward.NewAppModule(appCodec, *app.RewardKeeper),
		slashing.NewAppModule(appCodec, *app.SlashingKeeper),
		distr.NewAppModule(appCodec, *app.DistrKeeper),
		staking.NewAppModule(appCodec, *app.StakingKeeper),
		upgrade.NewAppModule(app.UpgradeKeeper, ac),
		evidence.NewAppModule(*app.EvidenceKeeper),
		authzmodule.NewAppModule(appCodec, *app.AuthzKeeper, app.interfaceRegistry),
		groupmodule.NewAppModule(appCodec, *app.GroupKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		consensus.NewAppModule(appCodec, *app.ConsensusParamsKeeper),
		move.NewAppModule(appCodec, *app.MoveKeeper, vc),
		auction.NewAppModule(app.appCodec, *app.AuctionKeeper),
		ophost.NewAppModule(appCodec, *app.OPHostKeeper),
		// slinky modules
		oracle.NewAppModule(appCodec, *app.OracleKeeper),
		marketmap.NewAppModule(appCodec, app.MarketMapKeeper),
		// ibc modules
		ibc.NewAppModule(app.IBCKeeper),
		ibctransfer.NewAppModule(*app.TransferKeeper),
		ibcnfttransfer.NewAppModule(appCodec, *app.NftTransferKeeper),
		ica.NewAppModule(app.ICAControllerKeeper, app.ICAHostKeeper),
		icaauth.NewAppModule(appCodec, *app.ICAAuthKeeper),
		ibcfee.NewAppModule(*app.IBCFeeKeeper),
		ibcperm.NewAppModule(*app.IBCPermKeeper),
		ibctm.NewAppModule(),
		solomachine.NewAppModule(),
		packetforward.NewAppModule(app.PacketForwardKeeper, nil),
		ibchooks.NewAppModule(appCodec, *app.IBCHooksKeeper),
		forwarding.NewAppModule(app.ForwardingKeeper),
	)

	// BasicModuleManager defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration and genesis verification.
	// By default it is composed of all the module from the module manager.
	// Additionally, app module basics can be overwritten by passing them as argument.
	app.BasicModuleManager = module.NewBasicManagerFromManager(
		app.ModuleManager,
		map[string]module.AppModuleBasic{
			genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutil.DefaultMessageValidator),
			govtypes.ModuleName:     gov.NewAppModuleBasic(appCodec),
		})
	app.BasicModuleManager.RegisterLegacyAminoCodec(legacyAmino)
	app.BasicModuleManager.RegisterInterfaces(interfaceRegistry)

	// NOTE: upgrade module is required to be prioritized
	app.ModuleManager.SetOrderPreBlockers(
		upgradetypes.ModuleName,
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	app.ModuleManager.SetOrderBeginBlockers(
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
	)

	app.ModuleManager.SetOrderEndBlockers(
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
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	genesisModuleOrder := []string{
		capabilitytypes.ModuleName, authtypes.ModuleName, movetypes.ModuleName, banktypes.ModuleName,
		distrtypes.ModuleName, stakingtypes.ModuleName, slashingtypes.ModuleName, govtypes.ModuleName,
		rewardtypes.ModuleName, crisistypes.ModuleName, genutiltypes.ModuleName, evidencetypes.ModuleName,
		authz.ModuleName, group.ModuleName, upgradetypes.ModuleName, feegrant.ModuleName,
		consensusparamtypes.ModuleName, ibcexported.ModuleName, ibctransfertypes.ModuleName,
		ibcnfttransfertypes.ModuleName, icatypes.ModuleName, icaauthtypes.ModuleName, ibcfeetypes.ModuleName,
		ibcpermtypes.ModuleName, consensusparamtypes.ModuleName, auctiontypes.ModuleName, ophosttypes.ModuleName,
		oracletypes.ModuleName, marketmaptypes.ModuleName, packetforwardtypes.ModuleName, ibchookstypes.ModuleName,
		forwardingtypes.ModuleName,
	}
	app.ModuleManager.SetOrderInitGenesis(genesisModuleOrder...)
	app.ModuleManager.SetOrderExportGenesis(genesisModuleOrder...)

	app.ModuleManager.RegisterInvariants(app.CrisisKeeper)

	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	err = app.ModuleManager.RegisterServices(app.configurator)
	if err != nil {
		panic(err)
	}

	// register upgrade handler for later use
	app.RegisterUpgradeHandlers(app.configurator)

	autocliv1.RegisterQueryServer(app.GRPCQueryRouter(), runtimeservices.NewAutoCLIQueryService(app.ModuleManager.Modules))

	reflectionSvc, err := runtimeservices.NewReflectionService()
	if err != nil {
		panic(err)
	}
	reflectionv1.RegisterReflectionServiceServer(app.GRPCQueryRouter(), reflectionSvc)

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetPreBlocker(app.PreBlocker)
	app.SetBeginBlocker(app.BeginBlocker)
	app.setPostHandler()
	app.SetEndBlocker(app.EndBlocker)

	//////////////////
	/// lane start ///
	//////////////////

	// initialize and set the InitiaApp mempool. The current mempool will be the
	// x/auction module's mempool which will extract the top bid from the current block's auction
	// and insert the txs at the top of the block spots.
	signerExtractor := signer_extraction.NewDefaultAdapter()

	systemLane := applanes.NewSystemLane(blockbase.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.05"),
		MaxTxs:          1,
		SignerExtractor: signerExtractor,
	}, applanes.RejectMatchHandler())

	factory := mevlane.NewDefaultAuctionFactory(app.txConfig.TxDecoder(), signerExtractor)
	mevLane := mevlane.NewMEVLane(blockbase.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.15"),
		MaxTxs:          100,
		SignerExtractor: signerExtractor,
	}, factory, factory.MatchHandler())

	freeLane := applanes.NewFreeLane(blockbase.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.2"),
		MaxTxs:          100,
		SignerExtractor: signerExtractor,
	}, applanes.FreeLaneMatchHandler())

	defaultLane := applanes.NewDefaultLane(blockbase.LaneConfig{
		Logger:          app.Logger(),
		TxEncoder:       app.txConfig.TxEncoder(),
		TxDecoder:       app.txConfig.TxDecoder(),
		MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.6"),
		MaxTxs:          maxDefaultLaneSize,
		SignerExtractor: signerExtractor,
	})

	lanes := []block.Lane{systemLane, mevLane, freeLane, defaultLane}
	mempool, err := block.NewLanedMempool(app.Logger(), lanes)
	if err != nil {
		panic(err)
	}

	// The application's mempool is now powered by the Block SDK!
	app.SetMempool(mempool)
	anteHandler := app.setAnteHandler(mevLane, freeLane)

	// set the ante handler for each lane for VerifyTx at PrepareLaneHandler
	//
	opt := []blockbase.LaneOption{
		blockbase.WithAnteHandler(anteHandler),
	}
	systemLane.(*blockbase.BaseLane).WithOptions(
		opt...,
	)
	mevLane.WithOptions(
		opt...,
	)
	freeLane.(*blockbase.BaseLane).WithOptions(
		opt...,
	)
	defaultLane.(*blockbase.BaseLane).WithOptions(
		opt...,
	)

	// override the base-app's ABCI methods (CheckTx, PrepareProposal, ProcessProposal)
	blockProposalHandlers := blockabci.NewProposalHandler(
		app.Logger(),
		app.txConfig.TxDecoder(),
		app.txConfig.TxEncoder(),
		mempool,
	)

	// overrde base-app's CheckTx
	mevCheckTx := blockchecktx.NewMEVCheckTxHandler(
		app.BaseApp,
		app.txConfig.TxDecoder(),
		mevLane,
		anteHandler,
		app.BaseApp.CheckTx,
	)
	checkTxHandler := blockchecktx.NewMempoolParityCheckTx(
		app.Logger(), mempool,
		app.txConfig.TxDecoder(), mevCheckTx.CheckTx(),
	)
	app.SetCheckTx(checkTxHandler.CheckTx())

	////////////////
	/// lane end ///
	////////////////

	////////////////////
	/// oracle start ///
	////////////////////

	if err := oracleConfig.ValidateBasic(); err != nil {
		panic(err)
	}

	serviceMetrics, err := servicemetrics.NewMetricsFromConfig(
		oracleConfig,
		app.ChainID(),
	)
	if err != nil {
		panic(err)
	}

	app.OracleClient, err = apporacle.NewClientFromConfig(
		oracleConfig,
		app.Logger(),
		serviceMetrics,
	)
	if err != nil {
		panic(err)
	}

	oracleProposalHandler := oracleproposals.NewProposalHandler(
		app.Logger(),
		blockProposalHandlers.PrepareProposalHandler(),
		blockProposalHandlers.ProcessProposalHandler(),
		ve.NewDefaultValidateVoteExtensionsFn(
			stakingkeeper.NewCompatibilityKeeper(app.StakingKeeper),
		),
		compression.NewCompressionVoteExtensionCodec(
			compression.NewDefaultVoteExtensionCodec(),
			compression.NewZLibCompressor(),
		),
		compression.NewCompressionExtendedCommitCodec(
			compression.NewDefaultExtendedCommitCodec(),
			compression.NewZStdCompressor(),
		),
		currencypair.NewHashCurrencyPairStrategy(app.OracleKeeper),
		serviceMetrics,
	)

	// override baseapp's ProcessProposal + PrepareProposal
	app.SetPrepareProposal(oracleProposalHandler.PrepareProposalHandler())
	app.SetProcessProposal(oracleProposalHandler.ProcessProposalHandler())

	app.oraclePreBlockHandler = oraclepreblock.NewOraclePreBlockHandler(
		app.Logger(),
		voteweighted.MedianFromContext(
			app.Logger(),
			stakingkeeper.NewCompatibilityKeeper(app.StakingKeeper),
			voteweighted.DefaultPowerThreshold),
		app.OracleKeeper,
		serviceMetrics,
		currencypair.NewHashCurrencyPairStrategy(app.OracleKeeper),
		compression.NewCompressionVoteExtensionCodec(
			compression.NewDefaultVoteExtensionCodec(),
			compression.NewZLibCompressor(),
		),
		compression.NewCompressionExtendedCommitCodec(
			compression.NewDefaultExtendedCommitCodec(),
			compression.NewZStdCompressor(),
		),
	)

	// Create the vote extensions handler that will be used to extend and verify
	// vote extensions (i.e. oracle data).
	voteExtensionsHandler := ve.NewVoteExtensionHandler(
		app.Logger(),
		app.OracleClient,
		time.Second,
		currencypair.NewHashCurrencyPairStrategy(app.OracleKeeper),
		compression.NewCompressionVoteExtensionCodec(
			compression.NewDefaultVoteExtensionCodec(),
			compression.NewZLibCompressor(),
		),
		app.oraclePreBlockHandler.PreBlocker(),
		serviceMetrics,
	)
	app.SetExtendVoteHandler(voteExtensionsHandler.ExtendVoteHandler())
	app.SetVerifyVoteExtensionHandler(voteExtensionsHandler.VerifyVoteExtensionHandler())

	//////////////////
	/// oracle end ///
	//////////////////

	// At startup, after all modules have been registered, check that all prot
	// annotations are correct.
	protoFiles, err := proto.MergedRegistry()
	if err != nil {
		panic(err)
	}
	err = msgservice.ValidateProtoAnnotations(protoFiles)
	if err != nil {
		// Once we switch to using protoreflect-based antehandlers, we might
		// want to panic here instead of logging a warning.
		fmt.Fprintln(os.Stderr, err.Error())
	}

	// Load the latest state from disk if necessary, and initialize the base-app. From this point on
	// no more modifications to the base-app can be made
	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}

	return app
}

// CheckTx will check the transaction with the provided checkTxHandler. We override the default
// handler so that we can verify bid transactions before they are inserted into the mempool.
// With the POB CheckTx, we can verify the bid transaction and all of the bundled transactions
// before inserting the bid transaction into the mempool.
func (app *InitiaApp) CheckTx(req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	return app.checkTxHandler(req)
}

// SetCheckTx sets the checkTxHandler for the app.
func (app *InitiaApp) SetCheckTx(handler blockchecktx.CheckTx) {
	app.checkTxHandler = handler
}

func (app *InitiaApp) setAnteHandler(
	mevLane auctionante.MEVLane,
	freeLane block.Lane,
) sdk.AnteHandler {
	anteHandler, err := appante.NewAnteHandler(
		appante.HandlerOptions{
			HandlerOptions: cosmosante.HandlerOptions{
				AccountKeeper:   app.AccountKeeper,
				BankKeeper:      app.BankKeeper,
				FeegrantKeeper:  app.FeeGrantKeeper,
				SignModeHandler: app.txConfig.SignModeHandler(),
			},
			IBCkeeper:     app.IBCKeeper,
			MoveKeeper:    movekeeper.NewDexKeeper(app.MoveKeeper),
			Codec:         app.appCodec,
			TxEncoder:     app.txConfig.TxEncoder(),
			AuctionKeeper: *app.AuctionKeeper,
			MevLane:       mevLane,
			FreeLane:      freeLane,
		},
	)
	if err != nil {
		panic(err)
	}

	app.SetAnteHandler(anteHandler)
	return anteHandler
}

func (app *InitiaApp) setPostHandler() {
	postHandler, err := posthandler.NewPostHandler(
		posthandler.HandlerOptions{},
	)
	if err != nil {
		panic(err)
	}

	app.SetPostHandler(postHandler)
}

// Name returns the name of the App
func (app *InitiaApp) Name() string { return app.BaseApp.Name() }

// PreBlocker application updates every pre block
func (app *InitiaApp) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	res, err := app.ModuleManager.PreBlock(ctx)
	if err != nil {
		return nil, err
	}

	_, err = app.oraclePreBlockHandler.PreBlocker()(ctx, req)
	return res, err
}

// BeginBlocker application updates every begin block
func (app *InitiaApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.ModuleManager.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *InitiaApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.ModuleManager.EndBlock(ctx)
}

// InitChainer application update at chain initialization
func (app *InitiaApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState GenesisState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	if err := app.UpgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap()); err != nil {
		panic(err)
	}
	return app.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads a particular height
func (app *InitiaApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *InitiaApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// LegacyAmino returns SimApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *InitiaApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns Initia's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *InitiaApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns Initia's InterfaceRegistry
func (app *InitiaApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *InitiaApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *InitiaApp) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *InitiaApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *InitiaApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register new tendermint queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register node gRPC service for grpc-gateway.
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register grpc-gateway routes for all modules.
	app.BasicModuleManager.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(apiSvr.Router)
	}
}

// Simulate customize gas simulation to add fee deduction gas amount.
func (app *InitiaApp) Simulate(txBytes []byte) (sdk.GasInfo, *sdk.Result, error) {
	gasInfo, result, err := app.BaseApp.Simulate(txBytes)
	gasInfo.GasUsed += FeeDeductionGasAmount
	return gasInfo, result, err
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *InitiaApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(
		app.BaseApp.GRPCQueryRouter(), clientCtx,
		app.Simulate, app.interfaceRegistry,
	)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *InitiaApp) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(
		clientCtx, app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry, app.Query,
	)
}

func (app *InitiaApp) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(rtr *mux.Router) {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}
	return dupMaccPerms
}

// VerifyAddressLen ensures that the address matches the expected length
func VerifyAddressLen() func(addr []byte) error {
	return func(addr []byte) error {
		addrLen := len(addr)
		if addrLen != 20 && addrLen != movetypes.AddressBytesLength {
			return sdkerrors.ErrInvalidAddress
		}
		return nil
	}
}

//////////////////////////////////////
// TestingApp functions

// GetBaseApp implements the TestingApp interface.
func (app *InitiaApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

// GetAccountKeeper implements the TestingApp interface.
func (app *InitiaApp) GetAccountKeeper() *authkeeper.AccountKeeper {
	return app.AccountKeeper
}

// GetStakingKeeper implements the TestingApp interface.
func (app *InitiaApp) GetStakingKeeper() ibctestingtypes.StakingKeeper {
	return app.StakingKeeper
}

// GetIBCKeeper implements the TestingApp interface.
func (app *InitiaApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// GetICAControllerKeeper implements the TestingApp interface.
func (app *InitiaApp) GetICAControllerKeeper() *icacontrollerkeeper.Keeper {
	return app.ICAControllerKeeper
}

// GetICAAuthKeeper implements the TestingApp interface.
func (app *InitiaApp) GetICAAuthKeeper() *icaauthkeeper.Keeper {
	return app.ICAAuthKeeper
}

// GetScopedIBCKeeper implements the TestingApp interface.
func (app *InitiaApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

// TxConfig implements the TestingApp interface.
func (app *InitiaApp) TxConfig() client.TxConfig {
	return app.txConfig
}

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *InitiaApp) DefaultGenesis() map[string]json.RawMessage {
	return app.BasicModuleManager.DefaultGenesis(app.appCodec)
}

// Close closes the underlying baseapp, the oracle service, and the prometheus server if required.
// This method blocks on the closure of both the prometheus server, and the oracle-service
func (app *InitiaApp) Close() error {
	if err := app.BaseApp.Close(); err != nil {
		return err
	}

	// close the oracle service
	if app.OracleClient != nil {
		app.OracleClient.Stop()
	}

	return nil
}
