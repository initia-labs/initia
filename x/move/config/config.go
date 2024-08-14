package config

import (
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
)

// DefaultContractSimulationGasLimit - default max simulation gas
const DefaultContractSimulationGasLimit = uint64(3_000_000)

const (
	flagContractSimulationGasLimit = "move.contract-simulation-gas-limit"
)

// MoveConfig is the extra config required for move
type MoveConfig struct {
	ContractSimulationGasLimit uint64 `mapstructure:"contract-simulation-gas-limit"`
}

// DefaultMoveConfig returns the default settings for MoveConfig
func DefaultMoveConfig() MoveConfig {
	return MoveConfig{
		ContractSimulationGasLimit: DefaultContractSimulationGasLimit,
	}
}

// GetConfig load config values from the app options
func GetConfig(appOpts servertypes.AppOptions) MoveConfig {
	return MoveConfig{
		ContractSimulationGasLimit: cast.ToUint64(appOpts.Get(flagContractSimulationGasLimit)),
	}
}

// AddConfigFlags implements servertypes.MoveConfigFlags interface.
func AddConfigFlags(startCmd *cobra.Command) {
	startCmd.Flags().Uint64(flagContractSimulationGasLimit, DefaultContractSimulationGasLimit, "Set the max simulation gas for move contract execution")
}

// DefaultConfigTemplate default config template for move module
const DefaultConfigTemplate = `
###############################################################################
###                         Move                                            ###
###############################################################################

[move]
# The maximum gas amount can be used in a tx simulation call.
contract-simulation-gas-limit = "{{ .MoveConfig.ContractSimulationGasLimit }}"
`
