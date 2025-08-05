package keeper_test

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/gogoproto/proto"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/tx/signing"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codecaddress "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/std"
	testutilsims "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"

	ibctransfer "github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	ibcnfttransfer "github.com/initia-labs/initia/x/ibc/nft-transfer"

	initiaapp "github.com/initia-labs/initia/app"
	initiaappparams "github.com/initia-labs/initia/app/params"
	"github.com/initia-labs/initia/x/bank"
	movebank "github.com/initia-labs/initia/x/bank/keeper"
	"github.com/initia-labs/initia/x/distribution"
	distrkeeper "github.com/initia-labs/initia/x/distribution/keeper"
	customdistrtypes "github.com/initia-labs/initia/x/distribution/types"
	"github.com/initia-labs/initia/x/gov"
	govkeeper "github.com/initia-labs/initia/x/gov/keeper"
	customgovtypes "github.com/initia-labs/initia/x/gov/types"
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

	vmapi "github.com/initia-labs/movevm/api"
	"github.com/initia-labs/movevm/precompile"
	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/skip-mev/connect/v2/x/oracle"
	oraclekeeper "github.com/skip-mev/connect/v2/x/oracle/keeper"
	oracletypes "github.com/skip-mev/connect/v2/x/oracle/types"

	dynamicfeekeeper "github.com/initia-labs/initia/x/dynamic-fee/keeper"
	dynamicfeetypes "github.com/initia-labs/initia/x/dynamic-fee/types"
)

var ModuleBasics = module.NewBasicManager(
	auth.AppModuleBasic{},
	bank.AppModuleBasic{},
	staking.AppModuleBasic{},
	reward.AppModuleBasic{},
	distribution.AppModuleBasic{},
	gov.AppModuleBasic{},
	slashing.AppModuleBasic{},
	move.AppModuleBasic{},
	oracle.AppModuleBasic{},
	ibctransfer.AppModuleBasic{},
	ibcnfttransfer.AppModuleBasic{},
)

// Bond denom should be set for staking test
const bondDenom = initiaapp.BondDenom

var (
	valPubKeys = testutilsims.CreateTestPubKeys(5)

	pubKeys = []crypto.PubKey{
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
	}

	addrs = []sdk.AccAddress{
		sdk.AccAddress(pubKeys[0].Address()),
		sdk.AccAddress(pubKeys[1].Address()),
		sdk.AccAddress(pubKeys[2].Address()),
		sdk.AccAddress(pubKeys[3].Address()),
		sdk.AccAddress(pubKeys[4].Address()),
	}

	valAddrs = []sdk.ValAddress{
		sdk.ValAddress(pubKeys[0].Address()),
		sdk.ValAddress(pubKeys[1].Address()),
		sdk.ValAddress(pubKeys[2].Address()),
		sdk.ValAddress(pubKeys[3].Address()),
		sdk.ValAddress(pubKeys[4].Address()),
	}

	testDenoms = []string{
		"test1",
		"test2",
		"test3",
		"test4",
		"test5",
	}

	initiaSupply = math.NewInt(100_000_000_000)
)

func MakeTestCodec(t testing.TB) codec.Codec {
	return MakeEncodingConfig(t).Codec
}

func MakeEncodingConfig(_ testing.TB) initiaappparams.EncodingConfig {
	interfaceRegistry, _ := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          codecaddress.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
			ValidatorAddressCodec: codecaddress.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		},
	})
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()
	txConfig := tx.NewTxConfig(appCodec, tx.DefaultSignModes)

	std.RegisterInterfaces(interfaceRegistry)
	std.RegisterLegacyAminoCodec(legacyAmino)

	ModuleBasics.RegisterLegacyAminoCodec(legacyAmino)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)

	return initiaappparams.EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             appCodec,
		TxConfig:          txConfig,
		Amino:             legacyAmino,
	}
}

func initialTotalSupply() sdk.Coins {
	faucetBalance := sdk.NewCoins(sdk.NewCoin(bondDenom, initiaSupply))
	for _, testDenom := range testDenoms {
		faucetBalance = faucetBalance.Add(sdk.NewCoin(testDenom, initiaSupply))
	}

	return faucetBalance
}

type TestFaucet struct {
	t                testing.TB
	bankKeeper       bankkeeper.Keeper
	moveKeeper       movekeeper.Keeper
	sender           sdk.AccAddress
	balance          sdk.Coins
	minterModuleName string
}

