package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/move/types"
)

func (k Keeper) acquireVM(ctx context.Context) (vm types.VMEngine) {
	err := k.moveVMSemaphore.Acquire(ctx, 1)
	if err != nil {
		panic(err)
	}

	k.moveVMMutx.Lock()
	idx := *k.moveVMIdx
	*k.moveVMIdx = (idx + 1) % len(k.moveVMs)
	vm = k.moveVMs[idx]
	k.moveVMMutx.Unlock()

	return
}

func (k Keeper) releaseVM() {
	k.moveVMSemaphore.Release(1)
}
