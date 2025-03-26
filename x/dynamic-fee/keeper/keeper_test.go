package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/dynamic-fee/types"
)

func Test_UpdateBaseFee(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	err := input.DynamicFeeKeeper.SetParams(ctx, types.Params{
		MinBaseGasPrice: math.LegacyNewDecWithPrec(1, 3),
		MaxBaseGasPrice: math.LegacyNewDec(200),
		BaseGasPrice:    math.LegacyNewDecWithPrec(1, 2),
		TargetGas:       100000,
		MaxChangeRate:   math.LegacyNewDecWithPrec(1, 1),
	})
	require.NoError(t, err)

	baseGasPrice, err := input.DynamicFeeKeeper.BaseGasPrice(ctx)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 2), baseGasPrice)

	// accumulate gas
	input.DynamicFeeKeeper.AccumulateGas(ctx, 100000)

	// update base fee
	err = input.DynamicFeeKeeper.UpdateBaseGasPrice(ctx)
	require.NoError(t, err)

	baseGasPrice, err = input.DynamicFeeKeeper.BaseGasPrice(ctx)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 2), baseGasPrice)

	// accumulate gas
	input.DynamicFeeKeeper.ResetAccumulatedGas(ctx)
	input.DynamicFeeKeeper.AccumulateGas(ctx, 200000)

	// update base fee
	err = input.DynamicFeeKeeper.UpdateBaseGasPrice(ctx)
	require.NoError(t, err)

	baseGasPrice, err = input.DynamicFeeKeeper.BaseGasPrice(ctx)
	require.Equal(t, math.LegacyNewDecWithPrec(11, 3), baseGasPrice)

	// consume gas
	input.DynamicFeeKeeper.ResetAccumulatedGas(ctx)

	// update base fee
	err = input.DynamicFeeKeeper.UpdateBaseGasPrice(ctx)
	require.NoError(t, err)

	baseGasPrice, err = input.DynamicFeeKeeper.BaseGasPrice(ctx)
	require.Equal(t, math.LegacyNewDecWithPrec(99, 4), baseGasPrice)
}

func Test_AccumulateGas(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	gasLimit := uint64(100000)
	err := input.DynamicFeeKeeper.AccumulateGas(ctx, gasLimit)
	require.NoError(t, err)

	accumulatedGas, err := input.DynamicFeeKeeper.GetAccumulatedGas(ctx)
	require.NoError(t, err)
	require.Equal(t, gasLimit, accumulatedGas)
}
