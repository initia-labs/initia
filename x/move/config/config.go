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

// DefaultContractSimulationGasLimit - default max simulation gas
const DefaultContractSimulationGasLimit = uint64(3_000_000)

const (
	flagModuleCacheCapacity        = "move.module-cache-capacity"
	flagScriptCacheCapacity        = "move.script-cache-capacity"
	flagContractSimulationGasLimit = "move.contract-simulation-gas-limit"
)

// MoveConfig is the extra config required for move
type MoveConfig struct {
	ModuleCacheCapacity        uint64 `mapstructure:"module-cache-capacity"`
	ScriptCacheCapacity        uint64 `mapstructure:"script-cache-capacity"`
	ContractSimulationGasLimit uint64 `mapstructure:"contract-simulation-gas-limit"`
}

// DefaultMoveConfig returns the default settings for MoveConfig
func DefaultMoveConfig() MoveConfig {
	return MoveConfig{
		ModuleCacheCapacity:        DefaultModuleCacheCapacity,
		ScriptCacheCapacity:        DefaultScriptCacheCapacity,
		ContractSimulationGasLimit: DefaultContractSimulationGasLimit,
	}
}

// GetConfig load config values from the app options
func GetConfig(appOpts servertypes.AppOptions) MoveConfig {
	return MoveConfig{
		ModuleCacheCapacity:        cast.ToUint64(appOpts.Get(flagModuleCacheCapacity)),
		ScriptCacheCapacity:        cast.ToUint64(appOpts.Get(flagScriptCacheCapacity)),
		ContractSimulationGasLimit: cast.ToUint64(appOpts.Get(flagContractSimulationGasLimit)),
	}
}

// AddConfigFlags implements servertypes.MoveConfigFlags interface.
func AddConfigFlags(startCmd *cobra.Command) {
	startCmd.Flags().Uint64(flagModuleCacheCapacity, DefaultModuleCacheCapacity, "Set the number of modules which can stay in the cache")
	startCmd.Flags().Uint64(flagScriptCacheCapacity, DefaultScriptCacheCapacity, "Set the number of scripts which can stay in the cache")
	startCmd.Flags().Uint64(flagContractSimulationGasLimit, DefaultContractSimulationGasLimit, "Set the max simulation gas for move contract execution")
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
`
