package types_test

import (
	"math/rand"
	"sort"
	"testing"

	"cosmossdk.io/math"

	cmtcrypto "github.com/cometbft/cometbft/crypto"
	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"

	"github.com/initia-labs/initia/x/mstaking/types"
)

func coins(amt int64) sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(amt)))
}

func decCoins(amt int64) sdk.DecCoins {
	return sdk.NewDecCoins(sdk.NewDecCoin(sdk.DefaultBondDenom, math.NewInt(amt)))
}

func decCoinsFromDec(amt math.LegacyDec) sdk.DecCoins {
	return sdk.NewDecCoins(sdk.NewDecCoinFromDec(sdk.DefaultBondDenom, amt))
}

func TestValidatorTestEquivalent(t *testing.T) {
	val1 := newValidator(t, valAddr1, pk1)
	val2 := newValidator(t, valAddr1, pk1)
	require.Equal(t, val1.String(), val2.String())

	val2 = newValidator(t, valAddr2, pk2)
	require.NotEqual(t, val1.String(), val2.String())
}

func TestUpdateDescription(t *testing.T) {
	d1 := types.Description{
		Website: "https://validator.cosmos",
		Details: "Test validator",
	}

	d2 := types.Description{
		Moniker:  types.DoNotModifyDesc,
		Identity: types.DoNotModifyDesc,
		Website:  types.DoNotModifyDesc,
		Details:  types.DoNotModifyDesc,
	}

	d3 := types.Description{
		Moniker:  "",
		Identity: "",
		Website:  "",
		Details:  "",
	}

	d, err := d1.UpdateDescription(d2)
	require.Nil(t, err)
	require.Equal(t, d, d1)

	d, err = d1.UpdateDescription(d3)
	require.Nil(t, err)
	require.Equal(t, d, d3)
}

func TestABCIValidatorUpdate(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	validator.VotingPower = math.NewInt(100).Mul(sdk.DefaultPowerReduction)
	validator.Status = types.Bonded
	abciVal := validator.ABCIValidatorUpdate(sdk.DefaultPowerReduction)
	pk, err := validator.TmConsPublicKey()
	require.NoError(t, err)
	require.Equal(t, pk, abciVal.PubKey)
	require.Equal(t, int64(100), abciVal.Power)
}

func TestABCIValidatorUpdateZero(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	abciVal := validator.ABCIValidatorUpdateZero()
	pk, err := validator.TmConsPublicKey()
	require.NoError(t, err)
	require.Equal(t, pk, abciVal.PubKey)
	require.Equal(t, int64(0), abciVal.Power)
}

func TestShareTokens(t *testing.T) {
	validator := mkValidator(coins(100), decCoins(100))

	assert.Equal(t, decCoins(50), validator.TokensFromShares(sdk.NewDecCoins(sdk.NewDecCoin(sdk.DefaultBondDenom, math.NewInt(50)))))

	validator.Tokens = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(50)))
	assert.Equal(t, decCoins(25), validator.TokensFromShares(sdk.NewDecCoins(sdk.NewDecCoin(sdk.DefaultBondDenom, math.NewInt(50)))))
	assert.Equal(t, decCoins(5), validator.TokensFromShares(sdk.NewDecCoins(sdk.NewDecCoin(sdk.DefaultBondDenom, math.NewInt(10)))))
}

func TestRemoveTokens(t *testing.T) {
	validator := mkValidator(coins(100), decCoins(100))

	// remove tokens and test check everything
	validator = validator.RemoveTokens(coins(10))
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(90))), validator.Tokens)

	// update validator to from bonded -> unbonded
	validator = validator.UpdateStatus(types.Unbonded)
	require.Equal(t, types.Unbonded, validator.Status)

	validator = validator.RemoveTokens(coins(10))
	require.Panics(t, func() { validator.RemoveTokens(coins(-1)) })
	require.Panics(t, func() { validator.RemoveTokens(coins(100)) })
}

