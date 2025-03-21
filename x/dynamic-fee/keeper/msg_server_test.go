package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/dynamic-fee/keeper"
	"github.com/initia-labs/initia/x/dynamic-fee/types"
)

func Test_UpdateParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	ms := keeper.NewMsgServerImpl(&input.DynamicFeeKeeper)

	msg := &types.MsgUpdateParams{
		Authority: input.DynamicFeeKeeper.GetAuthority(),
		Params: types.Params{
			BaseGasPrice:    math.LegacyNewDecWithPrec(1, 2),
			MinBaseGasPrice: math.LegacyNewDecWithPrec(1, 3),
			MaxBaseGasPrice: math.LegacyNewDec(200),
			MaxChangeRate:   math.LegacyNewDecWithPrec(10, 2),
			TargetGas:       1_000_000,
		},
	}
	_, err := ms.UpdateParams(ctx, msg)
	require.NoError(t, err)

	params, err := input.DynamicFeeKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, msg.Params, params)
}
