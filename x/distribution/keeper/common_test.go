package keeper_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/upgrade"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codecaddress "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
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
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/gogoproto/proto"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"

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
	"github.com/initia-labs/movevm/precompile"
)

var ModuleBasics = module.NewBasicManager(
	auth.AppModuleBasic{},
	bank.AppModuleBasic{},
	staking.AppModuleBasic{},
	reward.AppModuleBasic{},
	distribution.AppModuleBasic{},
	gov.AppModuleBasic{},
	crisis.AppModuleBasic{},
	slashing.AppModuleBasic{},
	upgrade.AppModuleBasic{},
	evidence.AppModuleBasic{},
	move.AppModuleBasic{},
)

// Bond denom should be set for staking test
const bondDenom = initiaapp.BondDenom

var (
	initiaSupply = math.NewInt(100_000_000_000)
	testDenoms   = []string{
		"test1",
		"test2",
		"test3",
		"test4",
		"test5",
	}
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
	AccountKeeper  authkeeper.AccountKeeper
	StakingKeeper  stakingkeeper.Keeper
	DistKeeper     distrkeeper.Keeper
	BankKeeper     bankkeeper.Keeper
	GovKeeper      govkeeper.Keeper
	MoveKeeper     movekeeper.Keeper
	DexKeeper      TestDexKeeper
	EncodingConfig initiaappparams.EncodingConfig
	Faucet         *TestFaucet
	MultiStore     storetypes.CommitMultiStore
}

// createDefaultTestInput common settings for createTestInput
func createDefaultTestInput(t testing.TB) (sdk.Context, TestKeepers) {
	return createTestInput(t, false)
}

