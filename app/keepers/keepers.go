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
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"

	packetforward "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward"
	packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/keeper"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/types"
	ratelimit "github.com/cosmos/ibc-apps/modules/rate-limiting/v10"
	ratelimitkeeper "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/keeper"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	ratelimitv2 "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/v2"
	icacontroller "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	ibctransfer "github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	ibctransferv2 "github.com/cosmos/ibc-go/v10/modules/apps/transfer/v2"

	ibctransferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"

	ibcnfttransfer "github.com/initia-labs/initia/x/ibc/nft-transfer"
	ibcnfttransferkeeper "github.com/initia-labs/initia/x/ibc/nft-transfer/keeper"
	ibcnfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	ibcnfttransferv2 "github.com/initia-labs/initia/x/ibc/nft-transfer/v2"
	ibcperm "github.com/initia-labs/initia/x/ibc/perm"
	ibcpermv2 "github.com/initia-labs/initia/x/ibc/perm/v2"

	ibcpermkeeper "github.com/initia-labs/initia/x/ibc/perm/keeper"
	ibcpermtypes "github.com/initia-labs/initia/x/ibc/perm/types"

	// this line is used by starport scaffolding # stargate/app/moduleImport

	applanes "github.com/initia-labs/initia/app/lanes"
	bankkeeper "github.com/initia-labs/initia/x/bank/keeper"
	distrkeeper "github.com/initia-labs/initia/x/distribution/keeper"
	dynamicfeekeeper "github.com/initia-labs/initia/x/dynamic-fee/keeper"
	dynamicfeetypes "github.com/initia-labs/initia/x/dynamic-fee/types"
	evidencekeeper "github.com/initia-labs/initia/x/evidence/keeper"
	govkeeper "github.com/initia-labs/initia/x/gov/keeper"
	ibchooks "github.com/initia-labs/initia/x/ibc-hooks"
	ibchooksv2 "github.com/initia-labs/initia/x/ibc-hooks/v2"

	ibchookskeeper "github.com/initia-labs/initia/x/ibc-hooks/keeper"
	ibcmovehooks "github.com/initia-labs/initia/x/ibc-hooks/move-hooks"
	ibchookstypes "github.com/initia-labs/initia/x/ibc-hooks/types"
	moveconfig "github.com/initia-labs/initia/x/move/config"
	movekeeper "github.com/initia-labs/initia/x/move/keeper"
	movetypes "github.com/initia-labs/initia/x/move/types"
	stakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	rewardkeeper "github.com/initia-labs/initia/x/reward/keeper"
	rewardtypes "github.com/initia-labs/initia/x/reward/types"
	slashingkeeper "github.com/initia-labs/initia/x/slashing/keeper"

	// block-sdk dependencies

	auctionkeeper "github.com/skip-mev/block-sdk/v2/x/auction/keeper"
	auctiontypes "github.com/skip-mev/block-sdk/v2/x/auction/types"

	// connect oracle dependencies

	marketmapkeeper "github.com/skip-mev/connect/v2/x/marketmap/keeper"
	marketmaptypes "github.com/skip-mev/connect/v2/x/marketmap/types"
	oraclekeeper "github.com/skip-mev/connect/v2/x/oracle/keeper"
	oracletypes "github.com/skip-mev/connect/v2/x/oracle/types"

	ophostkeeper "github.com/initia-labs/OPinit/x/ophost/keeper"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	ophosttypeshook "github.com/initia-labs/OPinit/x/ophost/types/hook"

	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	appheaderinfo "github.com/initia-labs/initia/app/header_info"
	"github.com/noble-assets/forwarding/v2"
	forwardingibcv2 "github.com/noble-assets/forwarding/v2/v2"

	forwardingkeeper "github.com/noble-assets/forwarding/v2/keeper"
	forwardingtypes "github.com/noble-assets/forwarding/v2/types"
)

