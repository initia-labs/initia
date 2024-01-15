package app

// DONTCOVER

import (
	"encoding/json"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"

	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	apporacle "github.com/initia-labs/initia/app/oracle"
	moveconfig "github.com/initia-labs/initia/x/move/config"
)

// defaultConsensusParams defines the default Tendermint consensus params used in
// InitiaApp testing.
var defaultConsensusParams = &tmproto.ConsensusParams{
	Block: &tmproto.BlockParams{
		MaxBytes: 8000000,
		MaxGas:   1234000000,
	},
	Evidence: &tmproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &tmproto.ValidatorParams{
		PubKeyTypes: []string{
			tmtypes.ABCIPubKeyTypeEd25519,
		},
	},
}

func getOrCreateMemDB(db *dbm.DB) dbm.DB {
	if db != nil {
		return *db
	}
	return dbm.NewMemDB()
}

func setup(db *dbm.DB, withGenesis bool) (*InitiaApp, GenesisState) {
	app := NewInitiaApp(
		log.NewNopLogger(),
		getOrCreateMemDB(db),
		nil,
		true,
		moveconfig.DefaultMoveConfig(),
		apporacle.DefaultConfig(),
		simtestutil.EmptyAppOptions{},
	)

	if withGenesis {
		return app, NewDefaultGenesisState(app.appCodec).
			ConfigureBondDenom(app.appCodec, BondDenom)
	}

	return app, GenesisState{}
}

// SetupWithGenesisAccounts setup initiaapp with genesis account
func SetupWithGenesisAccounts(
	valSet *tmtypes.ValidatorSet,
	genAccs []authtypes.GenesisAccount,
	balances ...banktypes.Balance,
) *InitiaApp {
	app, genesisState := setup(nil, true)

	if len(genAccs) == 0 {
		privAcc := secp256k1.GenPrivKey()
		genAccs = []authtypes.GenesisAccount{
			authtypes.NewBaseAccount(privAcc.PubKey().Address().Bytes(), privAcc.PubKey(), 0, 0),
		}
	}

	// set genesis accounts
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)

	// allow empty validator
	if valSet == nil || len(valSet.Validators) == 0 {
		privVal := ed25519.GenPrivKey()
		pubKey, err := cryptocodec.ToCmtPubKeyInterface(privVal.PubKey())
		if err != nil {
			panic(err)
		}

		validator := tmtypes.NewValidator(pubKey, 1)
		valSet = tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})
	}

	validators := make([]stakingtypes.Validator, 0, len(valSet.Validators))
	delegations := make([]stakingtypes.Delegation, 0, len(valSet.Validators))

	bondAmt := sdk.TokensFromConsensusPower(1, sdk.DefaultPowerReduction)
	bondCoins := sdk.NewCoins(sdk.NewCoin(BondDenom, bondAmt))

	for i, val := range valSet.Validators {
		pk, err := cryptocodec.FromCmtPubKeyInterface(val.PubKey)
		if err != nil {
			panic(err)
		}
		pkAny, err := codectypes.NewAnyWithValue(pk)
		if err != nil {
			panic(err)
		}

		validator := stakingtypes.Validator{
			OperatorAddress: sdk.ValAddress(val.Address).String(),
			ConsensusPubkey: pkAny,
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          bondCoins,
			DelegatorShares: sdk.NewDecCoins(sdk.NewDecCoinFromDec(BondDenom, math.LegacyOneDec())),
			Description:     stakingtypes.NewDescription("homeDir", "", "", "", ""),
			UnbondingHeight: int64(0),
			UnbondingTime:   time.Unix(0, 0).UTC(),
			Commission:      stakingtypes.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
		}

		validators = append(validators, validator)
		delegations = append(delegations, stakingtypes.NewDelegation(genAccs[i].GetAddress().String(), sdk.ValAddress(val.Address).String(), sdk.NewDecCoins(sdk.NewDecCoinFromDec(BondDenom, math.LegacyOneDec()))))
	}

	// set validators and delegations
	var stakingGenesis stakingtypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesisState[stakingtypes.ModuleName], &stakingGenesis)

	// add bonded amount to bonded pool module account
	balances = append(balances, banktypes.Balance{
		Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
		Coins:   sdk.Coins{sdk.NewCoin(BondDenom, bondAmt.Mul(math.NewInt(int64(len(valSet.Validators)))))},
	})

	// set validators and delegations
	stakingGenesis = *stakingtypes.NewGenesisState(stakingGenesis.Params, validators, delegations)
	genesisState[stakingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(&stakingGenesis)

	// update total supply
	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, sdk.NewCoins(), []banktypes.Metadata{}, []banktypes.SendEnabled{})
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	if err != nil {
		panic(err)
	}

	_, err = app.InitChain(
		&abci.RequestInitChain{
			Validators:      []abci.ValidatorUpdate{},
			ConsensusParams: defaultConsensusParams,
			AppStateBytes:   stateBytes,
		},
	)
	if err != nil {
		panic(err)
	}

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1})
	if err != nil {
		panic(err)
	}

	_, err = app.Commit()
	if err != nil {
		panic(err)
	}

	return app
}
