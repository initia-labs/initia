package keeper

import (
	"context"
	"sync/atomic"

	"github.com/initia-labs/initia/x/move/types"
)

func (k Keeper) acquireVM(ctx context.Context) (vm types.VMEngine) {
	err := k.moveVMSemaphore.Acquire(ctx, 1)
	if err != nil {
		panic(err)
	}

	idx := atomic.AddUint64(k.moveVMIdx, 1)
	vm = k.moveVMs[idx%uint64(len(k.moveVMs))]

	return
}

func (k Keeper) releaseVM() {
	k.moveVMSemaphore.Release(1)
}
