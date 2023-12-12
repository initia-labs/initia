package keeper_test

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	testutilsims "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	initiaapp "github.com/initia-labs/initia/app"
	initiaappparams "github.com/initia-labs/initia/app/params"
	movebank "github.com/initia-labs/initia/x/bank/keeper"
	"github.com/initia-labs/initia/x/distribution"
	distrkeeper "github.com/initia-labs/initia/x/distribution/keeper"
	customdistrtypes "github.com/initia-labs/initia/x/distribution/types"
	"github.com/initia-labs/initia/x/evidence"
	"github.com/initia-labs/initia/x/gov"
	govkeeper "github.com/initia-labs/initia/x/gov/keeper"
	customgovtypes "github.com/initia-labs/initia/x/gov/types"

	// nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
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

	vmapi "github.com/initia-labs/initiavm/api"
	"github.com/initia-labs/initiavm/precompile"
	vmtypes "github.com/initia-labs/initiavm/types"
)

var ModuleBasics = module.NewBasicManager(
	auth.AppModuleBasic{},
	bank.AppModuleBasic{},
	capability.AppModuleBasic{},
	staking.AppModuleBasic{},
	reward.AppModuleBasic{},
	distribution.AppModuleBasic{},
	gov.NewAppModuleBasic(),
	crisis.AppModuleBasic{},
	slashing.AppModuleBasic{},
	upgrade.AppModuleBasic{},
	evidence.AppModuleBasic{},
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

	initiaSupply = sdk.NewInt(100_000_000_000)
)

func MakeTestCodec(t testing.TB) codec.Codec {
	return MakeEncodingConfig(t).Marshaler
}

