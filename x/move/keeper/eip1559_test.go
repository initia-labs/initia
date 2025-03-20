package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/initia-labs/initia/x/move/keeper"
	"github.com/initia-labs/initia/x/move/types"
	"github.com/stretchr/testify/require"
)

func Test_UpdateBaseFee(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	eip1559FeeKeeper := keeper.NewEIP1559FeeKeeper(&input.MoveKeeper)

	err := eip1559FeeKeeper.SetParams(ctx, types.EIP1559FeeParams{
		MinBaseFee:    10,
		MaxBaseFee:    200,
		BaseFee:       100,
		TargetGas:     100000,
		MaxChangeRate: math.LegacyNewDecWithPrec(1, 1),
	})
	require.NoError(t, err)

	baseFee, err := eip1559FeeKeeper.GetBaseFee(ctx)
	require.Equal(t, int64(100), baseFee)

	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	ctx.BlockGasMeter().ConsumeGas(100000, "test")

	// update base fee
	err = eip1559FeeKeeper.UpdateBaseFee(ctx)
	require.NoError(t, err)

	baseFee, err = eip1559FeeKeeper.GetBaseFee(ctx)
	require.Equal(t, int64(100), baseFee)

	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	ctx.BlockGasMeter().ConsumeGas(200000, "test")

	// update base fee
	err = eip1559FeeKeeper.UpdateBaseFee(ctx)
	require.NoError(t, err)

	baseFee, err = eip1559FeeKeeper.GetBaseFee(ctx)
	require.Equal(t, int64(110), baseFee)

	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	ctx.BlockGasMeter().ConsumeGas(0, "test")

	// update base fee
	err = eip1559FeeKeeper.UpdateBaseFee(ctx)
	require.NoError(t, err)

	baseFee, err = eip1559FeeKeeper.GetBaseFee(ctx)
	require.Equal(t, int64(99), baseFee)
}