func NewTestFaucet(t testing.TB, ctx sdk.Context, bankKeeper bankkeeper.Keeper, moveKeeper movekeeper.Keeper, minterModuleName string, initiaSupply ...sdk.Coin) *TestFaucet {
	require.NotEmpty(t, initiaSupply)
	r := &TestFaucet{t: t, bankKeeper: bankKeeper, moveKeeper: moveKeeper, minterModuleName: minterModuleName}
	_, _, addr := keyPubAddr()
	r.sender = addr
	r.Mint(ctx, addr, initiaSupply...)
	r.balance = initiaSupply
	return r
}

func (f *TestFaucet) Mint(parentCtx sdk.Context, addr sdk.AccAddress, amounts ...sdk.Coin) {
	amounts = sdk.Coins(amounts).Sort()
	require.NotEmpty(f.t, amounts)
	ctx := parentCtx.WithEventManager(sdk.NewEventManager()) // discard all faucet related events
	err := f.bankKeeper.MintCoins(ctx, f.minterModuleName, amounts)
	require.NoError(f.t, err)
	err = f.bankKeeper.SendCoinsFromModuleToAccount(ctx, f.minterModuleName, addr, amounts)
	require.NoError(f.t, err)
	f.balance = f.balance.Add(amounts...)
}

func (f *TestFaucet) Fund(parentCtx sdk.Context, receiver sdk.AccAddress, amounts ...sdk.Coin) {
	require.NotEmpty(f.t, amounts)
	// ensure faucet is always filled
	if !f.balance.IsAllGTE(amounts) {
		f.Mint(parentCtx, f.sender, amounts...)
	}
	ctx := parentCtx.WithEventManager(sdk.NewEventManager()) // discard all faucet related events
	err := f.bankKeeper.SendCoins(ctx, f.sender, receiver, amounts)
	require.NoError(f.t, err)
	f.balance = f.balance.Sub(amounts...)
}

func (f *TestFaucet) NewFundedAccount(ctx sdk.Context, amounts ...sdk.Coin) sdk.AccAddress {
	_, _, addr := keyPubAddr()
	f.Fund(ctx, addr, amounts...)
	return addr
}

type TestKeepers struct {
	AccountKeeper    authkeeper.AccountKeeper
	StakingKeeper    stakingkeeper.Keeper
	DistKeeper       distrkeeper.Keeper
	BankKeeper       bankkeeper.Keeper
	GovKeeper        govkeeper.Keeper
	MoveKeeper       movekeeper.Keeper
	OracleKeeper     oraclekeeper.Keeper
	DynamicFeeKeeper dynamicfeekeeper.Keeper
	EncodingConfig   initiaappparams.EncodingConfig
	Faucet           *TestFaucet
	MultiStore       storetypes.CommitMultiStore
}

// createDefaultTestInput common settings for createTestInput
func createDefaultTestInput(t testing.TB) (sdk.Context, TestKeepers) {
	return createTestInput(t, false)
}

// createTestInput encoders can be nil to accept the defaults, or set it to override some of the message handlers (like default)
func createTestInput(t testing.TB, isCheckTx bool) (sdk.Context, TestKeepers) {
	return _createTestInput(t, isCheckTx, moveconfig.DefaultMoveConfig(), dbm.NewMemDB())
}

var keyCounter uint64

// we need to make this deterministic (same every test run), as encoded address size and thus gas cost,
// depends on the actual bytes (due to ugly CanonicalAddress encoding)
func keyPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.AccAddress) {
	keyCounter++
	seed := make([]byte, 8)
	binary.BigEndian.PutUint64(seed, keyCounter)

	key := ed25519.GenPrivKeyFromSecret(seed)
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}

