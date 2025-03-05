package types_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/v1/x/distribution/types"
)

var (
	testPoolDenom0 = "pool0"
	testPoolDenom1 = "pool1"
	testPoolDenom2 = "pool2"
	testPoolDenom4 = "pool4"
)

var (
	testCoinDenom1 = "coin1"
	testCoinDenom2 = "coin2"
)

type poolTestSuite struct {
	suite.Suite
	ca0, ca1, ca2, ca4, cm0, cm1, cm2, cm4 sdk.Coin
	emptyCoins                             sdk.Coins

	pool0, pool1, pool2, pool4                  types.Pool
	pool000, pool111, pool222, pool444          types.Pool
	pool101, pool110, pool122, pool211, pool244 types.Pool
	emptyPools                                  types.Pools
}

func TestPoolTestSuite(t *testing.T) {
	suite.Run(t, new(poolTestSuite))
}

func (s *poolTestSuite) SetupSuite() {
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
}

func (s *poolTestSuite) TestIsEqualPool() {
	coins11 := sdk.NewCoins(sdk.NewInt64Coin(testCoinDenom1, 1), sdk.NewInt64Coin(testCoinDenom2, 1))
	coins12 := sdk.NewCoins(sdk.NewInt64Coin(testCoinDenom1, 1), sdk.NewInt64Coin(testCoinDenom2, 2))

	cases := []struct {
		inputOne types.Pool
		inputTwo types.Pool
		expected bool
	}{
		{types.NewPool(testPoolDenom1, sdk.NewCoins(coins11...)), types.NewPool(testPoolDenom1, sdk.NewCoins(coins11...)), true},
		{types.NewPool(testPoolDenom1, sdk.NewCoins(coins11...)), types.NewPool(testPoolDenom2, sdk.NewCoins(coins11...)), false},
		{types.NewPool(testPoolDenom1, sdk.NewCoins(coins11...)), types.NewPool(testPoolDenom1, sdk.NewCoins(coins12...)), false},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.IsEqual(tc.inputTwo)
		s.Require().Equal(tc.expected, res, "pool equality relation is incorrect, tc #%d", tcIndex)
	}
}

func (s *poolTestSuite) TestIsEmptyPool() {
	cases := []struct {
		input    types.Pool
		expected bool
	}{
		{types.NewPool(testPoolDenom1, s.emptyCoins), true},
		{types.NewPool(testPoolDenom1, sdk.NewCoins(s.ca1, s.cm1)), false},
	}

	for tcIndex, tc := range cases {
		res := tc.input.IsEmpty()
		s.Require().Equal(tc.expected, res, "pool emptiness is incorrect, tc #%d", tcIndex)
	}
}

