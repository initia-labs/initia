package types_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/distribution/types"
)

type decpoolTestSuite struct {
	suite.Suite
	ca0, ca1, ca2, ca4, cm0, cm1, cm2, cm4 sdk.Coin
	emptyCoins                             sdk.Coins

	pool0, pool1, pool2, pool4                  types.Pool
	pool000, pool111, pool222, pool444          types.Pool
	pool101, pool110, pool122, pool211, pool244 types.Pool
	emptyPools                                  types.Pools

	decpool0, decpool1, decpool2, decpool4                     types.DecPool
	decpool000, decpool111, decpool222, decpool444             types.DecPool
	decpool101, decpool110, decpool122, decpool211, decpool244 types.DecPool
	emptyDecPools                                              types.DecPools
}

func TestDecPoolTestSuite(t *testing.T) {
	suite.Run(t, new(decpoolTestSuite))
}

func (s *decpoolTestSuite) SetupSuite() {
	zero := math.NewInt(0)
	one := math.OneInt()
	two := math.NewInt(2)
	four := math.NewInt(4)

	s.ca0, s.ca1, s.ca2, s.ca4 = sdk.NewCoin(testCoinDenom1, zero), sdk.NewCoin(testCoinDenom1, one), sdk.NewCoin(testCoinDenom1, two), sdk.NewCoin(testCoinDenom1, four)
	s.cm0, s.cm1, s.cm2, s.cm4 = sdk.NewCoin(testCoinDenom2, zero), sdk.NewCoin(testCoinDenom2, one), sdk.NewCoin(testCoinDenom2, two), sdk.NewCoin(testCoinDenom2, four)
	s.emptyCoins = sdk.Coins{}

	s.pool0 = types.NewPool(testPoolDenom0, sdk.Coins{})
	s.pool1 = types.NewPool(testPoolDenom1, sdk.Coins{})
	s.pool2 = types.NewPool(testPoolDenom2, sdk.Coins{})
	s.pool4 = types.NewPool(testPoolDenom4, sdk.Coins{})

	s.pool000 = types.NewPool(testPoolDenom0, sdk.NewCoins(s.ca0, s.cm0))
	s.pool111 = types.NewPool(testPoolDenom1, sdk.NewCoins(s.ca1, s.cm1))
	s.pool222 = types.NewPool(testPoolDenom2, sdk.NewCoins(s.ca2, s.cm2))
	s.pool444 = types.NewPool(testPoolDenom4, sdk.NewCoins(s.ca4, s.cm4))

	s.pool101 = types.NewPool(testPoolDenom1, sdk.NewCoins(s.cm1))
	s.pool110 = types.NewPool(testPoolDenom1, sdk.NewCoins(s.ca1))

	s.pool122 = types.NewPool(testPoolDenom1, sdk.NewCoins(s.ca2, s.cm2))
	s.pool211 = types.NewPool(testPoolDenom2, sdk.NewCoins(s.ca1, s.cm1))
	s.pool244 = types.NewPool(testPoolDenom2, sdk.NewCoins(s.ca4, s.cm4))

	s.emptyPools = types.Pools{}

	s.decpool0 = types.NewDecPool(testPoolDenom0, sdk.DecCoins{})
	s.decpool1 = types.NewDecPool(testPoolDenom1, sdk.DecCoins{})
	s.decpool2 = types.NewDecPool(testPoolDenom2, sdk.DecCoins{})
	s.decpool4 = types.NewDecPool(testPoolDenom4, sdk.DecCoins{})

	s.decpool000 = types.NewDecPool(testPoolDenom0, sdk.NewDecCoinsFromCoins(s.ca0, s.cm0))
	s.decpool111 = types.NewDecPool(testPoolDenom1, sdk.NewDecCoinsFromCoins(s.ca1, s.cm1))
	s.decpool222 = types.NewDecPool(testPoolDenom2, sdk.NewDecCoinsFromCoins(s.ca2, s.cm2))
	s.decpool444 = types.NewDecPool(testPoolDenom4, sdk.NewDecCoinsFromCoins(s.ca4, s.cm4))

	s.decpool101 = types.NewDecPool(testPoolDenom1, sdk.NewDecCoinsFromCoins(s.cm1))
	s.decpool110 = types.NewDecPool(testPoolDenom1, sdk.NewDecCoinsFromCoins(s.ca1))

	s.decpool122 = types.NewDecPool(testPoolDenom1, sdk.NewDecCoinsFromCoins(s.ca2, s.cm2))
	s.decpool211 = types.NewDecPool(testPoolDenom2, sdk.NewDecCoinsFromCoins(s.ca1, s.cm1))
	s.decpool244 = types.NewDecPool(testPoolDenom2, sdk.NewDecCoinsFromCoins(s.ca4, s.cm4))

	s.emptyDecPools = types.DecPools{}
}