// createTestInput encoders can be nil to accept the defaults, or set it to override some of the message handlers (like default)
func createTestInput(t testing.TB, isCheckTx bool) (sdk.Context, TestKeepers) {
	// Load default move config
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
	)
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	for _, v := range keys {
		ms.MountStoreWithDB(v, storetypes.StoreTypeIAVL, db)
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

	dexKeeper := NewTestDexKeeper(moveKeeper)
	distKeeper := distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distributiontypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		dexKeeper,
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

	*moveKeeper = *movekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[movetypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		nil,
		nil,
		nil,
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

	// load stdlib module bytes
	moduleBytes, err := precompile.ReadStdlib()
	require.NoError(t, err)

	err = moveKeeper.Initialize(ctx, moduleBytes, moveParams.AllowedPublishers)
	require.NoError(t, err)

	faucet := NewTestFaucet(t, ctx, bankKeeper, *moveKeeper, authtypes.Minter, initialTotalSupply()...)

	// register bank & move
	msgRouter := baseapp.NewMsgServiceRouter()
	msgRouter.SetInterfaceRegistry(encodingConfig.InterfaceRegistry)
	banktypes.RegisterMsgServer(msgRouter, bankkeeper.NewMsgServerImpl(bankKeeper))
	movetypes.RegisterMsgServer(msgRouter, movekeeper.NewMsgServerImpl(moveKeeper))

	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[govtypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		distKeeper,
		msgRouter,
		govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	require.NoError(t, govKeeper.ProposalID.Set(ctx, govtypesv1.DefaultStartingProposalID))
	require.NoError(t, govKeeper.Params.Set(ctx, customgovtypes.DefaultParams()))

	cfg := sdk.GetConfig()
	cfg.SetAddressVerifier(initiaapp.VerifyAddressLen())

	keepers := TestKeepers{
		AccountKeeper:  accountKeeper,
		StakingKeeper:  *stakingKeeper,
		DistKeeper:     *distKeeper,
		MoveKeeper:     *moveKeeper,
		BankKeeper:     bankKeeper,
		GovKeeper:      *govKeeper,
		DexKeeper:      dexKeeper,
		EncodingConfig: encodingConfig,
		Faucet:         faucet,
		MultiStore:     ms,
	}
	return ctx, keepers
}

func createValidatorWithBalance(
	ctx sdk.Context,
	input TestKeepers,
	balance int64,
	delBalance int64,
	index int,
) sdk.ValAddress {
	valPubKey := testutilsims.CreateTestPubKeys(index)[index-1]

	pubKey := secp256k1.GenPrivKey().PubKey()
	accAddr := sdk.AccAddress(sdk.AccAddress(pubKey.Address()))
	valAddr := sdk.ValAddress(sdk.AccAddress(pubKey.Address()))

	input.Faucet.Fund(ctx, accAddr, sdk.NewCoin(bondDenom, math.NewInt(balance)))

	sh := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err := sh.CreateValidator(ctx, newTestMsgCreateValidator(valAddr, valPubKey, sdk.NewCoin(bondDenom, math.NewInt(delBalance))))
	if err != nil {
		panic(err)
	}

	// power update
	_, err = input.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	if err != nil {
		panic(err)
	}

	return valAddr
}

func createValidatorWithCoin(
	ctx sdk.Context,
	input TestKeepers,
	balance sdk.Coins,
	delBalance sdk.Coins,
	index int,
) sdk.ValAddress {
	valPubKey := testutilsims.CreateTestPubKeys(index)[index-1]

	pubKey := secp256k1.GenPrivKey().PubKey()
	accAddr := sdk.AccAddress(sdk.AccAddress(pubKey.Address()))
	valAddr := sdk.ValAddress(sdk.AccAddress(pubKey.Address()))

	input.Faucet.Fund(ctx, accAddr, balance...)

	sh := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err := sh.CreateValidator(ctx, newTestMsgCreateValidator(valAddr, valPubKey, delBalance...))
	if err != nil {
		panic(err)
	}

	// power update
	_, err = input.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	if err != nil {
		panic(err)
	}

	return valAddr
}

// newTestMsgCreateValidator test msg creator
func newTestMsgCreateValidator(address sdk.ValAddress, pubKey cryptotypes.PubKey, amt ...sdk.Coin) *stakingtypes.MsgCreateValidator {
	commission := stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDec(0))
	msg, _ := stakingtypes.NewMsgCreateValidator(
		address.String(), pubKey, amt,
		stakingtypes.NewDescription("homeDir", "", "", "", ""), commission,
	)
	return msg
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
	case *transfertypes.MsgTransfer:
		return func(ctx sdk.Context, _req sdk.Msg) (*sdk.Result, error) {
			ctx.EventManager().EmitEvent(sdk.NewEvent("transfer",
				sdk.NewAttribute("sender", msg.Sender),
				sdk.NewAttribute("receiver", msg.Receiver),
				sdk.NewAttribute("source_port", msg.SourcePort),
				sdk.NewAttribute("source_channel", msg.SourceChannel),
				sdk.NewAttribute("token", msg.Token.String()),
				sdk.NewAttribute("timeout_height", msg.TimeoutHeight.String()),
				sdk.NewAttribute("timeout_timestamp", fmt.Sprint(msg.TimeoutTimestamp)),
				sdk.NewAttribute("memo", msg.Memo),
			))

			return sdk.WrapServiceResult(ctx, &stakingtypes.MsgDelegateResponse{}, nil)
		}
	}

	panic("handler not registered")
}

// test Keeper

type TestDexKeeper struct {
	prices     map[string]math.LegacyDec
	moveKeeper *movekeeper.Keeper
}

func NewTestDexKeeper(moveKeeper *movekeeper.Keeper) TestDexKeeper {
	return TestDexKeeper{
		prices:     make(map[string]math.LegacyDec),
		moveKeeper: moveKeeper,
	}
}

func (k TestDexKeeper) SetPrice(denom string, price math.LegacyDec) {
	k.prices[denom] = price
}

func (k TestDexKeeper) SwapToBase(ctx context.Context, addr sdk.AccAddress, quoteCoin sdk.Coin) error {
	price, ok := k.prices[quoteCoin.Denom]
	if !ok {
		return nil
	}

	bk := movekeeper.NewMoveBankKeeper(k.moveKeeper)
	baseDenom, err := k.moveKeeper.BaseDenom(ctx)
	if err != nil {
		return err
	}

	if err := bk.MintCoins(
		ctx, addr,
		sdk.NewCoins(sdk.NewCoin(
			baseDenom,
			price.MulInt(quoteCoin.Amount).TruncateInt(),
		)),
	); err != nil {
		return err
	}

	// withdraw coin
	_, _, dummyAddr := keyPubAddr()
	return bk.SendCoin(ctx, addr, dummyAddr, quoteCoin.Denom, quoteCoin.Amount)
}
