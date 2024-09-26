package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/move/types"
)

func (k Keeper) acquireVM(ctx context.Context) (vm types.VMEngine) {
	k.moveVMSemaphore.Acquire(ctx, 1)

	k.moveVMMutx.Lock()
	vm, *k.moveVMs = (*k.moveVMs)[0], (*k.moveVMs)[1:]
	k.moveVMMutx.Unlock()

	return
}

func (k Keeper) releaseVM(vm types.VMEngine) {
	k.moveVMMutx.Lock()
	*k.moveVMs = append(*k.moveVMs, vm)
	k.moveVMMutx.Unlock()

	k.moveVMSemaphore.Release(1)
}