func (s *decpoolTestSuite) TestIsEqualPool() {
	coins11 := sdk.NewDecCoins(sdk.NewInt64DecCoin(testCoinDenom1, 1), sdk.NewInt64DecCoin(testCoinDenom2, 1))
	coins12 := sdk.NewDecCoins(sdk.NewInt64DecCoin(testCoinDenom1, 1), sdk.NewInt64DecCoin(testCoinDenom2, 2))

	cases := []struct {
		inputOne types.DecPool
		inputTwo types.DecPool
		expected bool
	}{
		{types.NewDecPool(testPoolDenom1, sdk.NewDecCoins(coins11...)), types.NewDecPool(testPoolDenom1, sdk.NewDecCoins(coins11...)), true},
		{types.NewDecPool(testPoolDenom1, sdk.NewDecCoins(coins11...)), types.NewDecPool(testPoolDenom2, sdk.NewDecCoins(coins11...)), false},
		{types.NewDecPool(testPoolDenom1, sdk.NewDecCoins(coins11...)), types.NewDecPool(testPoolDenom1, sdk.NewDecCoins(coins12...)), false},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.IsEqual(tc.inputTwo)
		s.Require().Equal(tc.expected, res, "pool equality relation is incorrect, tc #%d", tcIndex)
	}
}

func (s *decpoolTestSuite) TestIsEmptyPool() {
	cases := []struct {
		input    types.DecPool
		expected bool
	}{
		{types.NewDecPool(testPoolDenom1, sdk.DecCoins{}), true},
		{types.NewDecPool(testPoolDenom1, sdk.NewDecCoins(sdk.NewDecCoinFromCoin(s.ca1), sdk.NewDecCoinFromCoin(s.cm1))), false},
	}

	for tcIndex, tc := range cases {
		res := tc.input.IsEmpty()
		s.Require().Equal(tc.expected, res, "pool emptiness is incorrect, tc #%d", tcIndex)
	}
}

func (s *decpoolTestSuite) TestAddPool() {
	cases := []struct {
		inputOne    types.DecPool
		inputTwo    types.DecPool
		expected    types.DecPool
		shouldPanic bool
	}{
		{s.decpool111, s.decpool111, s.decpool122, false},
		{s.decpool101, s.decpool110, s.decpool111, false},
		{types.NewDecPool(testPoolDenom1, sdk.DecCoins{}), s.decpool111, s.decpool111, false},
		{s.decpool111, s.decpool211, s.decpool111, true},
	}

	for tcIndex, tc := range cases {
		if tc.shouldPanic {
			s.Require().Panics(func() { tc.inputOne.Add(tc.inputTwo) })
		} else {
			res := tc.inputOne.Add(tc.inputTwo)
			s.Require().Equal(tc.expected, res, "sum of pools is incorrect, tc #%d", tcIndex)
		}
	}
}

func (s *decpoolTestSuite) TestSubPool() {
	cases := []struct {
		inputOne    types.DecPool
		inputTwo    types.DecPool
		expected    types.DecPool
		shouldPanic bool
	}{
		{s.decpool122, s.decpool111, s.decpool111, false},
		{s.decpool111, s.decpool110, s.decpool101, false},
		{s.decpool111, s.decpool111, types.DecPool{testPoolDenom1, sdk.DecCoins(nil)}, false},
		{s.decpool111, s.decpool211, s.decpool111, true},
	}

	for tcIndex, tc := range cases {
		if tc.shouldPanic {
			s.Require().Panics(func() { tc.inputOne.Sub(tc.inputTwo) })
		} else {
			res := tc.inputOne.Sub(tc.inputTwo)
			s.Require().Equal(tc.expected, res, "sum of pools is incorrect, tc #%d", tcIndex)
		}
	}
}