func MakeEncodingConfig(_ testing.TB) initiaappparams.EncodingConfig {
	encodingConfig := initiaappparams.MakeEncodingConfig()
	amino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	std.RegisterInterfaces(interfaceRegistry)
	std.RegisterLegacyAminoCodec(amino)

	ModuleBasics.RegisterLegacyAminoCodec(amino)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	// add initiad types
	movetypes.RegisterInterfaces(interfaceRegistry)
	movetypes.RegisterLegacyAminoCodec(amino)

	return encodingConfig
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
	AccountKeeper authkeeper.AccountKeeper
	StakingKeeper stakingkeeper.Keeper
	DistKeeper    distrkeeper.Keeper
	BankKeeper    bankkeeper.Keeper
	GovKeeper     govkeeper.Keeper
	MoveKeeper    movekeeper.Keeper
	// NftTransferKeeper TestIBCNftTransferKeeper
	EncodingConfig initiaappparams.EncodingConfig
	Faucet         *TestFaucet
	MultiStore     sdk.CommitMultiStore
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
	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		rewardtypes.StoreKey, distributiontypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, upgradetypes.StoreKey, evidencetypes.StoreKey,
		capabilitytypes.StoreKey, feegrant.StoreKey, authzkeeper.StoreKey,
		movetypes.StoreKey,
	)
	ms := store.NewCommitMultiStore(db)
	for _, v := range keys {
		ms.MountStoreWithDB(v, storetypes.StoreTypeIAVL, db)
	}
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)
	for _, v := range memKeys {
		ms.MountStoreWithDB(v, storetypes.StoreTypeMemory, db)
	}

	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, tmproto.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, isCheckTx, log.NewNopLogger())

	encodingConfig := MakeEncodingConfig(t)
	appCodec := encodingConfig.Marshaler

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
	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		keys[authtypes.StoreKey],   // target store
		authtypes.ProtoBaseAccount, // prototype
		maccPerms,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	blockedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	bankKeeper := movebank.NewBaseKeeper(
		appCodec,
		keys[banktypes.StoreKey],
		accountKeeper,
		movekeeper.NewMoveBankKeeper(moveKeeper),
		blockedAddrs,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	bankKeeper.SetParams(ctx, banktypes.DefaultParams())

	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec,
		keys[stakingtypes.StoreKey],
		accountKeeper,
		bankKeeper,
		movekeeper.NewVotingPowerKeeper(moveKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenoms = []string{bondDenom}
	stakingKeeper.SetParams(ctx, stakingParams)

	rewardKeeper := rewardkeeper.NewKeeper(
		appCodec,
		keys[rewardtypes.StoreKey],
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	rewardParams := rewardtypes.DefaultParams()
	rewardParams.RewardDenom = bondDenom
	rewardKeeper.SetParams(ctx, rewardParams)

	distKeeper := distrkeeper.NewKeeper(
		appCodec,
		keys[distributiontypes.StoreKey],
		accountKeeper,
		bankKeeper,
		&stakingKeeper,
		movekeeper.NewDexKeeper(moveKeeper),
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	distrParams := customdistrtypes.DefaultParams()
	distrParams.RewardWeights = []customdistrtypes.RewardWeight{
		{Denom: bondDenom, Weight: sdk.OneDec()},
	}
	distKeeper.SetParams(ctx, distrParams)
	stakingKeeper.SetHooks(distKeeper.Hooks())

	// set genesis items required for distribution
	distKeeper.SetFeePool(ctx, distributiontypes.InitialFeePool())

	accountKeeper.GetModuleAccount(ctx, movetypes.MoveStakingModuleName)

	// nftTransferKeeper := TestIBCNftTransferKeeper{
	// 	classTraces: make(map[string]string),
	// }

	*moveKeeper = movekeeper.NewKeeper(
		appCodec,
		keys[movetypes.StoreKey],
		accountKeeper,
		distKeeper,
		// nftTransferKeeper,
		TestMsgRouter{},
		moveConfig,
		bankKeeper,
		distKeeper,
		&stakingKeeper,
		rewardKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	moveParams := movetypes.DefaultParams()
	moveParams.BaseDenom = bondDenom

	moveKeeper.SetRawParams(ctx, moveParams.ToRaw())
	stakingKeeper.SetSlashingHooks(moveKeeper.Hooks())

	// load stdlib module bytes
	moduleBytes, err := precompile.ReadStdlib()
	require.NoError(t, err)

	// append test module
	moduleBytes = append(moduleBytes, basicCoinModule)
	err = moveKeeper.Initialize(ctx, moduleBytes, moveParams.ArbitraryEnabled, moveParams.AllowedPublishers)
	require.NoError(t, err)

	faucet := NewTestFaucet(t, ctx, bankKeeper, *moveKeeper, authtypes.Minter, initialTotalSupply()...)

	// set some funds to pay out validatores, based on code from:
	// https://github.com/cosmos/cosmos-sdk/blob/fea231556aee4d549d7551a6190389c4328194eb/x/distribution/keeper/keeper_test.go#L50-L57
	distrAcc := distKeeper.GetDistributionAccount(ctx)
	faucet.Fund(ctx, distrAcc.GetAddress(), sdk.NewCoin(bondDenom, sdk.NewInt(2000000)))
	accountKeeper.SetModuleAccount(ctx, distrAcc)

	// register bank & move
	msgRouter := baseapp.NewMsgServiceRouter()
	msgRouter.SetInterfaceRegistry(encodingConfig.InterfaceRegistry)
	banktypes.RegisterMsgServer(msgRouter, bankkeeper.NewMsgServerImpl(bankKeeper))
	movetypes.RegisterMsgServer(msgRouter, movekeeper.NewMsgServerImpl(*moveKeeper))

	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		keys[govtypes.StoreKey],
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		msgRouter,
		govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	govKeeper.SetProposalID(ctx, govtypesv1.DefaultStartingProposalID)
	govKeeper.SetParams(ctx, customgovtypes.DefaultParams())

	cfg := sdk.GetConfig()
	cfg.SetAddressVerifier(initiaapp.VerifyAddressLen())

	keepers := TestKeepers{
		AccountKeeper: accountKeeper,
		StakingKeeper: stakingKeeper,
		DistKeeper:    distKeeper,
		MoveKeeper:    *moveKeeper,
		BankKeeper:    bankKeeper,
		GovKeeper:     govKeeper,
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

func init() {
	basicCoinModule = ReadMoveFile("BasicCoin")
	basicCoinModuleAbi = `{"address":"0x1","name":"BasicCoin","friends":[],"exposed_functions":[{"name":"get","visibility":"public","is_entry":false,"is_view":true,"generic_type_params":[{"constraints":[]}],"params":["address"],"return":["u64"]},{"name":"get_coin","visibility":"public","is_entry":false,"is_view":true,"generic_type_params":[{"constraints":[]}],"params":["address"],"return":["0x1::BasicCoin::Coin<T0>"]},{"name":"mint","visibility":"public","is_entry":true,"is_view":false,"generic_type_params":[{"constraints":[]}],"params":["signer","u64"],"return":[]},{"name":"number","visibility":"public","is_entry":false,"is_view":true,"generic_type_params":[],"params":[],"return":["u64"]}],"structs":[{"name":"Coin","is_native":false,"abilities":["copy","key"],"generic_type_params":[{"constraints":[],"is_phantom":true}],"fields":[{"name":"value","type":"u64"},{"name":"test","type":"bool"}]},{"name":"Initia","is_native":false,"abilities":[],"generic_type_params":[],"fields":[{"name":"dummy_field","type":"bool"}]},{"name":"MintEvent","is_native":false,"abilities":["drop","store"],"generic_type_params":[],"fields":[{"name":"account","type":"address"},{"name":"amount","type":"u64"},{"name":"coin_type","type":"0x1::string::String"}]}]}`
	stdCoinTestModule = ReadMoveFile("StdCoinTest")
	tableGeneratorModule = ReadMoveFile("TableGenerator")

	basicCoinMintScript = ReadScriptFile("main")
}

func ReadMoveFile(filename string) []byte {
	path := "./binaries/" + filename + ".mv"
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

func ReadScriptFile(filename string) []byte {
	path := "./binaries/" + filename + ".mv"
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

type TestMsgRouter struct{}

func (router TestMsgRouter) Handler(msg sdk.Msg) baseapp.MsgServiceHandler {
	switch msg := msg.(type) {
	case *stakingtypes.MsgDelegate:
		return func(ctx sdk.Context, _req sdk.Msg) (*sdk.Result, error) {
			ctx.EventManager().EmitEvent(sdk.NewEvent("delegate",
				sdk.NewAttribute("delegator_address", msg.DelegatorAddress),
				sdk.NewAttribute("validator_address", msg.ValidatorAddress),
				sdk.NewAttribute("amount", msg.Amount.String()),
			))

			return sdk.WrapServiceResult(ctx, &stakingtypes.MsgDelegateResponse{}, nil)
		}
	case *distributiontypes.MsgFundCommunityPool:
		return func(ctx sdk.Context, _req sdk.Msg) (*sdk.Result, error) {
			ctx.EventManager().EmitEvent(sdk.NewEvent("fund_community_pool",
				sdk.NewAttribute("depositor_address", msg.Depositor),
				sdk.NewAttribute("amount", msg.Amount.String()),
			))

			return sdk.WrapServiceResult(ctx, &stakingtypes.MsgDelegateResponse{}, nil)
		}
	case *transfertypes.MsgTransfer:
		return func(ctx sdk.Context, _req sdk.Msg) (*sdk.Result, error) {
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
	case *ibcfeetypes.MsgPayPacketFee:
		return func(ctx sdk.Context, _req sdk.Msg) (*sdk.Result, error) {
			ctx.EventManager().EmitEvent(sdk.NewEvent("pay_fee",
				sdk.NewAttribute("signer", msg.Signer),
				sdk.NewAttribute("source_port", msg.SourcePortId),
				sdk.NewAttribute("source_channel", msg.SourceChannelId),
				sdk.NewAttribute("recv_fee", msg.Fee.RecvFee.String()),
				sdk.NewAttribute("ack_fee", msg.Fee.AckFee.String()),
				sdk.NewAttribute("timeout_fee", msg.Fee.TimeoutFee.String()),
				sdk.NewAttribute("relayers", strings.Join(msg.Relayers, ",")),
			))

			return sdk.WrapServiceResult(ctx, &stakingtypes.MsgDelegateResponse{}, nil)
		}

	}

	panic("handler not registered")
}

// test Keepers

// type TestIBCNftTransferKeeper struct {
// 	classTraces map[string]string
// }

// func (k TestIBCNftTransferKeeper) GetClassTrace(ctx sdk.Context, classTraceHash tmbytes.HexBytes) (nfttransfertypes.ClassTrace, bool) {
// 	trace, found := k.classTraces[classTraceHash.String()]
// 	if !found {
// 		return nfttransfertypes.ClassTrace{}, false
// 	}

// 	return nfttransfertypes.ClassTrace{
// 		Path:        "",
// 		BaseClassId: trace,
// 	}, true
// }

// func (k TestIBCNftTransferKeeper) SetClassTrace(ctx sdk.Context, classTrace nfttransfertypes.ClassTrace) {
// 	k.classTraces[classTrace.Hash().String()] = classTrace.BaseClassId
// }

func MustConvertStringToTypeTag(str string) vmtypes.TypeTag {
	tt, err := vmapi.TypeTagFromString(str)
	if err != nil {
		panic(err)
	}

	return tt
}