func TestAddTokensValidatorBonded(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	validator = validator.UpdateStatus(types.Bonded)
	validator, delShares := validator.AddTokensFromDel(coins(10))

	require.True(t, decCoins(10).Equal(delShares))
	require.True(t, coins(10).Equal(validator.BondedTokens()))
	require.True(t, decCoins(10).Equal(validator.DelegatorShares))
}

func TestAddTokensValidatorUnbonding(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	validator = validator.UpdateStatus(types.Unbonding)
	validator, delShares := validator.AddTokensFromDel(coins(10))

	require.Equal(t, decCoins(10), delShares)
	require.Equal(t, types.Unbonding, validator.Status)
	require.Equal(t, coins(10), validator.Tokens)
	require.Equal(t, decCoins(10), validator.DelegatorShares)
}

func TestAddTokensValidatorUnbonded(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	validator = validator.UpdateStatus(types.Unbonded)
	validator, delShares := validator.AddTokensFromDel(coins(10))

	require.Equal(t, decCoins(10), delShares)
	require.Equal(t, types.Unbonded, validator.Status)
	require.Equal(t, coins(10), validator.Tokens)
	require.Equal(t, decCoins(10), validator.DelegatorShares)
}

// TODO refactor to make simpler like the AddToken tests above
func TestRemoveDelShares(t *testing.T) {
	valA := types.Validator{
		OperatorAddress: valAddr1.String(),
		ConsensusPubkey: pk1Any,
		Status:          types.Bonded,
		Tokens:          coins(100),
		DelegatorShares: decCoins(100),
	}

	// Remove delegator shares
	valB, coinsB := valA.RemoveDelShares(decCoins(10))
	require.Equal(t, coins(10), coinsB)
	require.Equal(t, decCoins(90), valB.DelegatorShares)
	require.Equal(t, coins(90), valB.BondedTokens())

	// specific case from random tests
	validator := mkValidator(coins(5102), decCoins(115))
	_, tokens := validator.RemoveDelShares(decCoins(29))

	require.Equal(t, coins(1286), tokens)
}

func TestAddTokensFromDel(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)

	validator, shares := validator.AddTokensFromDel(coins(6))
	require.Equal(t, decCoins(6), shares)
	require.Equal(t, decCoins(6), validator.DelegatorShares)
	require.Equal(t, coins(6), validator.Tokens)

	validator, shares = validator.AddTokensFromDel(coins(3))
	require.Equal(t, decCoins(3), shares)
	require.Equal(t, decCoins(9), validator.DelegatorShares)
	require.Equal(t, coins(9), validator.Tokens)
}

func TestUpdateStatus(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	validator, _ = validator.AddTokensFromDel(coins(100))
	require.Equal(t, types.Unbonded, validator.Status)
	require.Equal(t, coins(100), validator.Tokens)

	// Unbonded to Bonded
	validator = validator.UpdateStatus(types.Bonded)
	require.Equal(t, types.Bonded, validator.Status)

	// Bonded to Unbonding
	validator = validator.UpdateStatus(types.Unbonding)
	require.Equal(t, types.Unbonding, validator.Status)

	// Unbonding to Bonded
	validator = validator.UpdateStatus(types.Bonded)
	require.Equal(t, types.Bonded, validator.Status)
}

func TestPossibleOverflow(t *testing.T) {
	delShares := math.LegacyNewDec(391432570689183511).Quo(math.LegacyNewDec(40113011844664))
	validator := mkValidator(coins(2159), decCoinsFromDec(delShares))
	newValidator, _ := validator.AddTokensFromDel(coins(71))

	require.False(t, newValidator.DelegatorShares.IsAnyNegative())
	require.False(t, newValidator.Tokens.IsAnyNegative())
}

