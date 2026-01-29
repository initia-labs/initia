package keepers

import (
	"os"

	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"

	packetforward "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward"
	packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/keeper"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/types"
	ratelimit "github.com/cosmos/ibc-apps/modules/rate-limiting/v8"
	ratelimitkeeper "github.com/cosmos/ibc-apps/modules/rate-limiting/v8/keeper"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v8/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	icacontroller "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	ibcfee "github.com/cosmos/ibc-go/v8/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	ibcnfttransfer "github.com/initia-labs/initia/x/ibc/nft-transfer"
	ibcnfttransferkeeper "github.com/initia-labs/initia/x/ibc/nft-transfer/keeper"
	ibcnfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	ibcperm "github.com/initia-labs/initia/x/ibc/perm"
	ibcpermkeeper "github.com/initia-labs/initia/x/ibc/perm/keeper"
	ibcpermtypes "github.com/initia-labs/initia/x/ibc/perm/types"
	icaauth "github.com/initia-labs/initia/x/intertx"
	icaauthkeeper "github.com/initia-labs/initia/x/intertx/keeper"
	icaauthtypes "github.com/initia-labs/initia/x/intertx/types"

	// this line is used by starport scaffolding # stargate/app/moduleImport

	appheaderinfo "github.com/initia-labs/initia/app/header_info"
	bankkeeper "github.com/initia-labs/initia/x/bank/keeper"
	distrkeeper "github.com/initia-labs/initia/x/distribution/keeper"
	dynamicfeekeeper "github.com/initia-labs/initia/x/dynamic-fee/keeper"
	dynamicfeetypes "github.com/initia-labs/initia/x/dynamic-fee/types"
	evidencekeeper "github.com/initia-labs/initia/x/evidence/keeper"
	govkeeper "github.com/initia-labs/initia/x/gov/keeper"
	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	ibchookskeeper "github.com/initia-labs/initia/x/ibc-hooks/keeper"
	ibcmovehooks "github.com/initia-labs/initia/x/ibc-hooks/move-hooks"
	ibchookstypes "github.com/initia-labs/initia/x/ibc-hooks/types"
	ibcupgrade "github.com/initia-labs/initia/x/ibc/upgrade"
	moveconfig "github.com/initia-labs/initia/x/move/config"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"
	movetypes "github.com/initia-labs/initia/x/move/types"
	stakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	rewardkeeper "github.com/initia-labs/initia/x/reward/keeper"
	rewardtypes "github.com/initia-labs/initia/x/reward/types"
	slashingkeeper "github.com/initia-labs/initia/x/slashing/keeper"

	// connect oracle dependencies

	marketmapkeeper "github.com/skip-mev/connect/v2/x/marketmap/keeper"
	marketmaptypes "github.com/skip-mev/connect/v2/x/marketmap/types"
	oraclekeeper "github.com/skip-mev/connect/v2/x/oracle/keeper"
	oracletypes "github.com/skip-mev/connect/v2/x/oracle/types"

	ophostkeeper "github.com/initia-labs/OPinit/x/ophost/keeper"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	ophosttypeshook "github.com/initia-labs/OPinit/x/ophost/types/hook"

	// noble forwarding keeper
	forwarding "github.com/noble-assets/forwarding/v2"
	forwardingkeeper "github.com/noble-assets/forwarding/v2/keeper"
	forwardingtypes "github.com/noble-assets/forwarding/v2/types"
)

type AppKeepers struct {
	ac address.Codec

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
	DynamicFeeKeeper      *dynamicfeekeeper.Keeper
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
	OPHostKeeper          *ophostkeeper.Keeper
	OracleKeeper          *oraclekeeper.Keeper // x/oracle keeper used for the connect oracle
	MarketMapKeeper       *marketmapkeeper.Keeper
	ForwardingKeeper      *forwardingkeeper.Keeper
	RatelimitKeeper       *ratelimitkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper           capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper      capabilitykeeper.ScopedKeeper
	ScopedNftTransferKeeper   capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
	ScopedICAAuthKeeper       capabilitykeeper.ScopedKeeper
}