func (s *decpoolTestSuite) TestNewPoolsSorted() {
	cases := []struct {
		input    types.DecPools
		expected types.DecPools
	}{
		{types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.DecPools{s.decpool111, s.decpool222}},
		{types.NewDecPoolsFromPools(types.NewPools(s.pool444, s.pool111)), types.DecPools{s.decpool111, s.decpool444}},
	}

	for tcIndex, tc := range cases {
		s.Require().Equal(tc.input.IsEqual(tc.expected), true, "pools are not sorted, tc #%d", tcIndex)
	}
}

func (s *decpoolTestSuite) TestAddPools() {
	cases := []struct {
		inputOne types.DecPools
		inputTwo types.DecPools
		expected types.DecPools
	}{
		{types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.DecPools{}, types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222))},
		{types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.NewDecPoolsFromPools(types.NewPools(s.pool122, s.pool244))},
		{types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.NewDecPoolsFromPools(types.NewPools(s.pool222, s.pool111)), types.NewDecPoolsFromPools(types.NewPools(s.pool122, s.pool244))},
		{types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.NewDecPoolsFromPools(types.NewPools(s.pool444, s.pool000)), types.NewDecPoolsFromPools(types.NewPools(s.pool000, s.pool222, s.pool111, s.pool444))},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.Add(tc.inputTwo...)

		s.Require().Equal(tc.expected, res, "sum of pools is incorrect, tc #%d", tcIndex)
	}
}

func (s *decpoolTestSuite) TestSubPools() {
	cases := []struct {
		inputOne types.DecPools
		inputTwo types.DecPools
		expected types.DecPools
	}{
		{types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.DecPools(nil)},
		{types.NewDecPoolsFromPools(types.NewPools(s.pool122, s.pool244)), types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222))},
		{types.NewDecPoolsFromPools(types.NewPools(s.pool122, s.pool244)), types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.NewDecPoolsFromPools(types.NewPools(s.pool222, s.pool111))},
		{types.NewDecPoolsFromPools(types.NewPools(s.pool000, s.pool222, s.pool111, s.pool444)), types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.NewDecPoolsFromPools(types.NewPools(s.pool444, s.pool000))},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.Sub(tc.inputTwo)

		s.Require().Equal(tc.expected, res, "sum of pools is incorrect, tc #%d", tcIndex)
	}
}

func (s *decpoolTestSuite) TestIsAnyNegativePools() {
	cases := []struct {
		input    types.DecPools
		expected bool
	}{
		{types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222, s.pool444)), false},
		{types.DecPools{types.DecPool{"test", sdk.DecCoins{sdk.DecCoin{Denom: "testdenom", Amount: math.LegacyNewDecFromInt(math.NewInt(-10))}}}}, true},
	}

	for tcIndex, tc := range cases {
		res := tc.input.IsAnyNegative()
		s.Require().Equal(tc.expected, res, "negative pool coins check is incorrect, tc #%d", tcIndex)
	}
}

func (s *decpoolTestSuite) TestCoinsOfPools() {
	pools := types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222, s.pool444))

	cases := []struct {
		input    string
		expected sdk.DecCoins
	}{
		{testPoolDenom1, sdk.NewDecCoinsFromCoins(s.ca1, s.cm1)},
		{testPoolDenom2, sdk.NewDecCoinsFromCoins(s.ca2, s.cm2)},
		{testPoolDenom4, sdk.NewDecCoinsFromCoins(s.ca4, s.cm4)},
	}

	for tcIndex, tc := range cases {
		res := pools.CoinsOf(tc.input)
		s.Require().True(tc.expected.Equal(res), "pool coins retrieval is incorrect, tc #%d", tcIndex)
	}
}

