package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"

	storetypes "cosmossdk.io/store/types"
)

// VerifyAccountAbstractionSignature verifies the signature of an account abstraction transaction.
// It returns the signer which is returned by the authenticate function; for now, it is the same as the sender.
func (k Keeper) VerifyAccountAbstractionSignature(ctx context.Context, sender string, abstractionData vmtypes.AbstractionData) (res string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %v", r)
		}
	}()

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
		abstractionData,
	)
}
