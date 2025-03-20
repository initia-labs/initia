package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/move/types"

	"cosmossdk.io/math"
)

type EIP1559FeeKeeper struct {
	*Keeper
}

func NewEIP1559FeeKeeper(moveKeeper *Keeper) EIP1559FeeKeeper {
	return EIP1559FeeKeeper{
		moveKeeper,
	}
}

func (k EIP1559FeeKeeper) SetParams(ctx context.Context, params types.EIP1559FeeParams) error {
	return k.EIP1559FeeParams.Set(ctx, params)
}

func (k EIP1559FeeKeeper) GetParams(ctx context.Context) (types.EIP1559FeeParams, error) {
	return k.EIP1559FeeParams.Get(ctx)
}

func (k EIP1559FeeKeeper) GetBaseFee(ctx context.Context) (int64, error) {
	params, err := k.EIP1559FeeParams.Get(ctx)
	if err != nil {
		return 0, err
	}

	return params.BaseFee, nil
}

// this should be called in EndBlocker
func (k EIP1559FeeKeeper) UpdateBaseFee(ctx sdk.Context) error {
	params, err := k.EIP1559FeeParams.Get(ctx)
	if err != nil {
		return err
	}

	gasUsed := ctx.BlockGasMeter().GasConsumed()

	// baseFeeMultiplier = (gasUsed - targetGas) / targetGas * maxChangeRate + 1
	baseFeeMultiplier := math.LegacyNewDec(int64(gasUsed) - params.TargetGas).QuoInt64(params.TargetGas).Mul(params.MaxChangeRate).Add(math.OneInt().ToLegacyDec())
	newBaseFee := math.LegacyNewDec(params.BaseFee).Mul(baseFeeMultiplier).TruncateInt64()
	newBaseFee = math.Max(newBaseFee, params.MinBaseFee)
	newBaseFee = math.Min(newBaseFee, params.MaxBaseFee)
	params.BaseFee = newBaseFee

	err = k.EIP1559FeeParams.Set(ctx, params)
	if err != nil {
		return err
	}
	return nil
}
