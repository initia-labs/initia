package gov_test

import (
	"testing"

	initiaapp "github.com/initia-labs/initia/app"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	"cosmossdk.io/math"
	cmtypes "github.com/cometbft/cometbft/types"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Bond denom should be set for staking test
const bondDenom = initiaapp.BondDenom

var (
	priv1 = secp256k1.GenPrivKey()
	addr1 = sdk.AccAddress(priv1.PubKey().Address())
	priv2 = secp256k1.GenPrivKey()
	addr2 = sdk.AccAddress(priv2.PubKey().Address())
	priv3 = secp256k1.GenPrivKey()
	addr3 = sdk.AccAddress(priv2.PubKey().Address())

	valKey1 = ed25519.GenPrivKey()
	valKey2 = ed25519.GenPrivKey()

	commissionRates = stakingtypes.NewCommissionRates(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec())

	genCoins        = sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(50000000))).Sort()
	bondCoin        = sdk.NewCoin(bondDenom, math.NewInt(10000000))
	votePower int64 = 10000
)

func createAppWithSimpleValidators(t *testing.T) *initiaapp.InitiaApp {
	privVal1 := ed25519.GenPrivKey()
	pubKey1, err := cryptocodec.ToCmtPubKeyInterface(privVal1.PubKey())
	if err != nil {
		panic(err)
	}

	privVal2 := ed25519.GenPrivKey()
	pubKey2, err := cryptocodec.ToCmtPubKeyInterface(privVal2.PubKey())
	if err != nil {
		panic(err)
	}

	validator1 := cmtypes.NewValidator(pubKey1, votePower)
	validator2 := cmtypes.NewValidator(pubKey2, votePower)

	valSet := cmtypes.NewValidatorSet([]*cmtypes.Validator{validator1, validator2})

	app := initiaapp.SetupWithGenesisAccounts(valSet, authtypes.GenesisAccounts{
		&authtypes.BaseAccount{Address: addr1.String()},
		&authtypes.BaseAccount{Address: addr2.String()},
		&authtypes.BaseAccount{Address: addr3.String()},
	},
		banktypes.Balance{Address: addr1.String(), Coins: genCoins},
		banktypes.Balance{Address: addr2.String(), Coins: genCoins},
		banktypes.Balance{Address: addr3.String(), Coins: genCoins},
	)

	return app
}
