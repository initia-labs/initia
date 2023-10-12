package types

const (
	defaultSmartQueryGasLimit uint64 = 3_000_000
)

type MoveConfig struct {
	// SimulationGasLimit is the max gas to be used in a tx simulation call.
	// When not set the consensus max block gas is used instead
	SimulationGasLimit *uint64
	// SimulationGasLimit is the max gas to be used in a smart query contract call
	SmartQueryGasLimit uint64
}

// DefaultMoveConfig returns the default settings for MoveConfig
func DefaultMoveConfig() MoveConfig {
	return MoveConfig{
		SmartQueryGasLimit: defaultSmartQueryGasLimit,
	}
}
