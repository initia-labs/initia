package config

import (
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
)

// DefaultCacheCapacity the number of modules can be stay in module cache
const DefaultCacheCapacity = uint64(1_000)

// DefaultContractQueryGasLimit - default max query gas for external query
const DefaultContractQueryGasLimit = uint64(3_000_000)

// DefaultContractSimulationGasLimit - default max simulation gas
const DefaultContractSimulationGasLimit = uint64(3_000_000)

const (
	flagCacheCapacity              = "move.cache-capacity"
	flagContractSimulationGasLimit = "move.contract-simulation-gas-limit"
	flagContractQueryGasLimit      = "move.contract-query-gas-limit"
)

// MoveConfig is the extra config required for move
type MoveConfig struct {
	CacheCapacity              uint64 `mapstructure:"cache-capacity"`
	ContractSimulationGasLimit uint64 `mapstructure:"contract-simulation-gas-limit"`
	ContractQueryGasLimit      uint64 `mapstructure:"contract-query-gas-limit"`
}

// DefaultMoveConfig returns the default settings for MoveConfig
func DefaultMoveConfig() MoveConfig {
	return MoveConfig{
		CacheCapacity:              DefaultCacheCapacity,
		ContractSimulationGasLimit: DefaultContractSimulationGasLimit,
		ContractQueryGasLimit:      DefaultContractQueryGasLimit,
	}
}

// GetConfig load config values from the app options
func GetConfig(appOpts servertypes.AppOptions) MoveConfig {
	return MoveConfig{
		CacheCapacity:              cast.ToUint64(appOpts.Get(flagCacheCapacity)),
		ContractSimulationGasLimit: cast.ToUint64(appOpts.Get(flagContractSimulationGasLimit)),
		ContractQueryGasLimit:      cast.ToUint64(appOpts.Get(flagContractQueryGasLimit)),
	}
}

// AddConfigFlags implements servertypes.MoveConfigFlags interface.
func AddConfigFlags(startCmd *cobra.Command) {
	startCmd.Flags().Uint64(flagCacheCapacity, DefaultCacheCapacity, "Set the max simulation gas for move contract execution")
	startCmd.Flags().Uint64(flagContractSimulationGasLimit, DefaultContractSimulationGasLimit, "Set the max simulation gas for move contract execution")
	startCmd.Flags().Uint64(flagContractQueryGasLimit, DefaultContractQueryGasLimit, "Set the max gas that can be spent on executing a query with a Move contract")
}

// DefaultConfigTemplate default config template for move module
const DefaultConfigTemplate = `
###############################################################################
###                         Move                                            ###
###############################################################################

[move]
# The number of modules can be live in module cache.
cache-capacity = "{{ .MoveConfig.CacheCapacity }}"

# The maximum gas amount can be used in a tx simulation call.
contract-simulation-gas-limit = "{{ .MoveConfig.ContractSimulationGasLimit }}"

# The maximum gas amount can be spent for contract query.
# The contract query will invoke contract execution vm,
# so we need to restrict the max usage to prevent DoS attack
contract-query-gas-limit = "{{ .MoveConfig.ContractQueryGasLimit }}"
`
