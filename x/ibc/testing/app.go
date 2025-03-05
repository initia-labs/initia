package ibctesting

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	abci "github.com/cometbft/cometbft/abci/types"

	tmtypes "github.com/cometbft/cometbft/types"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	testutilsims "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"

	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v8/modules/core/keeper"

	initiaapp "github.com/initia-labs/initia/v1/app"
	ibctestingtypes "github.com/initia-labs/initia/v1/x/ibc/testing/types"
	icaauthkeeper "github.com/initia-labs/initia/v1/x/intertx/keeper"
	moveconfig "github.com/initia-labs/initia/v1/x/move/config"
	stakingtypes "github.com/initia-labs/initia/v1/x/mstaking/types"

	oracleconfig "github.com/skip-mev/connect/v2/oracle/config"
)

func coins(amt int64) sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(amt)))
}

func decCoins(amt int64) sdk.DecCoins {
	return sdk.NewDecCoins(sdk.NewDecCoin(sdk.DefaultBondDenom, math.NewInt(amt)))
}

var DefaultTestingAppInit = SetupTestingApp

type TestingApp interface {
	servertypes.ABCI

	// ibc-go additions
	GetBaseApp() *baseapp.BaseApp
	GetAccountKeeper() *authkeeper.AccountKeeper
	GetStakingKeeper() ibctestingtypes.StakingKeeper
	GetIBCKeeper() *keeper.Keeper
	GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper
	GetICAControllerKeeper() *icacontrollerkeeper.Keeper
	GetICAAuthKeeper() *icaauthkeeper.Keeper
	TxConfig() client.TxConfig

	// Implemented by SimApp
	AppCodec() codec.Codec

	// Implemented by BaseApp
	LastCommitID() storetypes.CommitID
	LastBlockHeight() int64
}

func SetupTestingApp(t *testing.T) (TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	encCdc := initiaapp.MakeEncodingConfig()
	app := initiaapp.NewInitiaApp(log.NewNopLogger(), db, nil, true, moveconfig.DefaultMoveConfig(), oracleconfig.NewDefaultAppConfig(), testutilsims.EmptyAppOptions{})
	return app, initiaapp.NewDefaultGenesisState(encCdc.Codec, sdk.DefaultBondDenom)
}

// SetupWithGenesisValSet initializes a new SimApp with a validator set and genesis accounts
// that also act as delegators. For simplicity, each validator is bonded with a delegation
// of one consensus engine unit (10^6) in the default token of the simapp from first genesis
// account. A Nop logger is set in SimApp.
func SetupWithGenesisValSet(t *testing.T, valSet *tmtypes.ValidatorSet, genAccs []authtypes.GenesisAccount, chainID string, powerReduction math.Int, balances ...banktypes.Balance) TestingApp {
	app, genesisState := DefaultTestingAppInit(t)

	// ensure baseapp has a chain-id set before running InitChain
	baseapp.SetChainID(chainID)(app.GetBaseApp())

	// set genesis accounts
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)

	validators := make([]stakingtypes.Validator, 0, len(valSet.Validators))
	delegations := make([]stakingtypes.Delegation, 0, len(valSet.Validators))

	bondAmt := sdk.TokensFromConsensusPower(1, powerReduction)

	for _, val := range valSet.Validators {
		pk, err := cryptocodec.FromCmtPubKeyInterface(val.PubKey)
		require.NoError(t, err)
		pkAny, err := codectypes.NewAnyWithValue(pk)
		require.NoError(t, err)
		validator := stakingtypes.Validator{
			OperatorAddress: sdk.ValAddress(val.Address).String(),
			ConsensusPubkey: pkAny,
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          coins(1_000_000),
			DelegatorShares: decCoins(1_000_000),
			Description:     stakingtypes.NewDescription("homeDir", "", "", "", ""),
			UnbondingHeight: int64(0),
			UnbondingTime:   time.Unix(0, 0).UTC(),
			Commission:      stakingtypes.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
		}

		validators = append(validators, validator)
		delegations = append(delegations, stakingtypes.NewDelegation(genAccs[0].GetAddress().String(), validator.GetOperator(), decCoins(1_000_000)))
	}

	// set validators and delegations
	var stakingGenesis stakingtypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesisState[stakingtypes.ModuleName], &stakingGenesis)

	bondDenom := stakingGenesis.Params.BondDenoms[0]

	// add bonded amount to bonded pool module account
	balances = append(balances, banktypes.Balance{
		Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
		Coins:   sdk.Coins{sdk.NewCoin(bondDenom, bondAmt.Mul(math.NewInt(int64(len(valSet.Validators)))))},
	})

	// set validators and delegations
	stakingGenesis = *stakingtypes.NewGenesisState(stakingGenesis.Params, validators, delegations)
	genesisState[stakingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(&stakingGenesis)

	// update total supply
	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, sdk.NewCoins(), []banktypes.Metadata{}, []banktypes.SendEnabled{})
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	require.NoError(t, err)

	// init chain will set the validator set and initialize the genesis accounts
	_, err = app.InitChain(
		&abci.RequestInitChain{
			ChainId:         chainID,
			Validators:      []abci.ValidatorUpdate{},
			ConsensusParams: testutilsims.DefaultConsensusParams,
			AppStateBytes:   stateBytes,
		},
	)
	require.NoError(t, err)

	return app
}
