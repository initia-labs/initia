package keeper

import (
	"context"
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"

	storetypes "cosmossdk.io/store/types"
)

const MaxGasForVerification = 1_000_000

// VerifyAccountAbstractionSignature verifies the signature of an account abstraction transaction.
// It returns the signer which is returned by the authenticate function; for now, it is the same as the sender.
func (k Keeper) VerifyAccountAbstractionSignature(ctx context.Context, sender string, abstractionData vmtypes.AbstractionData) (returnedSigner *vmtypes.AccountAddress, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case storetypes.ErrorOutOfGas:
				// propagate out of gas error
				panic(r)
			default:
				k.Logger(ctx).Error("panic in VerifyAccountAbstractionSignature", "error", r)
				err = errors.New("panic in VerifyAccountAbstractionSignature occurred")
			}
		}
	}()

	signer, err := types.AccAddressFromString(k.ac, sender)
	if err != nil {
		return nil, err
	}

	ac := types.NextAccountNumber(ctx, k.authKeeper)
	ec, err := k.ExecutionCounter.Next(ctx)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	gasMeter := sdkCtx.GasMeter()
	gasForRuntime := min(MaxGasForVerification, k.computeGasForRuntime(ctx, gasMeter))

	// delegate gas metering to move vm
	sdkCtx = sdkCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	gasBalance := gasForRuntime
	returnedSigner, err = k.initiaMoveVM.ExecuteAuthenticate(
		&gasBalance,
		types.NewVMStore(sdkCtx, k.VMStore),
		NewApi(k, sdkCtx),
		types.NewEnv(sdkCtx, ac, ec),
		signer,
		abstractionData,
	)

	// consume gas first and check error
	gasUsed := gasForRuntime - gasBalance
	gasMeter.ConsumeGas(gasUsed, "verify account abstraction signature")
	if err != nil {
		return nil, err
	}

	return returnedSigner, nil
}
