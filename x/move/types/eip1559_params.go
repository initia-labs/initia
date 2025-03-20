package types

import (
	"fmt"

	"cosmossdk.io/math"
)

var (
	// 0.01 INIT
	DefaultBaseFee int64 = 10_000
	// 0.001 INIT
	DefaultMinBaseFee int64 = 1_000
	// 10 INIT
	DefaultMaxBaseFee int64 = 10_000_000

	// 200_000_000 * 0.5
	DefaultTargetGas int64 = 100_000_000

	// 0.1
	DefaultMaxChangeRate = math.LegacyNewDecWithPrec(1, 1)
)

func DefaultEIP1559Params() EIP1559FeeParams {
	return EIP1559FeeParams{
		BaseFee:       DefaultBaseFee,
		MinBaseFee:    DefaultMinBaseFee,
		MaxBaseFee:    DefaultMaxBaseFee,
		TargetGas:     DefaultTargetGas,
		MaxChangeRate: DefaultMaxChangeRate,
	}
}

func (p EIP1559FeeParams) Validate() error {
	if p.BaseFee < 0 {
		return fmt.Errorf("base fee must be non-negative")
	}

	if p.MinBaseFee < 0 {
		return fmt.Errorf("min base fee must be non-negative")
	}

	if p.MaxBaseFee < 0 {
		return fmt.Errorf("max base fee must be non-negative")
	}

	if p.TargetGas < 0 {
		return fmt.Errorf("target gas must be non-negative")
	}

	if p.BaseFee < p.MinBaseFee {
		return fmt.Errorf("base fee must be greater than or equal to min base fee")
	}

	if p.BaseFee > p.MaxBaseFee {
		return fmt.Errorf("base fee must be less than or equal to max base fee")
	}

	if p.MaxChangeRate.IsNegative() {
		return fmt.Errorf("max change rate must be non-negative")
	}

	return nil
}
