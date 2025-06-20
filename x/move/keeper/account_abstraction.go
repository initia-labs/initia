package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/move/types"

	storetypes "cosmossdk.io/store/types"
)

func (k Keeper) VerifyAccountAbstractionSignature(ctx context.Context, sender string, signature []byte) (string, error) {
	signer, err := types.AccAddressFromString(k.ac, sender)
	if err != nil {
		return "", err
	}

	ac := types.NextAccountNumber(ctx, k.authKeeper)
	ec, err := k.ExecutionCounter.Next(ctx)
	if err != nil {
		return "", err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	gasMeter := sdkCtx.GasMeter()
	gasForRuntime := k.computeGasForRuntime(ctx, gasMeter)

	// delegate gas metering to move vm
	sdkCtx = sdkCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	gasBalance := gasForRuntime
	return k.initiaMoveVM.ExecuteAuthenticate(
		&gasBalance,
		types.NewVMStore(sdkCtx, k.VMStore),
		NewApi(k, sdkCtx),
		types.NewEnv(sdkCtx, ac, ec),
		signer,
		signature,
	)
}
