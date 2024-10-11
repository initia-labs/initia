package config

import (
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
)

// DefaultContractSimulationGasLimit - default max simulation gas
const DefaultContractSimulationGasLimit = uint64(3_000_000)
const DefaultScriptCacheCapacity = uint64(100)
const DefaultModuleCacheCapacity = uint64(500)

const (
	flagContractSimulationGasLimit = "move.contract-simulation-gas-limit"
	flagScriptCacheCapacity        = "move.script-cache-capacity"
	flagModuleCacheCapacity        = "move.module-cache-capacity"
)

// MoveConfig is the extra config required for move
type MoveConfig struct {
	ContractSimulationGasLimit uint64 `mapstructure:"contract-simulation-gas-limit"`
	ScriptCacheCapacity        uint64 `mapstructure:"script-cache-capacity"`
	ModuleCacheCapacity        uint64 `mapstructure:"module-cache-capacity"`
}

// DefaultMoveConfig returns the default settings for MoveConfig
func DefaultMoveConfig() MoveConfig {
	return MoveConfig{
		ContractSimulationGasLimit: DefaultContractSimulationGasLimit,
		ScriptCacheCapacity:        DefaultScriptCacheCapacity,
		ModuleCacheCapacity:        DefaultModuleCacheCapacity,
	}
}

// GetConfig load config values from the app options
func GetConfig(appOpts servertypes.AppOptions) MoveConfig {
	return MoveConfig{
		ContractSimulationGasLimit: cast.ToUint64(appOpts.Get(flagContractSimulationGasLimit)),
		ScriptCacheCapacity:        cast.ToUint64(appOpts.Get(flagScriptCacheCapacity)),
		ModuleCacheCapacity:        cast.ToUint64(appOpts.Get(flagModuleCacheCapacity)),
	}
}

// AddConfigFlags implements servertypes.MoveConfigFlags interface.
func AddConfigFlags(startCmd *cobra.Command) {
	startCmd.Flags().Uint64(flagContractSimulationGasLimit, DefaultContractSimulationGasLimit, "Set the max simulation gas for move contract execution")
	startCmd.Flags().Uint64(flagScriptCacheCapacity, DefaultScriptCacheCapacity, "Set the script cache capacity")
	startCmd.Flags().Uint64(flagModuleCacheCapacity, DefaultModuleCacheCapacity, "Set the module cache capacity")
}

// DefaultConfigTemplate default config template for move module
const DefaultConfigTemplate = `
###############################################################################
###                         Move                                            ###
###############################################################################

[move]
# The maximum gas amount can be used in a tx simulation call.
contract-simulation-gas-limit = "{{ .MoveConfig.ContractSimulationGasLimit }}"
# The capacity of the script cache.
script-cache-capacity = "{{ .MoveConfig.ScriptCacheCapacity }}"
# The capacity of the module cache.
module-cache-capacity = "{{ .MoveConfig.ModuleCacheCapacity }}"
`
