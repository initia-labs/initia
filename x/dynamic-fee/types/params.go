package types

import (
	"fmt"

	"cosmossdk.io/math"
)

var (
	DefaultBaseGasPrice    = math.LegacyNewDecWithPrec(15, 3) // 0.015
	DefaultMinBaseGasPrice = math.LegacyNewDecWithPrec(15, 3) // 0.015
	DefaultMaxBaseGasPrice = math.LegacyNewDec(10)            // 10

	// 200_000_000 * 0.5
	DefaultTargetGas int64 = 100_000_000

	// 0.1
	DefaultMaxChangeRate = math.LegacyNewDecWithPrec(1, 1)
)

func DefaultParams() Params {
	return Params{
		BaseGasPrice:    DefaultBaseGasPrice,
		MinBaseGasPrice: DefaultMinBaseGasPrice,
		MaxBaseGasPrice: DefaultMaxBaseGasPrice,
		TargetGas:       DefaultTargetGas,
		MaxChangeRate:   DefaultMaxChangeRate,
	}
}

func NoBaseGasPriceChangeParams() Params {
	return Params{
		BaseGasPrice:    DefaultBaseGasPrice,
		MinBaseGasPrice: DefaultMinBaseGasPrice,
		MaxBaseGasPrice: DefaultMaxBaseGasPrice,
		TargetGas:       DefaultTargetGas,
		MaxChangeRate:   math.LegacyZeroDec(),
	}
}

func (p Params) Validate() error {
	if p.BaseGasPrice.IsNegative() {
		return fmt.Errorf("base gas price must be non-negative")
	}

	if p.MinBaseGasPrice.IsNegative() {
		return fmt.Errorf("min base gas price must be non-negative")
	}

	if p.MaxBaseGasPrice.IsNegative() {
		return fmt.Errorf("max base gas price must be non-negative")
	}

	if p.TargetGas < 0 {
		return fmt.Errorf("target gas must be non-negative")
	}

	if p.BaseGasPrice.LT(p.MinBaseGasPrice) {
		return fmt.Errorf("base gas price must be greater than or equal to min base gas price")
	}

	if p.BaseGasPrice.GT(p.MaxBaseGasPrice) {
		return fmt.Errorf("base gas price must be less than or equal to max base gas price")
	}

	if p.MaxChangeRate.IsNegative() {
		return fmt.Errorf("max change rate must be non-negative")
	}

	return nil
}
