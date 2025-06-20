package types

import (
	"github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"
)

// VMEngine defines required VM features
type VMEngine interface {
	Initialize(
		kvStore api.KVStore,
		goApi api.GoAPI,
		env vmtypes.Env,
		moduleBundle vmtypes.ModuleBundle,
		allowedPublishers []vmtypes.AccountAddress,
	) (vmtypes.ExecutionResult, error)
	Destroy()
	ExecuteViewFunction(
		gasBalance *uint64,
		kvStore api.KVStore,
		goApi api.GoAPI,
		env vmtypes.Env,
		payload vmtypes.ViewFunction,
	) (vmtypes.ViewOutput, error)
	ExecuteEntryFunction(
		gasBalance *uint64,
		kvStore api.KVStore,
		goApi api.GoAPI,
		env vmtypes.Env,
		senders []vmtypes.AccountAddress,
		payload vmtypes.EntryFunction,
	) (vmtypes.ExecutionResult, error)
	ExecuteScript(
		gasBalance *uint64,
		kvStore api.KVStore,
		goApi api.GoAPI,
		env vmtypes.Env,
		senders []vmtypes.AccountAddress,
		payload vmtypes.Script,
	) (vmtypes.ExecutionResult, error)
	ExecuteAuthenticate(
		gasBalance *uint64,
		kvStore api.KVStore,
		goApi api.GoAPI,
		env vmtypes.Env,
		sender vmtypes.AccountAddress,
		signature []byte,
	) (string, error)
}