func TestValidatorMarshalUnmarshalJSON(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	js, err := legacy.Cdc.MarshalJSON(validator)
	require.NoError(t, err)
	require.NotEmpty(t, js)
	require.Contains(t, string(js), "\"consensus_pubkey\":{\"type\":\"tendermint/PubKeyEd25519\"")
	got := &types.Validator{}
	err = legacy.Cdc.UnmarshalJSON(js, got)
	assert.NoError(t, err)
	assert.True(t, validator.Equal(got))
}

func TestValidatorSetInitialCommission(t *testing.T) {
	val := newValidator(t, valAddr1, pk1)
	testCases := []struct {
		validator   types.Validator
		commission  types.Commission
		expectedErr bool
	}{
		{val, types.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()), false},
		{val, types.NewCommission(math.LegacyZeroDec(), math.LegacyNewDecWithPrec(-1, 1), math.LegacyZeroDec()), true},
		{val, types.NewCommission(math.LegacyZeroDec(), math.LegacyNewDec(15000000000), math.LegacyZeroDec()), true},
		{val, types.NewCommission(math.LegacyNewDecWithPrec(-1, 1), math.LegacyZeroDec(), math.LegacyZeroDec()), true},
		{val, types.NewCommission(math.LegacyNewDecWithPrec(2, 1), math.LegacyNewDecWithPrec(1, 1), math.LegacyZeroDec()), true},
		{val, types.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyNewDecWithPrec(-1, 1)), true},
		{val, types.NewCommission(math.LegacyZeroDec(), math.LegacyNewDecWithPrec(1, 1), math.LegacyNewDecWithPrec(2, 1)), true},
	}

	for i, tc := range testCases {
		val, err := tc.validator.SetInitialCommission(tc.commission)

		if tc.expectedErr {
			require.Error(t, err,
				"expected error for test case #%d with commission: %s", i, tc.commission,
			)
		} else {
			require.NoError(t, err,
				"unexpected error for test case #%d with commission: %s", i, tc.commission,
			)
			require.Equal(t, tc.commission, val.Commission,
				"invalid validator commission for test case #%d with commission: %s", i, tc.commission,
			)
		}
	}
}

// Check that sort will create deterministic ordering of validators
func TestValidatorsSortDeterminism(t *testing.T) {
	vals := make([]types.Validator, 10)
	sortedVals := make([]types.Validator, 10)

	// Create random validator slice
	for i := range vals {
		pk := ed25519.GenPrivKey().PubKey()
		vals[i] = newValidator(t, sdk.ValAddress(pk.Address()), pk)
	}

	// Save sorted copy
	sort.Sort(types.Validators{Validators: vals, ValidatorCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())})
	copy(sortedVals, vals)

	// Randomly shuffle validators, sort, and check it is equal to original sort
	for i := 0; i < 10; i++ {
		rand.Shuffle(10, func(i, j int) {
			it := vals[i]
			vals[i] = vals[j]
			vals[j] = it
		})

		types.Validators{Validators: vals, ValidatorCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())}.Sort()
		require.Equal(t, sortedVals, vals, "Validator sort returned different slices")
	}
}

// Check SortTendermint sorts the same as tendermint
func TestValidatorsSortCometBFT(t *testing.T) {
	vals := make([]types.Validator, 100)

	for i := range vals {
		pk := ed25519.GenPrivKey().PubKey()
		pk2 := ed25519.GenPrivKey().PubKey()
		vals[i] = newValidator(t, sdk.ValAddress(pk2.Address()), pk)
		vals[i].Status = types.Bonded
		vals[i].VotingPower = math.NewInt(rand.Int63())
	}
	// create some validators with the same power
	for i := 0; i < 10; i++ {
		vals[i].VotingPower = math.NewInt(1000000)
	}

	valz := types.Validators{Validators: vals, ValidatorCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())}

	// create expected tendermint validators by converting to tendermint then sorting
	expectedVals, err := toCmtValidators(valz, sdk.DefaultPowerReduction)
	require.NoError(t, err)
	sort.Sort(cmttypes.ValidatorsByVotingPower(expectedVals))

	// sort in SDK and then convert to tendermint
	sort.SliceStable(valz.Validators, func(i, j int) bool {
		return types.ValidatorsByVotingPower(valz.Validators).Less(i, j, sdk.DefaultPowerReduction)
	})
	actualVals, err := toCmtValidators(valz, sdk.DefaultPowerReduction)
	require.NoError(t, err)

	require.Equal(t, expectedVals, actualVals, "sorting in SDK is not the same as sorting in Tendermint")
}

