package types

import (
	"github.com/initia-labs/initiavm/api"
	vmtypes "github.com/initia-labs/initiavm/types"
)

// VMEngine defines required VM features
type VMEngine interface {
	Initialize(
		kvStore api.KVStore,
		goApi api.GoAPI,
		env vmtypes.Env,
		moduleBundle vmtypes.ModuleBundle,
		allowArbitrary bool,
		allowedPublishers []vmtypes.AccountAddress,
	) error
	Destroy()
	ExecuteViewFunction(
		kvStore api.KVStore,
		goApi api.GoAPI,
		env vmtypes.Env,
		gasLimit uint64,
		payload vmtypes.ViewFunction,
	) (string, error)
	ExecuteEntryFunction(
		kvStore api.KVStore,
		goApi api.GoAPI,
		env vmtypes.Env,
		gasLimit uint64,
		senders []vmtypes.AccountAddress,
		payload vmtypes.EntryFunction,
	) (vmtypes.ExecutionResult, error)
	ExecuteScript(
		kvStore api.KVStore,
		goApi api.GoAPI,
		env vmtypes.Env,
		gasLimit uint64,
		senders []vmtypes.AccountAddress,
		payload vmtypes.Script,
	) (vmtypes.ExecutionResult, error)
	MarkLoaderCacheAsInvalid() error
}
