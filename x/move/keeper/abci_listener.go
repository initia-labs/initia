package keeper

import (
	"context"
	"sync"

	"cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	store "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/initia-labs/initia/x/move/types"
)

var _ baseapp.StreamingService = &ABCIListener{}

// ABCIListener is the abci listener to flush move loader cache
// when the publish module operations are failed during
// baseapp.DeliverTx.
type ABCIListener struct {
	newPublishedModulesLoaded bool
	vm                        types.VMEngine
}

func newABCIListener(vm types.VMEngine) ABCIListener {
	return ABCIListener{newPublishedModulesLoaded: false, vm: vm}
}

func (listener *ABCIListener) SetNewPublishedModulesLoaded(loaded bool) {
	listener.newPublishedModulesLoaded = loaded
}

func (listener *ABCIListener) GetNewPublishedModulesLoaded() bool {
	return listener.newPublishedModulesLoaded
}

// ListenDeliverTx updates the steaming service with the latest DeliverTx messages
func (listener *ABCIListener) ListenDeliverTx(ctx context.Context, req abci.RequestDeliverTx, res abci.ResponseDeliverTx) error {
	// When there is no newly published modules, skip below
	if !listener.GetNewPublishedModulesLoaded() {
		return nil
	}

	// reset flag
	listener.SetNewPublishedModulesLoaded(false)

	// check tx failed
	if res.Code != errors.SuccessABCICode {
		// mark loader cache as invalid to flush vm cache
		if err := listener.vm.MarkLoaderCacheAsInvalid(); err != nil {
			return err
		}
	}

	return nil
}

// Stream is the streaming service loop, awaits kv pairs and writes them to some destination stream or file
func (listener *ABCIListener) Stream(wg *sync.WaitGroup) error { return nil }

// Listeners returns the streaming service's listeners for the BaseApp to register
func (listener *ABCIListener) Listeners() map[store.StoreKey][]store.WriteListener { return nil }

// ListenBeginBlock updates the streaming service with the latest BeginBlock messages
func (listener *ABCIListener) ListenBeginBlock(ctx context.Context, req abci.RequestBeginBlock, res abci.ResponseBeginBlock) error {
	return nil
}

// ListenEndBlock updates the steaming service with the latest EndBlock messages
func (listener *ABCIListener) ListenEndBlock(ctx context.Context, req abci.RequestEndBlock, res abci.ResponseEndBlock) error {
	return nil
}

// ListenCommit updates the steaming service with the latest Commit event
func (listener *ABCIListener) ListenCommit(ctx context.Context, res abci.ResponseCommit) error {
	return nil
}

// Closer is the interface that wraps the basic Close method.
func (listener *ABCIListener) Close() error {
	return nil
}