// encoders can be nil to accept the defaults, or set it to override some of the message handlers (like default)
func _createTestInput(
	t testing.TB,
	isCheckTx bool,
	moveConfig moveconfig.MoveConfig,
	db dbm.DB,
) (sdk.Context, TestKeepers) {
	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		rewardtypes.StoreKey, distributiontypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, authzkeeper.StoreKey, movetypes.StoreKey,
		oracletypes.StoreKey, dynamicfeetypes.StoreKey,
	)
	transientKeys := storetypes.NewTransientStoreKeys(dynamicfeetypes.TStoreKey)
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	for _, v := range keys {
		ms.MountStoreWithDB(v, storetypes.StoreTypeIAVL, db)
	}
	for _, v := range transientKeys {
		ms.MountStoreWithDB(v, storetypes.StoreTypeTransient, db)
	}
	memKeys := storetypes.NewMemoryStoreKeys()
	for _, v := range memKeys {
		ms.MountStoreWithDB(v, storetypes.StoreTypeMemory, db)
	}

	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, tmproto.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, isCheckTx, log.NewNopLogger())

	encodingConfig := MakeEncodingConfig(t)
	appCodec := encodingConfig.Codec

	moveKeeper := &movekeeper.Keeper{}
	maccPerms := map[string][]string{ // module account permissions
		authtypes.FeeCollectorName:      nil,
		distributiontypes.ModuleName:    nil,
		rewardtypes.ModuleName:          nil,
		stakingtypes.BondedPoolName:     {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName:  {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:             {authtypes.Burner},
		movetypes.MoveStakingModuleName: nil,

		// for testing
		authtypes.Minter: {authtypes.Minter, authtypes.Burner},
	}

	ac := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	vc := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	cc := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix())

	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]), // target store
		authtypes.ProtoBaseAccount,                          // prototype
		maccPerms,
		ac,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	blockedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	bankKeeper := movebank.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		accountKeeper,
		movekeeper.NewMoveBankKeeper(moveKeeper),
		blockedAddrs,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	require.NoError(t, bankKeeper.SetParams(ctx, banktypes.DefaultParams()))

	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		movekeeper.NewVotingPowerKeeper(moveKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		vc, cc,
	)
	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenoms = []string{bondDenom}
	require.NoError(t, stakingKeeper.SetParams(ctx, stakingParams))

	rewardKeeper := rewardkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[rewardtypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	rewardParams := rewardtypes.DefaultParams()
	rewardParams.RewardDenom = bondDenom
	require.NoError(t, rewardKeeper.SetParams(ctx, rewardParams))

	distKeeper := distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distributiontypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		movekeeper.NewDexKeeper(moveKeeper),
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	distrParams := customdistrtypes.DefaultParams()
	distrParams.RewardWeights = []customdistrtypes.RewardWeight{
		{Denom: bondDenom, Weight: math.LegacyOneDec()},
	}
	require.NoError(t, distKeeper.Params.Set(ctx, distrParams))
	stakingKeeper.SetHooks(distKeeper.Hooks())

	// set genesis items required for distribution
	require.NoError(t, distKeeper.FeePool.Set(ctx, distributiontypes.InitialFeePool()))

	accountKeeper.GetModuleAccount(ctx, movetypes.MoveStakingModuleName)

	oracleKeeper := oraclekeeper.NewKeeper(
		runtime.NewKVStoreService(keys[oracletypes.StoreKey]),
		appCodec,
		nil,
		authtypes.NewModuleAddress(govtypes.ModuleName),
	)

	queryRouter := baseapp.NewGRPCQueryRouter()
	queryRouter.SetInterfaceRegistry(encodingConfig.InterfaceRegistry)

	*moveKeeper = *movekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[movetypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		&oracleKeeper,
		TestMsgRouter{},
		queryRouter,
		moveConfig,
		distKeeper,
		stakingKeeper,
		rewardKeeper,
		distKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		ac, vc,
	)
	moveParams := movetypes.DefaultParams()
	moveParams.BaseDenom = bondDenom

	require.NoError(t, moveKeeper.SetRawParams(ctx, moveParams.ToRaw()))
	stakingKeeper.SetSlashingHooks(moveKeeper.Hooks())

	dynamicFeeKeeper := dynamicfeekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[dynamicfeetypes.StoreKey]),
		runtime.NewTransientStoreService(transientKeys[dynamicfeetypes.TStoreKey]),
		movekeeper.NewDexKeeper(moveKeeper),
		moveKeeper,
		moveKeeper,
		ac,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	require.NoError(t, dynamicFeeKeeper.SetParams(ctx, dynamicfeetypes.Params{
		MinBaseGasPrice: math.LegacyNewDecWithPrec(1, 3),
		MaxBaseGasPrice: math.LegacyNewDec(200),
		BaseGasPrice:    math.LegacyNewDecWithPrec(1, 2),
		TargetGas:       100000,
		MaxChangeRate:   math.LegacyNewDecWithPrec(1, 1),
	}))

	// load stdlib module bytes
	moduleBytes, err := precompile.ReadStdlib()
	require.NoError(t, err)

	// append test module
	moduleBytes = append(moduleBytes, basicCoinModule)

	err = moveKeeper.Initialize(ctx, moduleBytes, moveParams.AllowedPublishers, bondDenom)
	require.NoError(t, err)

	faucet := NewTestFaucet(t, ctx, bankKeeper, *moveKeeper, authtypes.Minter, initialTotalSupply()...)

	// set some funds to pay out validatores, based on code from:
	// https://github.com/cosmos/cosmos-sdk/blob/fea231556aee4d549d7551a6190389c4328194eb/x/distribution/keeper/keeper_test.go#L50-L57
	distrAcc := distKeeper.GetDistributionAccount(ctx)
	faucet.Fund(ctx, distrAcc.GetAddress(), sdk.NewCoin(bondDenom, math.NewInt(2000000)))
	accountKeeper.SetModuleAccount(ctx, distrAcc)

	// register bank & move
	msgRouter := baseapp.NewMsgServiceRouter()
	msgRouter.SetInterfaceRegistry(encodingConfig.InterfaceRegistry)

	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[govtypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		distKeeper,
		movekeeper.NewVestingKeeper(moveKeeper),
		msgRouter,
		govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	require.NoError(t, govKeeper.ProposalID.Set(ctx, govtypesv1.DefaultStartingProposalID))
	require.NoError(t, govKeeper.Params.Set(ctx, customgovtypes.DefaultParams()))

	cfg := sdk.GetConfig()
	cfg.SetAddressVerifier(initiaapp.VerifyAddressLen())

	am := module.NewManager( // minimal module set that we use for message/ query tests
		bank.NewAppModule(appCodec, bankKeeper, accountKeeper),
		staking.NewAppModule(appCodec, *stakingKeeper),
		distribution.NewAppModule(appCodec, *distKeeper),
		gov.NewAppModule(appCodec, govKeeper, accountKeeper, bankKeeper),
	)
	am.RegisterServices(module.NewConfigurator(appCodec, msgRouter, queryRouter)) //nolint:errcheck
	movetypes.RegisterMsgServer(msgRouter, movekeeper.NewMsgServerImpl(moveKeeper))
	movetypes.RegisterQueryServer(queryRouter, movekeeper.NewQuerier(moveKeeper))

	keepers := TestKeepers{
		AccountKeeper:    accountKeeper,
		StakingKeeper:    *stakingKeeper,
		DistKeeper:       *distKeeper,
		MoveKeeper:       *moveKeeper,
		BankKeeper:       bankKeeper,
		GovKeeper:        *govKeeper,
		OracleKeeper:     oracleKeeper,
		DynamicFeeKeeper: *dynamicFeeKeeper,
		// NftTransferKeeper: nftTransferKeeper,
		EncodingConfig: encodingConfig,
		Faucet:         faucet,
		MultiStore:     ms,
	}
	return ctx, keepers
}