func NewAppKeeper(
	ac, vc, cc address.Codec,
	appCodec codec.Codec,
	bApp *baseapp.BaseApp,
	legacyAmino *codec.LegacyAmino,
	maccPerms map[string][]string,
	blockedAddress map[string]bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	logger log.Logger,
	moveConfig moveconfig.MoveConfig,
	appOpts servertypes.AppOptions,
) AppKeepers {
	appKeepers := AppKeepers{
		ac: ac,
	}

	// Set keys KVStoreKey, TransientStoreKey, MemoryStoreKey
	appKeepers.GenerateKeys()

	// register streaming services
	if err := bApp.RegisterStreamingServices(appOpts, appKeepers.keys); err != nil {
		logger.Error("failed to load state streaming", "err", err)
		os.Exit(1)
	}

	authorityAccAddr := authtypes.NewModuleAddress(govtypes.ModuleName)
	authorityAddr, err := ac.BytesToString(authorityAccAddr)
	if err != nil {
		logger.Error("failed to retrieve authority address", "err", err)
		os.Exit(1)
	}

	// set the BaseApp's parameter store
	consensusParamsKeeper := consensusparamkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(appKeepers.keys[consensusparamtypes.StoreKey]), authorityAddr, runtime.EventService{})
	appKeepers.ConsensusParamsKeeper = &consensusParamsKeeper
	bApp.SetParamStore(appKeepers.ConsensusParamsKeeper.ParamsStore)

	// add capability keeper and ScopeToModule for ibc module
	appKeepers.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, appKeepers.keys[capabilitytypes.StoreKey], appKeepers.memKeys[capabilitytypes.MemStoreKey])

	// grant capabilities for the ibc and ibc-transfer modules
	appKeepers.ScopedIBCKeeper = appKeepers.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	appKeepers.ScopedTransferKeeper = appKeepers.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	appKeepers.ScopedNftTransferKeeper = appKeepers.CapabilityKeeper.ScopeToModule(ibcnfttransfertypes.ModuleName)
	appKeepers.ScopedICAHostKeeper = appKeepers.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	appKeepers.ScopedICAControllerKeeper = appKeepers.CapabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)
	appKeepers.ScopedICAAuthKeeper = appKeepers.CapabilityKeeper.ScopeToModule(icaauthtypes.ModuleName)

	// Applications that wish to enforce statically created ScopedKeepers should call `Seal` after creating
	// their scoped modules in `NewApp` with `ScopeToModule`
	appKeepers.CapabilityKeeper.Seal()

	// add keepers
	appKeepers.MoveKeeper = &movekeeper.Keeper{}

	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		ac,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authorityAddr,
	)
	appKeepers.AccountKeeper = &accountKeeper

	bankKeeper := bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[banktypes.StoreKey]),
		appKeepers.AccountKeeper,
		movekeeper.NewMoveBankKeeper(appKeepers.MoveKeeper),
		blockedAddress,
		authorityAddr,
	)
	appKeepers.BankKeeper = &bankKeeper

	appKeepers.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[stakingtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		movekeeper.NewVotingPowerKeeper(appKeepers.MoveKeeper),
		authorityAddr,
		vc,
		cc,
	)

	appKeepers.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[distrtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		movekeeper.NewDexKeeper(appKeepers.MoveKeeper),
		authtypes.FeeCollectorName,
		authorityAddr,
	)

	appKeepers.RewardKeeper = rewardkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[rewardtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.DistrKeeper,
		authtypes.FeeCollectorName,
		authorityAddr,
	)

	slashingKeeper := slashingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[slashingtypes.StoreKey]),
		appKeepers.StakingKeeper,
		authorityAddr,
	)
	appKeepers.SlashingKeeper = &slashingKeeper

	appKeepers.CrisisKeeper = crisiskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[crisistypes.StoreKey]),
		invCheckPeriod,
		appKeepers.BankKeeper,
		authtypes.FeeCollectorName,
		authorityAddr,
		ac,
	)

	appKeepers.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(appKeepers.keys[upgradetypes.StoreKey]),
		appCodec,
		homePath,
		bApp,
		authorityAddr,
	)

	feeGrantKeeper := feegrantkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(appKeepers.keys[feegrant.StoreKey]), appKeepers.AccountKeeper)
	appKeepers.FeeGrantKeeper = &feeGrantKeeper

	authzKeeper := authzkeeper.NewKeeper(runtime.NewKVStoreService(appKeepers.keys[authzkeeper.StoreKey]), appCodec, bApp.MsgServiceRouter(), appKeepers.AccountKeeper)
	authzKeeper = authzKeeper.SetBankKeeper(appKeepers.BankKeeper)
	appKeepers.AuthzKeeper = &authzKeeper

	// Create evidence Keeper for to register the IBC light client misbehavior evidence route
	appKeepers.EvidenceKeeper = evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[evidencetypes.StoreKey]),
		appKeepers.StakingKeeper,
		appKeepers.SlashingKeeper,
		ac,
		runtime.ProvideCometInfoService(),
	)

	groupConfig := group.DefaultConfig()
	groupKeeper := groupkeeper.NewKeeper(
		appKeepers.keys[group.StoreKey],
		appCodec,
		bApp.MsgServiceRouter(),
		appKeepers.AccountKeeper,
		groupConfig,
	)
	appKeepers.GroupKeeper = &groupKeeper

	dynamicFeeKeeper := dynamicfeekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[dynamicfeetypes.StoreKey]),
		runtime.NewTransientStoreService(appKeepers.tkeys[dynamicfeetypes.TStoreKey]),
		movekeeper.NewDexKeeper(appKeepers.MoveKeeper),
		appKeepers.MoveKeeper,
		appKeepers.MoveKeeper,
		ac,
		authorityAddr,
	)
	appKeepers.DynamicFeeKeeper = dynamicFeeKeeper

	// Create IBC Keeper
	appKeepers.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		appKeepers.keys[ibcexported.StoreKey],
		nil, // we don't need migration
		appKeepers.StakingKeeper,
		appKeepers.UpgradeKeeper,
		appKeepers.ScopedIBCKeeper,
		authorityAddr,
	)

	ibcFeeKeeper := ibcfeekeeper.NewKeeper(
		appCodec,
		appKeepers.keys[ibcfeetypes.StoreKey],
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.PortKeeper,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
	)
	appKeepers.IBCFeeKeeper = &ibcFeeKeeper

	appKeepers.IBCPermKeeper = ibcpermkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[ibcpermtypes.StoreKey]),
		authorityAddr,
		ac,
	)

	marketMapKeeper := marketmapkeeper.NewKeeper(
		runtime.NewKVStoreService(appKeepers.keys[marketmaptypes.StoreKey]),
		appCodec,
		authorityAccAddr,
	)
	appKeepers.MarketMapKeeper = marketMapKeeper

	oracleKeeper := oraclekeeper.NewKeeper(
		runtime.NewKVStoreService(appKeepers.keys[oracletypes.StoreKey]),
		appCodec,
		appKeepers.MarketMapKeeper,
		authorityAccAddr,
	)
	appKeepers.OracleKeeper = &oracleKeeper

	// Add the oracle keeper as a hook to market map keeper so new market map entries can be created
	// and propagated to the oracle keeper.
	appKeepers.MarketMapKeeper.SetHooks(appKeepers.OracleKeeper.Hooks())

	appKeepers.IBCHooksKeeper = ibchookskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[ibchookstypes.StoreKey]),
		runtime.NewTransientStoreService(appKeepers.tkeys[ibchookstypes.TStoreKey]),
		authorityAddr,
		ac,
	)

	appKeepers.ForwardingKeeper = forwardingkeeper.NewKeeper(
		appCodec,
		logger,
		runtime.NewKVStoreService(appKeepers.keys[forwardingtypes.StoreKey]),
		runtime.NewTransientStoreService(appKeepers.tkeys[forwardingtypes.TransientStoreKey]),
		appheaderinfo.NewHeaderInfoService(),
		runtime.ProvideEventService(),
		authorityAddr,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.TransferKeeper,
	)
	appKeepers.BankKeeper.AppendSendRestriction(appKeepers.ForwardingKeeper.SendRestrictionFn)

	////////////////////////////
	// Transfer configuration //
	////////////////////////////
	// Send   : transfer -> packet forward -> rate limit -> fee  -> channel
	// Receive: channel  -> upgrade        -> perm       -> fee  -> move -> rate limit -> packet forward -> forwarding -> transfer

	var transferStack porttypes.IBCModule
	{
		packetForwardKeeper := &packetforwardkeeper.Keeper{}
		rateLimitKeeper := &ratelimitkeeper.Keeper{}
		ibcHooksICS4Wrapper := &ibchooks.ICS4Middleware{}

		// create Transfer Keepers
		transferKeeper := ibctransferkeeper.NewKeeper(
			appCodec,
			appKeepers.keys[ibctransfertypes.StoreKey],
			nil, // we don't need migration
			// ics4wrapper: transfer -> packet forward
			packetForwardKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.IBCKeeper.PortKeeper,
			appKeepers.AccountKeeper,
			appKeepers.BankKeeper,
			appKeepers.ScopedTransferKeeper,
			authorityAddr,
		)
		appKeepers.TransferKeeper = &transferKeeper
		transferIBCModule := ibctransfer.NewIBCModule(*appKeepers.TransferKeeper)
		transferStack = transferIBCModule

		// forwarding middleware
		transferStack = forwarding.NewMiddleware(
			// receive: forwarding -> transfer
			transferStack,
			appKeepers.AccountKeeper,
			appKeepers.ForwardingKeeper,
		)

		// create packet forward middleware
		*packetForwardKeeper = *packetforwardkeeper.NewKeeper(
			appCodec,
			appKeepers.keys[packetforwardtypes.StoreKey],
			appKeepers.TransferKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.BankKeeper,
			// ics4wrapper: transfer -> packet forward -> rate limit
			rateLimitKeeper,
			authorityAddr,
		)
		appKeepers.PacketForwardKeeper = packetForwardKeeper
		transferStack = packetforward.NewIBCMiddleware(
			// receive: packet forward -> forwarding -> transfer
			transferStack,
			appKeepers.PacketForwardKeeper,
			0,
			packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp,
		)

		// create the rate limit keeper
		*rateLimitKeeper = *ratelimitkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(appKeepers.keys[ratelimittypes.StoreKey]),
			paramtypes.Subspace{}, // empty params
			authorityAddr,
			appKeepers.BankKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			// ics4wrapper: transfer -> packet forward -> rate limit -> ibchooks
			ibcHooksICS4Wrapper,
		)
		appKeepers.RatelimitKeeper = rateLimitKeeper

		// rate limit middleware
		transferStack = ratelimit.NewIBCMiddleware(
			*appKeepers.RatelimitKeeper,
			// receive: rate limit -> packet forward -> forwarding -> transfer
			transferStack,
		)

		// create move middleware for transfer
		*ibcHooksICS4Wrapper = *ibchooks.NewICS4Middleware(
			// ics4wrapper: transfer -> packet forward -> rate limit -> ibchooks -> fee
			appKeepers.IBCFeeKeeper,
			appKeepers.IBCHooksKeeper,
			ibcmovehooks.NewMoveHooks(ac, appCodec, logger, appKeepers.MoveKeeper),
		)
		transferStack = ibchooks.NewIBCMiddleware(
			// receive: move -> rate limit -> packet forward -> forwarding -> transfer
			transferStack,
			ibcHooksICS4Wrapper,
			appKeepers.IBCHooksKeeper,
		)

		// create ibcfee middleware for transfer
		transferStack = ibcfee.NewIBCMiddleware(
			// receive: fee -> move -> rate limit -> packet forward -> forwarding -> transfer
			transferStack,
			*appKeepers.IBCFeeKeeper,
		)

		// create perm middleware for transfer
		transferStack = ibcperm.NewIBCMiddleware(
			// receive: perm -> fee -> move -> rate limit -> packet forward -> forwarding -> transfer
			transferStack,
			// ics4wrapper: not used
			nil,
			*appKeepers.IBCPermKeeper,
		)

		// create upgrade middleware for transfer
		transferStack = ibcupgrade.NewIBCMiddleware(
			// receive: upgrade -> perm -> fee -> move -> rate limit -> packet forward -> forwarding -> transfer
			transferStack,
			// ics4wrapper: not used
			nil,
			// upgrade: upgrade -> transfer
			transferIBCModule,
		)
	}

	////////////////////////////////
	// Nft Transfer configuration //
	////////////////////////////////

	var nftTransferStack porttypes.IBCModule
	{
		ibcHooksICS4Wrapper := &ibchooks.ICS4Middleware{}

		// Create Transfer Keepers
		appKeepers.NftTransferKeeper = ibcnfttransferkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(appKeepers.keys[ibcnfttransfertypes.StoreKey]),
			// ics4wrapper: nft transfer -> ibchooks
			ibcHooksICS4Wrapper,
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.IBCKeeper.PortKeeper,
			appKeepers.AccountKeeper,
			movekeeper.NewNftKeeper(appKeepers.MoveKeeper),
			appKeepers.ScopedNftTransferKeeper,
			authorityAddr,
		)
		nftTransferIBCModule := ibcnfttransfer.NewIBCModule(*appKeepers.NftTransferKeeper)
		nftTransferStack = nftTransferIBCModule

		// create move middleware for nft-transfer
		*ibcHooksICS4Wrapper = *ibchooks.NewICS4Middleware(
			// ics4wrapper: nft transfer -> ibchooks -> fee
			appKeepers.IBCFeeKeeper,
			appKeepers.IBCHooksKeeper,
			ibcmovehooks.NewMoveHooks(ac, appCodec, logger, appKeepers.MoveKeeper),
		)
		nftTransferStack = ibchooks.NewIBCMiddleware(
			// receive: move -> nft-transfer
			nftTransferStack,
			ibcHooksICS4Wrapper,
			appKeepers.IBCHooksKeeper,
		)

		nftTransferStack = ibcperm.NewIBCMiddleware(
			// receive: perm -> fee -> nft transfer
			ibcfee.NewIBCMiddleware(
				// receive: channel -> fee -> move -> nft transfer
				nftTransferStack,
				*appKeepers.IBCFeeKeeper,
			),
			// ics4wrapper: not used
			nil,
			*appKeepers.IBCPermKeeper,
		)
	}

	///////////////////////
	// ICA configuration //
	///////////////////////

	var icaHostStack porttypes.IBCModule
	var icaControllerStack porttypes.IBCModule
	{
		icaHostKeeper := icahostkeeper.NewKeeper(
			appCodec, appKeepers.keys[icahosttypes.StoreKey],
			nil, // we don't need migration
			appKeepers.IBCFeeKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.IBCKeeper.PortKeeper,
			appKeepers.AccountKeeper,
			appKeepers.ScopedICAHostKeeper,
			bApp.MsgServiceRouter(),
			authorityAddr,
		)
		icaHostKeeper.WithQueryRouter(bApp.GRPCQueryRouter())
		// icaHostKeeper.WithICS4Wrapper()
		appKeepers.ICAHostKeeper = &icaHostKeeper

		icaControllerKeeper := icacontrollerkeeper.NewKeeper(
			appCodec, appKeepers.keys[icacontrollertypes.StoreKey],
			nil, // we don't need migration
			appKeepers.IBCFeeKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.IBCKeeper.PortKeeper,
			appKeepers.ScopedICAControllerKeeper,
			bApp.MsgServiceRouter(),
			authorityAddr,
		)
		// icaControllerKeeper.WithICS4Wrapper()
		appKeepers.ICAControllerKeeper = &icaControllerKeeper

		icaAuthKeeper := icaauthkeeper.NewKeeper(
			appCodec,
			*appKeepers.ICAControllerKeeper,
			appKeepers.ScopedICAAuthKeeper,
			ac,
		)
		appKeepers.ICAAuthKeeper = &icaAuthKeeper

		icaAuthIBCModule := icaauth.NewIBCModule(*appKeepers.ICAAuthKeeper)
		icaHostIBCModule := icahost.NewIBCModule(*appKeepers.ICAHostKeeper)
		icaHostStack = ibcperm.NewIBCMiddleware(
			// receive: perm -> fee -> ica host
			ibcfee.NewIBCMiddleware(icaHostIBCModule, *appKeepers.IBCFeeKeeper),
			// ics4wrapper: not used
			nil,
			*appKeepers.IBCPermKeeper,
		)
		icaControllerIBCModule := icacontroller.NewIBCMiddleware(icaAuthIBCModule, *appKeepers.ICAControllerKeeper)
		icaControllerStack = ibcperm.NewIBCMiddleware(
			// receive: perm -> fee -> ica controller
			ibcfee.NewIBCMiddleware(icaControllerIBCModule, *appKeepers.IBCFeeKeeper),
			// ics4wrapper: not used
			nil,
			*appKeepers.IBCPermKeeper,
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
	appKeepers.IBCKeeper.SetRouter(ibcRouter)

	//////////////////////////////
	// MoveKeeper Configuration //
	//////////////////////////////
	queryWhitelist := appKeepers.makeQueryWhitelist()

	*appKeepers.MoveKeeper = movekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[movetypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.OracleKeeper,
		// appKeepers.NftTransferKeeper,
		bApp.MsgServiceRouter(),
		bApp.GRPCQueryRouter(),
		moveConfig,
		// staking feature
		appKeepers.DistrKeeper,
		appKeepers.StakingKeeper,
		appKeepers.RewardKeeper,
		appKeepers.DistrKeeper,
		authtypes.FeeCollectorName,
		authorityAddr,
		ac, vc,
	).WithVMQueryWhitelist(queryWhitelist)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	appKeepers.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			appKeepers.DistrKeeper.Hooks(),
			appKeepers.SlashingKeeper.Hooks(),
		),
	)
	appKeepers.StakingKeeper.SetSlashingHooks(appKeepers.MoveKeeper.Hooks())

	appKeepers.OPHostKeeper = ophostkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[ophosttypes.StoreKey]),
		bApp.MsgServiceRouter(),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.DistrKeeper,
		ophosttypes.NewBridgeHooks(ophosttypeshook.NewBridgeHook(appKeepers.IBCKeeper.ChannelKeeper, appKeepers.IBCPermKeeper, ac)),
		authorityAddr,
	)

	govConfig := govtypes.DefaultConfig()
	appKeepers.GovKeeper = govkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(appKeepers.keys[govtypes.StoreKey]), appKeepers.AccountKeeper, appKeepers.BankKeeper,
		appKeepers.StakingKeeper, appKeepers.DistrKeeper, movekeeper.NewVestingKeeper(appKeepers.MoveKeeper), bApp.MsgServiceRouter(), govConfig, authorityAddr,
	)

	return appKeepers
}
