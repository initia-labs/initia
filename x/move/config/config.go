package config

import (
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
)

// DefaultModuleCacheCapacity the number of modules can be stay in module cache
const DefaultModuleCacheCapacity = uint64(5_000)

// DefaultScriptCacheCapacity the number of modules can be stay in module cache
const DefaultScriptCacheCapacity = uint64(1_000)

// DefaultContractQueryGasLimit - default max query gas for external query
const DefaultContractQueryGasLimit = uint64(3_000_000)

// DefaultContractSimulationGasLimit - default max simulation gas
const DefaultContractSimulationGasLimit = uint64(3_000_000)

// DefaultContractViewBatchLimit - default max view batch limit
const DefaultContractViewBatchLimit = uint64(10)

const (
	flagModuleCacheCapacity        = "move.module-cache-capacity"
	flagScriptCacheCapacity        = "move.script-cache-capacity"
	flagContractSimulationGasLimit = "move.contract-simulation-gas-limit"
	flagContractQueryGasLimit      = "move.contract-query-gas-limit"
	flagContractViewBatchLimit     = "move.contract-view-batch-limit"
)

// MoveConfig is the extra config required for move
type MoveConfig struct {
	ModuleCacheCapacity        uint64 `mapstructure:"module-cache-capacity"`
	ScriptCacheCapacity        uint64 `mapstructure:"script-cache-capacity"`
	ContractSimulationGasLimit uint64 `mapstructure:"contract-simulation-gas-limit"`
	ContractQueryGasLimit      uint64 `mapstructure:"contract-query-gas-limit"`
	ContractViewBatchLimit     uint64 `mapstructure:"contract-view-batch-limit"`
}

// DefaultMoveConfig returns the default settings for MoveConfig
func DefaultMoveConfig() MoveConfig {
	return MoveConfig{
		ModuleCacheCapacity:        DefaultModuleCacheCapacity,
		ScriptCacheCapacity:        DefaultScriptCacheCapacity,
		ContractSimulationGasLimit: DefaultContractSimulationGasLimit,
		ContractQueryGasLimit:      DefaultContractQueryGasLimit,
		ContractViewBatchLimit:     DefaultContractViewBatchLimit,
	}
}

// GetConfig load config values from the app options
func GetConfig(appOpts servertypes.AppOptions) MoveConfig {
	return MoveConfig{
		ModuleCacheCapacity:        cast.ToUint64(appOpts.Get(flagModuleCacheCapacity)),
		ScriptCacheCapacity:        cast.ToUint64(appOpts.Get(flagScriptCacheCapacity)),
		ContractSimulationGasLimit: cast.ToUint64(appOpts.Get(flagContractSimulationGasLimit)),
		ContractQueryGasLimit:      cast.ToUint64(appOpts.Get(flagContractQueryGasLimit)),
		ContractViewBatchLimit:     cast.ToUint64(appOpts.Get(flagContractViewBatchLimit)),
	}
}

// AddConfigFlags implements servertypes.MoveConfigFlags interface.
func AddConfigFlags(startCmd *cobra.Command) {
	startCmd.Flags().Uint64(flagModuleCacheCapacity, DefaultModuleCacheCapacity, "Set the number of modules which can stay in the cache")
	startCmd.Flags().Uint64(flagScriptCacheCapacity, DefaultScriptCacheCapacity, "Set the number of scripts which can stay in the cache")
	startCmd.Flags().Uint64(flagContractSimulationGasLimit, DefaultContractSimulationGasLimit, "Set the max simulation gas for move contract execution")
	startCmd.Flags().Uint64(flagContractQueryGasLimit, DefaultContractQueryGasLimit, "Set the max gas that can be spent on executing a query with a Move contract")
	startCmd.Flags().Uint64(flagContractViewBatchLimit, DefaultContractViewBatchLimit, "Set the maximum number of view function call requests that can be performed by a single ViewBatch gRPC call.")
}

// DefaultConfigTemplate default config template for move module
const DefaultConfigTemplate = `
###############################################################################
###                         Move                                            ###
###############################################################################

[move]
# The number of modules can be live in module cache.
module-cache-capacity = "{{ .MoveConfig.ModuleCacheCapacity }}"

# The number of modules can be live in script cache.
script-cache-capacity = "{{ .MoveConfig.ScriptCacheCapacity }}"

# The maximum gas amount can be used in a tx simulation call.
contract-simulation-gas-limit = "{{ .MoveConfig.ContractSimulationGasLimit }}"

# The maximum gas amount can be spent for contract query.
# The contract query will invoke contract execution vm,
# so we need to restrict the max usage to prevent DoS attack
contract-query-gas-limit = "{{ .MoveConfig.ContractQueryGasLimit }}"

# The maximum number of view function call requests that can 
# be performed by a single ViewBatch gRPC call. Exceeding this 
# limit will result in an error. This limit is set to prevent 
# overloading the system with too many requests at once.
contract-view-batch-limit = "{{ .MoveConfig.ContractViewBatchLimit }}"
`