type AppKeepers struct {
	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper         *authkeeper.AccountKeeper
	BankKeeper            *bankkeeper.BaseKeeper
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
	IBCPermKeeper         *ibcpermkeeper.Keeper
	PacketForwardKeeper   *packetforwardkeeper.Keeper
	MoveKeeper            *movekeeper.Keeper
	IBCHooksKeeper        *ibchookskeeper.Keeper
	AuctionKeeper         *auctionkeeper.Keeper // x/auction keeper used to process bids for TOB auctions
	OPHostKeeper          *ophostkeeper.Keeper
	OracleKeeper          *oraclekeeper.Keeper // x/oracle keeper used for the connect oracle
	MarketMapKeeper       *marketmapkeeper.Keeper
	RatelimitKeeper       *ratelimitkeeper.Keeper
	ForwardingKeeper      *forwardingkeeper.Keeper
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
	appKeepers := AppKeepers{}

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

	appKeepers.RewardKeeper = rewardkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[rewardtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authtypes.FeeCollectorName,
		authorityAddr,
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

	// Create evidence Keeper for to register the IBC light client misbehaviour evidence route
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
		runtime.NewKVStoreService(appKeepers.keys[ibcexported.StoreKey]),
		nil, // we don't need migration
		appKeepers.UpgradeKeeper,
		authorityAddr,
	)

	clientKeeper := appKeepers.IBCKeeper.ClientKeeper
	storeProvider := clientKeeper.GetStoreProvider()
	tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
	clientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)

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

	//////////////////////////////
	// TransferV1 configuration //
	//////////////////////////////
	// Send   : transfer -> packet forward -> rate limit -> channel
	// Receive: channel  -> perm           -> move       -> rate limit -> packet forward -> forwarding -> transfer

	var transferStack porttypes.IBCModule
	var transferStackV2 ibcapi.IBCModule
	{
		packetForwardKeeper := &packetforwardkeeper.Keeper{}
		rateLimitKeeper := &ratelimitkeeper.Keeper{}

		// create Transfer Keepers
		transferKeeper := ibctransferkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(appKeepers.keys[ibctransfertypes.StoreKey]),
			nil, // we don't need migration
			// ics4wrapper: transfer -> packet forward
			packetForwardKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			bApp.MsgServiceRouter(),
			appKeepers.AccountKeeper,
			appKeepers.BankKeeper,
			authorityAddr,
		)
		appKeepers.TransferKeeper = &transferKeeper
		transferModule := ibctransfer.NewIBCModule(*appKeepers.TransferKeeper)

		forwardingMiddleware := forwarding.NewMiddleware(
			// receive: forwarding -> transfer
			transferModule,
			appKeepers.AccountKeeper,
			appKeepers.ForwardingKeeper,
		)

		// create packet forward middleware
		*packetForwardKeeper = *packetforwardkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(appKeepers.keys[packetforwardtypes.StoreKey]),
			appKeepers.TransferKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.BankKeeper,
			// ics4wrapper: transfer -> packet forward -> rate limit
			rateLimitKeeper,
			authorityAddr,
		)
		appKeepers.PacketForwardKeeper = packetForwardKeeper
		packetForwardMiddleware := packetforward.NewIBCMiddleware(
			// receive: packet forward -> forwarding -> transfer
			forwardingMiddleware,
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
			appKeepers.IBCKeeper.ClientKeeper,
			// ics4wrapper: transfer -> packet forward -> rate limit
			appKeepers.IBCKeeper.ChannelKeeper,
		)
		appKeepers.RatelimitKeeper = rateLimitKeeper

		// rate limit middleware
		rateLimitMiddleware := ratelimit.NewIBCMiddleware(
			*appKeepers.RatelimitKeeper,
			// receive: rate limit -> packet forward -> forwarding -> transfer
			packetForwardMiddleware,
		)

		// create move middleware for transfer
		moveHookMiddleware := ibchooks.NewIBCMiddleware(
			// receive: move -> rate limit -> packet forward -> forwarding -> transfer
			rateLimitMiddleware,
			ibchooks.NewICS4Middleware(
				nil, /* ics4wrapper: not used */
				ibcmovehooks.NewMoveHooks(appCodec, ac, appKeepers.MoveKeeper),
			),
			appKeepers.IBCHooksKeeper,
		)

		// create perm middleware for transfer
		transferStack = ibcperm.NewIBCMiddleware(
			// receive: perm -> move -> rate limit -> packet forward -> forwarding -> transfer
			moveHookMiddleware,
			// ics4wrapper: not used
			nil,
			*appKeepers.IBCPermKeeper,
		)

		// v2
		transferStackV2 = ibctransferv2.NewIBCModule(*appKeepers.TransferKeeper)
		// TODO: packet forwarding
		forwardingMiddlewareV2 := forwardingibcv2.NewMiddleware(
			// receive: forwarding -> transfer
			transferStackV2,
			appKeepers.AccountKeeper,
			appKeepers.ForwardingKeeper,
		)
		ratelimitMiddlewareV2 := ratelimitv2.NewIBCMiddleware(*appKeepers.RatelimitKeeper, &forwardingMiddlewareV2)
		movehookMiddlewareV2 := ibchooksv2.NewIBCMiddleware(ratelimitMiddlewareV2, nil, appKeepers.IBCHooksKeeper)
		transferStackV2 = ibcpermv2.NewIBCMiddleware(
			&movehookMiddlewareV2,
			*appKeepers.IBCPermKeeper,
		)

	}

	////////////////////////////////
	// Nft Transfer configuration //
	////////////////////////////////

	var nftTransferStack porttypes.IBCModule
	var nftTransferStackV2 ibcapi.IBCModule
	{
		// Create Transfer Keepers
		appKeepers.NftTransferKeeper = ibcnfttransferkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(appKeepers.keys[ibcnfttransfertypes.StoreKey]),
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.AccountKeeper,
			movekeeper.NewNftKeeper(appKeepers.MoveKeeper),
			authorityAddr,
		)
		nftTransferIBCModule := ibcnfttransfer.NewIBCModule(*appKeepers.NftTransferKeeper)

		// create move middleware for nft-transfer
		hookMiddleware := ibchooks.NewIBCMiddleware(
			// receive: move -> nft-transfer
			nftTransferIBCModule,
			nil,
			appKeepers.IBCHooksKeeper,
		)

		nftTransferStack = ibcperm.NewIBCMiddleware(
			// receive: perm -> move -> nft transfer
			hookMiddleware,
			// ics4wrapper: not used
			nil,
			*appKeepers.IBCPermKeeper,
		)

		// v2
		nftTransferStackV2 = ibcnfttransferv2.NewIBCModule(*appKeepers.NftTransferKeeper)
		movehookMiddlewareNftV2 := ibchooksv2.NewIBCMiddleware(nftTransferStackV2, nil, appKeepers.IBCHooksKeeper)
		nftTransferStackV2 = ibcpermv2.NewIBCMiddleware(
			&movehookMiddlewareNftV2,
			*appKeepers.IBCPermKeeper,
		)
	}

	///////////////////////
	// ICA configuration //
	///////////////////////
	// TODO icahost stack, ica controller stack for ibc v2
	var icaHostStack porttypes.IBCModule
	var icaControllerStack porttypes.IBCModule
	{
		icaHostKeeper := icahostkeeper.NewKeeper(
			appCodec, runtime.NewKVStoreService(appKeepers.keys[icahosttypes.StoreKey]),
			nil, // we don't need migration
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.AccountKeeper,
			bApp.MsgServiceRouter(),
			bApp.GRPCQueryRouter(),
			authorityAddr,
		)
		appKeepers.ICAHostKeeper = &icaHostKeeper

		icaControllerKeeper := icacontrollerkeeper.NewKeeper(
			appCodec, runtime.NewKVStoreService(appKeepers.keys[icacontrollertypes.StoreKey]),
			nil, // we don't need migration
			appKeepers.IBCKeeper.ChannelKeeper,
			appKeepers.IBCKeeper.ChannelKeeper,
			bApp.MsgServiceRouter(),
			authorityAddr,
		)
		appKeepers.ICAControllerKeeper = &icaControllerKeeper

		icaHostIBCModule := icahost.NewIBCModule(*appKeepers.ICAHostKeeper)
		icaHostStack = ibcperm.NewIBCMiddleware(
			// receive: perm -> ica host
			icaHostIBCModule,
			// ics4wrapper: not used
			nil,
			*appKeepers.IBCPermKeeper,
		)
		icaControllerIBCModule := icacontroller.NewIBCMiddleware(*appKeepers.ICAControllerKeeper)
		icaControllerStack = ibcperm.NewIBCMiddleware(
			// receive: perm -> ica controller
			icaControllerIBCModule,
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
		AddRoute(ibcnfttransfertypes.ModuleName, nftTransferStack)

	ibcRouterV2 := ibcapi.NewRouter()
	ibcRouterV2.AddRoute(ibctransfertypes.PortID, transferStackV2).
		AddRoute(ibcnfttransfertypes.PortID, nftTransferStackV2)

	appKeepers.IBCKeeper.SetRouter(ibcRouter)
	appKeepers.IBCKeeper.SetRouterV2(ibcRouterV2)

	//////////////////////////////
	// MoveKeeper Configuration //
	//////////////////////////////

	queryWhitelist := movetypes.DefaultVMQueryWhiteList(ac)
	queryWhitelist.Stargate["/initia.mstaking.v1.Query/UnbondingDelegation"] = movetypes.ProtoSet{
		Request:  &stakingtypes.QueryUnbondingDelegationRequest{},
		Response: &stakingtypes.QueryUnbondingDelegationResponse{},
	}
	queryWhitelist.Stargate["/initia.mstaking.v1.Query/Pool"] = movetypes.ProtoSet{
		Request:  &stakingtypes.QueryPoolRequest{},
		Response: &stakingtypes.QueryPoolResponse{},
	}
	queryWhitelist.Stargate["/initia.mstaking.v1.Query/DelegatorDelegations"] = movetypes.ProtoSet{
		Request:  &stakingtypes.QueryDelegatorDelegationsRequest{},
		Response: &stakingtypes.QueryDelegatorDelegationsResponse{},
	}
	queryWhitelist.Stargate["/initia.mstaking.v1.Query/DelegatorTotalDelegationBalance"] = movetypes.ProtoSet{
		Request:  &stakingtypes.QueryDelegatorTotalDelegationBalanceRequest{},
		Response: &stakingtypes.QueryDelegatorTotalDelegationBalanceResponse{},
	}
	queryWhitelist.Stargate["/initia.mstaking.v1.Query/Delegation"] = movetypes.ProtoSet{
		Request:  &stakingtypes.QueryDelegationRequest{},
		Response: &stakingtypes.QueryDelegationResponse{},
	}
	queryWhitelist.Stargate["/initia.mstaking.v1.Query/Redelegations"] = movetypes.ProtoSet{
		Request:  &stakingtypes.QueryRedelegationsRequest{},
		Response: &stakingtypes.QueryRedelegationsResponse{},
	}
	queryWhitelist.Stargate["/connect.oracle.v2.Query/GetAllCurrencyPairs"] = movetypes.ProtoSet{
		Request:  &oracletypes.GetAllCurrencyPairsRequest{},
		Response: &oracletypes.GetAllCurrencyPairsResponse{},
	}
	queryWhitelist.Stargate["/connect.oracle.v2.Query/GetPrice"] = movetypes.ProtoSet{
		Request:  &oracletypes.GetPriceRequest{},
		Response: &oracletypes.GetPriceResponse{},
	}
	queryWhitelist.Stargate["/connect.oracle.v2.Query/GetPrices"] = movetypes.ProtoSet{
		Request:  &oracletypes.GetPricesRequest{},
		Response: &oracletypes.GetPricesResponse{},
	}

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

	// x/auction module keeper initialization

	// initialize the keeper
	auctionKeeper := auctionkeeper.NewKeeperWithRewardsAddressProvider(
		appCodec,
		appKeepers.keys[auctiontypes.StoreKey],
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		applanes.NewRewardsAddressProvider(*appKeepers.StakingKeeper, *appKeepers.DistrKeeper),
		authorityAddr,
	)
	appKeepers.AuctionKeeper = &auctionKeeper

	appKeepers.OPHostKeeper = ophostkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(appKeepers.keys[ophosttypes.StoreKey]),
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
