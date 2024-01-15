package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/types"
)

// PostHandler like AnteHandler but it executes after RunMsgs. Runs on success
// or failure and enables use cases like gas refunding.
type PostHandler struct {
	newPublishedModulesLoaded bool
	vm                        types.VMEngine
}

func newPostHandler(vm types.VMEngine) PostHandler {
	return PostHandler{newPublishedModulesLoaded: false, vm: vm}
}

func (listener *PostHandler) SetNewPublishedModulesLoaded(loaded bool) {
	listener.newPublishedModulesLoaded = loaded
}

func (listener *PostHandler) GetNewPublishedModulesLoaded() bool {
	return listener.newPublishedModulesLoaded
}

// ListenDeliverTx updates the steaming service with the latest DeliverTx messages
func (listener *PostHandler) PostHandle(ctx sdk.Context, tx sdk.Tx, simulate, success bool) (newCtx sdk.Context, err error) {
	// When there is no newly published modules, skip below
	if !listener.GetNewPublishedModulesLoaded() {
		return ctx, nil
	}

	// reset flag
	listener.SetNewPublishedModulesLoaded(false)

	// check tx failed
	if !success {
		// mark loader cache as invalid to flush vm cache
		if err := listener.vm.MarkLoaderCacheAsInvalid(); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}