var basicCoinModule []byte
var basicCoinModuleAbi string
var stdCoinTestModule []byte
var basicCoinMintScript []byte
var tableGeneratorModule []byte
var testAddressModule []byte
var vestingModule []byte
var submsgModule []byte

func init() {
	basicCoinModule = ReadMoveFile("BasicCoin")
	basicCoinModuleAbi = "{\"address\":\"0x1\",\"name\":\"BasicCoin\",\"friends\":[],\"exposed_functions\":[{\"name\":\"get\",\"visibility\":\"public\",\"is_entry\":false,\"is_view\":true,\"generic_type_params\":[{\"constraints\":[]}],\"params\":[\"address\"],\"return\":[\"u64\"]},{\"name\":\"get_coin\",\"visibility\":\"public\",\"is_entry\":false,\"is_view\":true,\"generic_type_params\":[{\"constraints\":[]}],\"params\":[\"address\"],\"return\":[\"0x1::BasicCoin::Coin<T0>\"]},{\"name\":\"mint\",\"visibility\":\"public\",\"is_entry\":true,\"is_view\":false,\"generic_type_params\":[{\"constraints\":[]}],\"params\":[\"signer\",\"u64\"],\"return\":[]},{\"name\":\"number\",\"visibility\":\"public\",\"is_entry\":false,\"is_view\":true,\"generic_type_params\":[],\"params\":[],\"return\":[\"u64\"]}],\"structs\":[{\"name\":\"Coin\",\"is_native\":false,\"abilities\":[\"copy\",\"key\"],\"generic_type_params\":[{\"constraints\":[],\"is_phantom\":true}],\"fields\":[{\"name\":\"value\",\"type\":\"u64\"},{\"name\":\"test\",\"type\":\"bool\"}]},{\"name\":\"Initia\",\"is_native\":false,\"abilities\":[],\"generic_type_params\":[],\"fields\":[{\"name\":\"dummy_field\",\"type\":\"bool\"}]},{\"name\":\"MintEvent\",\"is_native\":false,\"abilities\":[\"drop\",\"store\"],\"generic_type_params\":[],\"fields\":[{\"name\":\"account\",\"type\":\"address\"},{\"name\":\"amount\",\"type\":\"u64\"},{\"name\":\"coin_type\",\"type\":\"0x1::string::String\"}]},{\"name\":\"ViewEvent\",\"is_native\":false,\"abilities\":[\"drop\",\"store\"],\"generic_type_params\":[],\"fields\":[{\"name\":\"data\",\"type\":\"0x1::string::String\"}]}]}"
	stdCoinTestModule = ReadMoveFile("StdCoinTest")
	tableGeneratorModule = ReadMoveFile("TableGenerator")
	testAddressModule = ReadMoveFile("TestAddress")
	vestingModule = ReadMoveFile("Vesting")
	submsgModule = ReadMoveFile("submsg")

	basicCoinMintScript = ReadScriptFile("main")
}