func (s *decpoolTestSuite) TestIsEmptyPools() {
	cases := []struct {
		input    types.DecPools
		expected bool
	}{
		{types.NewDecPoolsFromPools(types.NewPools(s.pool0)), true},
		{types.NewDecPoolsFromPools(types.NewPools(s.pool111)), false},
	}

	for tcIndex, tc := range cases {
		res := tc.input.IsEmpty()
		s.Require().Equal(tc.expected, res, "pool emptiness is incorrect, tc #%d", tcIndex)
	}
}

func (s *decpoolTestSuite) TestIsEqualPools() {
	cases := []struct {
		inputOne types.DecPools
		inputTwo types.DecPools
		expected bool
	}{
		{types.NewDecPoolsFromPools(types.NewPools(s.pool000, s.pool111)), types.NewDecPoolsFromPools(types.NewPools(s.pool111)), true},
		{types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.NewDecPoolsFromPools(types.NewPools(s.pool111)), false},
		{types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)), types.NewDecPoolsFromPools(types.Pools{s.pool111, s.pool222, s.pool000}), false}, // should we delete empty pool?
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.IsEqual(tc.inputTwo)
		s.Require().Equal(tc.expected, res, "pools equality relation is incorrect, tc #%d", tcIndex)
	}
}

func (s *decpoolTestSuite) TestSumPools() {
	cases := []struct {
		input    types.DecPools
		expected sdk.DecCoins
	}{
		{types.NewDecPoolsFromPools(types.NewPools(s.pool122, s.pool222)), sdk.NewDecCoinsFromCoins(s.ca4, s.cm4)},
		{types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool111)), sdk.NewDecCoinsFromCoins(s.ca2, s.cm2)},
	}

	for tcIndex, tc := range cases {
		res := tc.input.Sum()
		s.Require().True(tc.expected.Equal(res), "sum of pools is incorrect, tc #%d", tcIndex)
	}
}

func (s *decpoolTestSuite) TestTruncatePools() {
	cases := []struct {
		input     types.DecPools
		expected1 types.Pools
		expected2 types.DecPools
	}{
		{
			types.DecPools{types.DecPool{"test1", sdk.DecCoins{sdk.DecCoin{Denom: "testdenom1", Amount: math.LegacyNewDecFromIntWithPrec(math.NewInt(10500), 3)}}}},
			types.Pools{types.Pool{"test1", sdk.Coins{sdk.Coin{Denom: "testdenom1", Amount: math.NewInt(10)}}}},
			types.DecPools{types.DecPool{"test1", sdk.DecCoins{sdk.DecCoin{Denom: "testdenom1", Amount: math.LegacyNewDecFromIntWithPrec(math.NewInt(500), 3)}}}},
		},
		{
			types.DecPools{types.DecPool{"test1", sdk.DecCoins{sdk.DecCoin{Denom: "testdenom1", Amount: math.LegacyNewDecFromIntWithPrec(math.NewInt(10000), 3)}}}},
			types.Pools{types.Pool{"test1", sdk.Coins{sdk.Coin{Denom: "testdenom1", Amount: math.NewInt(10)}}}},
			types.DecPools{},
		},
	}

	for tcIndex, tc := range cases {
		res1, res2 := tc.input.TruncateDecimal()
		s.Require().True(tc.expected1.IsEqual(res1), "truncated pools are incorrect, tc #%d", tcIndex)
		s.Require().True(tc.expected2.IsEqual(res2), "change pools are incorrect, tc #%d", tcIndex)
	}
}

func (s *decpoolTestSuite) TestIntersectPools() {
	cases := []struct {
		inputOne types.DecPools
		inputTwo types.DecPools
		expected types.DecPools
	}{
		{
			types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)),
			types.NewDecPoolsFromPools(types.NewPools(s.pool110)),
			types.NewDecPoolsFromPools(types.NewPools(s.pool110)),
		},
		{
			types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool222)),
			types.NewDecPoolsFromPools(types.NewPools(s.pool111, s.pool444)),
			types.NewDecPoolsFromPools(types.NewPools(s.pool111)),
		},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.Intersect(tc.inputTwo)
		s.Require().True(tc.expected.IsEqual(res), "intersection of pools is incorrect, tc #%d", tcIndex)
	}
}