func TestValidatorToCmt(t *testing.T) {
	vals := types.Validators{}
	expected := make([]*cmttypes.Validator, 10)

	for i := 0; i < 10; i++ {
		pk := ed25519.GenPrivKey().PubKey()
		val := newValidator(t, sdk.ValAddress(pk.Address()), pk)
		val.Status = types.Bonded
		val.VotingPower = math.NewInt(rand.Int63())
		vals.Validators = append(vals.Validators, val)
		cmtPk, err := cryptocodec.ToCmtPubKeyInterface(pk)
		require.NoError(t, err)
		expected[i] = cmttypes.NewValidator(cmtPk, val.ConsensusPower(sdk.DefaultPowerReduction))
	}

	vs, err := toCmtValidators(vals, sdk.DefaultPowerReduction)
	require.NoError(t, err)
	require.Equal(t, expected, vs)
}

func TestBondStatus(t *testing.T) {
	require.False(t, types.Unbonded == types.Bonded)
	require.False(t, types.Unbonded == types.Unbonding)
	require.False(t, types.Bonded == types.Unbonding)
	require.Equal(t, types.BondStatus(4).String(), "4")
	require.Equal(t, types.BondStatusUnspecified, types.Unspecified.String())
	require.Equal(t, types.BondStatusUnbonded, types.Unbonded.String())
	require.Equal(t, types.BondStatusBonded, types.Bonded.String())
	require.Equal(t, types.BondStatusUnbonding, types.Unbonding.String())
}

func mkValidator(tokens sdk.Coins, shares sdk.DecCoins) types.Validator {
	return types.Validator{
		OperatorAddress: valAddr1.String(),
		ConsensusPubkey: pk1Any,
		Status:          types.Bonded,
		Tokens:          tokens,
		DelegatorShares: shares,
	}
}

// Creates a new validators and asserts the error check.
func newValidator(t *testing.T, operator sdk.ValAddress, pubKey cryptotypes.PubKey) types.Validator {
	valAddr, err := address.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()).BytesToString(operator)
	require.NoError(t, err)

	v, err := types.NewValidator(valAddr, pubKey, types.Description{})
	require.NoError(t, err)
	return v
}

// getCmtConsPubKey gets the validator's public key as a cmtcrypto.PubKey.
func getCmtConsPubKey(v types.Validator) (cmtcrypto.PubKey, error) {
	pk, err := v.ConsPubKey()
	if err != nil {
		return nil, err
	}

	return cryptocodec.ToCmtPubKeyInterface(pk)
}

// toCmtValidator casts an SDK validator to a CometBFT type Validator.
func toCmtValidator(v types.Validator, r math.Int) (*cmttypes.Validator, error) {
	tmPk, err := getCmtConsPubKey(v)
	if err != nil {
		return nil, err
	}

	return cmttypes.NewValidator(tmPk, v.ConsensusPower(r)), nil
}

// toCmtValidators casts all validators to the corresponding CometBFT type.
func toCmtValidators(v types.Validators, r math.Int) ([]*cmttypes.Validator, error) {
	validators := make([]*cmttypes.Validator, len(v.Validators))
	var err error
	for i, val := range v.Validators {
		validators[i], err = toCmtValidator(val, r)
		if err != nil {
			return nil, err
		}
	}

	return validators, nil
}