func (s *poolTestSuite) TestAddPool() {
	cases := []struct {
		inputOne    types.Pool
		inputTwo    types.Pool
		expected    types.Pool
		shouldPanic bool
	}{
		{s.pool111, s.pool111, s.pool122, false},
		{s.pool101, s.pool110, s.pool111, false},
		{types.NewPool(testPoolDenom1, sdk.Coins{}), s.pool111, s.pool111, false},
		{s.pool111, s.pool211, s.pool111, true},
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

func (s *poolTestSuite) TestSubPool() {
	cases := []struct {
		inputOne    types.Pool
		inputTwo    types.Pool
		expected    types.Pool
		shouldPanic bool
	}{
		{s.pool122, s.pool111, s.pool111, false},
		{s.pool111, s.pool110, s.pool101, false},
		{s.pool111, s.pool111, types.NewPool(testPoolDenom1, sdk.Coins{}), false},
		{s.pool111, s.pool211, s.pool111, true},
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

func (s *poolTestSuite) TestNewPoolsSorted() {
	cases := []struct {
		input    types.Pools
		expected types.Pools
	}{
		{types.NewPools(s.pool111, s.pool222), types.Pools{s.pool111, s.pool222}},
		{types.NewPools(s.pool444, s.pool111), types.Pools{s.pool111, s.pool444}},
	}

	for tcIndex, tc := range cases {
		s.Require().Equal(tc.input.IsEqual(tc.expected), true, "pools are not sorted, tc #%d", tcIndex)
	}
}

func (s *poolTestSuite) TestAddPools() {
	cases := []struct {
		inputOne types.Pools
		inputTwo types.Pools
		expected types.Pools
	}{
		{types.NewPools(s.pool111, s.pool222), types.Pools{}, types.NewPools(s.pool111, s.pool222)},
		{types.NewPools(s.pool111, s.pool222), types.NewPools(s.pool111, s.pool222), types.NewPools(s.pool122, s.pool244)},
		{types.NewPools(s.pool111, s.pool222), types.NewPools(s.pool222, s.pool111), types.NewPools(s.pool122, s.pool244)},
		{types.NewPools(s.pool111, s.pool222), types.NewPools(s.pool444, s.pool000), types.NewPools(s.pool000, s.pool222, s.pool111, s.pool444)},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.Add(tc.inputTwo...)

		s.Require().Equal(tc.expected, res, "sum of pools is incorrect, tc #%d", tcIndex)
	}
}

func (s *poolTestSuite) TestSubPools() {
	cases := []struct {
		inputOne types.Pools
		inputTwo types.Pools
		expected types.Pools
	}{
		{types.NewPools(s.pool111, s.pool222), types.NewPools(s.pool111, s.pool222), types.Pools{}},
		{types.NewPools(s.pool122, s.pool244), types.NewPools(s.pool111, s.pool222), types.NewPools(s.pool111, s.pool222)},
		{types.NewPools(s.pool122, s.pool244), types.NewPools(s.pool111, s.pool222), types.NewPools(s.pool222, s.pool111)},
		{types.NewPools(s.pool000, s.pool222, s.pool111, s.pool444), types.NewPools(s.pool111, s.pool222), types.NewPools(s.pool444, s.pool000)},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.Sub(tc.inputTwo)

		s.Require().Equal(tc.expected, res, "sum of pools is incorrect, tc #%d", tcIndex)
	}
}

func (s *poolTestSuite) TestIsAnyNegativePools() {
	cases := []struct {
		input    types.Pools
		expected bool
	}{
		{types.NewPools(s.pool111, s.pool222, s.pool444), false},
		{types.NewPools(types.Pool{"test", sdk.Coins{sdk.Coin{"testdenom", math.NewInt(-10)}}}), true},
	}

	for tcIndex, tc := range cases {
		res := tc.input.IsAnyNegative()
		s.Require().Equal(tc.expected, res, "negative pool coins check is incorrect, tc #%d", tcIndex)
	}
}

func (s *poolTestSuite) TestCoinsOfPools() {
	pools := types.NewPools(s.pool111, s.pool222, s.pool444)

	cases := []struct {
		input    string
		expected sdk.Coins
	}{
		{testPoolDenom1, sdk.NewCoins(s.ca1, s.cm1)},
		{testPoolDenom2, sdk.NewCoins(s.ca2, s.cm2)},
		{testPoolDenom4, sdk.NewCoins(s.ca4, s.cm4)},
	}

	for tcIndex, tc := range cases {
		res := pools.CoinsOf(tc.input)
		s.Require().True(tc.expected.Equal(res), "pool coins retrieval is incorrect, tc #%d", tcIndex)
	}
}

func (s *poolTestSuite) TestIsEmptyPools() {
	cases := []struct {
		input    types.Pools
		expected bool
	}{
		{types.NewPools(s.pool0), true},
		{types.NewPools(s.pool111), false},
	}

	for tcIndex, tc := range cases {
		res := tc.input.IsEmpty()
		s.Require().Equal(tc.expected, res, "pool emptiness is incorrect, tc #%d", tcIndex)
	}
}

func (s *poolTestSuite) TestIsEqualPools() {
	cases := []struct {
		inputOne types.Pools
		inputTwo types.Pools
		expected bool
	}{
		{types.NewPools(s.pool000, s.pool111), types.NewPools(s.pool111), true},
		{types.NewPools(s.pool111, s.pool222), types.NewPools(s.pool111), false},
		{types.NewPools(s.pool111, s.pool222), types.Pools{s.pool111, s.pool222, s.pool000}, false}, // should we delete empty pool?
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.IsEqual(tc.inputTwo)
		s.Require().Equal(tc.expected, res, "pools equality relation is incorrect, tc #%d", tcIndex)
	}
}
