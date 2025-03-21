package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
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

	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	ctx.BlockGasMeter().ConsumeGas(100000, "test")

	// update base fee
	err = input.DynamicFeeKeeper.UpdateBaseGasPrice(ctx)
	require.NoError(t, err)

	baseGasPrice, err = input.DynamicFeeKeeper.BaseGasPrice(ctx)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 2), baseGasPrice)

	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	ctx.BlockGasMeter().ConsumeGas(200000, "test")

	// update base fee
	err = input.DynamicFeeKeeper.UpdateBaseGasPrice(ctx)
	require.NoError(t, err)

	baseGasPrice, err = input.DynamicFeeKeeper.BaseGasPrice(ctx)
	require.Equal(t, math.LegacyNewDecWithPrec(11, 3), baseGasPrice)

	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	ctx.BlockGasMeter().ConsumeGas(0, "test")

	// update base fee
	err = input.DynamicFeeKeeper.UpdateBaseGasPrice(ctx)
	require.NoError(t, err)

	baseGasPrice, err = input.DynamicFeeKeeper.BaseGasPrice(ctx)
	require.Equal(t, math.LegacyNewDecWithPrec(99, 4), baseGasPrice)
}