func ReadMoveFile(filename string) []byte {
	path := "../../move/keeper/binaries/" + filename + ".mv"
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

func ReadScriptFile(filename string) []byte {
	path := "../../move/keeper/binaries/" + filename + ".mv"
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

type TestMsgRouter struct{}

func (router TestMsgRouter) Handler(msg sdk.Msg) baseapp.MsgServiceHandler {
	return router.HandlerByTypeURL(sdk.MsgTypeURL(msg))
}

func (router TestMsgRouter) HandlerByTypeURL(typeURL string) baseapp.MsgServiceHandler {
	switch typeURL {
	case sdk.MsgTypeURL(&movetypes.MsgExecute{}):
		return func(ctx sdk.Context, _msg sdk.Msg) (*sdk.Result, error) {
			msg := _msg.(*movetypes.MsgExecute)

			argStrs := []string{}
			for _, arg := range msg.Args {
				argStrs = append(argStrs, string(arg))
			}

			ctx.EventManager().EmitEvent(sdk.NewEvent("move_execute",
				sdk.NewAttribute("sender", msg.Sender),
				sdk.NewAttribute("module_addr", msg.ModuleAddress),
				sdk.NewAttribute("module_name", msg.ModuleName),
				sdk.NewAttribute("function_name", msg.FunctionName),
				sdk.NewAttribute("type_args", strings.Join(msg.TypeArgs, ",")),
				sdk.NewAttribute("args", strings.Join(argStrs, ",")),
			))

			return sdk.WrapServiceResult(ctx, &stakingtypes.MsgDelegateResponse{}, nil)
		}
	case sdk.MsgTypeURL(&movetypes.MsgExecuteJSON{}):
		return func(ctx sdk.Context, _msg sdk.Msg) (*sdk.Result, error) {
			msg := _msg.(*movetypes.MsgExecuteJSON)

			ctx.EventManager().EmitEvent(sdk.NewEvent("move_execute_with_json",
				sdk.NewAttribute("sender", msg.Sender),
				sdk.NewAttribute("module_addr", msg.ModuleAddress),
				sdk.NewAttribute("module_name", msg.ModuleName),
				sdk.NewAttribute("function_name", msg.FunctionName),
				sdk.NewAttribute("type_args", strings.Join(msg.TypeArgs, ",")),
				sdk.NewAttribute("args", strings.Join(msg.Args, ",")),
			))

			if msg.FunctionName == "fail" {
				return nil, fmt.Errorf("fail")
			}

			return sdk.WrapServiceResult(ctx, &stakingtypes.MsgDelegateResponse{}, nil)
		}
	case sdk.MsgTypeURL(&movetypes.MsgScript{}):
		return func(ctx sdk.Context, _msg sdk.Msg) (*sdk.Result, error) {
			msg := _msg.(*movetypes.MsgScript)

			argStrs := []string{}
			for _, arg := range msg.Args {
				argStrs = append(argStrs, string(arg))
			}

			ctx.EventManager().EmitEvent(sdk.NewEvent("move_script",
				sdk.NewAttribute("sender", msg.Sender),
				sdk.NewAttribute("code_bytes", hex.EncodeToString(msg.CodeBytes)),
				sdk.NewAttribute("type_args", strings.Join(msg.TypeArgs, ",")),
				sdk.NewAttribute("args", strings.Join(argStrs, ",")),
			))

			return sdk.WrapServiceResult(ctx, &stakingtypes.MsgDelegateResponse{}, nil)
		}
	case sdk.MsgTypeURL(&movetypes.MsgScriptJSON{}):
		return func(ctx sdk.Context, _msg sdk.Msg) (*sdk.Result, error) {
			msg := _msg.(*movetypes.MsgScriptJSON)

			ctx.EventManager().EmitEvent(sdk.NewEvent("move_script_with_json",
				sdk.NewAttribute("sender", msg.Sender),
				sdk.NewAttribute("code_bytes", hex.EncodeToString(msg.CodeBytes)),
				sdk.NewAttribute("type_args", strings.Join(msg.TypeArgs, ",")),
				sdk.NewAttribute("args", strings.Join(msg.Args, ",")),
			))

			return sdk.WrapServiceResult(ctx, &stakingtypes.MsgDelegateResponse{}, nil)
		}
	case sdk.MsgTypeURL(&stakingtypes.MsgDelegate{}):
		return func(ctx sdk.Context, _msg sdk.Msg) (*sdk.Result, error) {
			msg := _msg.(*stakingtypes.MsgDelegate)
			ctx.EventManager().EmitEvent(sdk.NewEvent("delegate",
				sdk.NewAttribute("delegator_address", msg.DelegatorAddress),
				sdk.NewAttribute("validator_address", msg.ValidatorAddress),
				sdk.NewAttribute("amount", msg.Amount.String()),
			))

			return sdk.WrapServiceResult(ctx, &stakingtypes.MsgDelegateResponse{}, nil)
		}
	case sdk.MsgTypeURL(&distributiontypes.MsgFundCommunityPool{}):
		return func(ctx sdk.Context, _msg sdk.Msg) (*sdk.Result, error) {
			msg := _msg.(*distributiontypes.MsgFundCommunityPool)
			ctx.EventManager().EmitEvent(sdk.NewEvent("fund_community_pool",
				sdk.NewAttribute("depositor_address", msg.Depositor),
				sdk.NewAttribute("amount", msg.Amount.String()),
			))

			return sdk.WrapServiceResult(ctx, &stakingtypes.MsgDelegateResponse{}, nil)
		}
	case sdk.MsgTypeURL(&transfertypes.MsgTransfer{}):
		return func(ctx sdk.Context, _msg sdk.Msg) (*sdk.Result, error) {
			msg := _msg.(*transfertypes.MsgTransfer)
			ctx.EventManager().EmitEvent(sdk.NewEvent("transfer",
				sdk.NewAttribute("sender", msg.Sender),
				sdk.NewAttribute("receiver", msg.Receiver),
				sdk.NewAttribute("token", msg.Token.String()),
				sdk.NewAttribute("source_port", msg.SourcePort),
				sdk.NewAttribute("source_channel", msg.SourceChannel),
				sdk.NewAttribute("timeout_height", msg.TimeoutHeight.String()),
				sdk.NewAttribute("timeout_timestamp", fmt.Sprint(msg.TimeoutTimestamp)),
				sdk.NewAttribute("memo", msg.Memo),
			))

			return sdk.WrapServiceResult(ctx, &stakingtypes.MsgDelegateResponse{}, nil)
		}
	}

	panic("handler not registered")
}

func MustConvertStringToTypeTag(str string) vmtypes.TypeTag {
	tt, err := vmapi.TypeTagFromString(str)
	if err != nil {
		panic(err)
	}

	return tt
}
