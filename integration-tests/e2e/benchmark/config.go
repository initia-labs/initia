package benchmark

// Variant identifies which optimization layer is being benchmarked.
//
// The three-way comparison measures improvements from:
//  1. Baseline: CListMempool + standard IAVL (pre-proxy cometbft tag)
//  2. MempoolOnly: ProxyMempool + PriorityMempool + standard IAVL
//  3. Combined: ProxyMempool + PriorityMempool + MemIAVL
//
// Baseline results are obtained by checking out the pre-proxy cometbft tag,
// rebuilding initiad, and running the same benchmark harness. Those results
// are then loaded from JSON for cross-run comparison.
type Variant string

const (
	VariantBaseline    Variant = "baseline"     // CListMempool + IAVL
	VariantMempoolOnly Variant = "mempool-only" // ProxyMempool+PriorityMempool + IAVL
	VariantCombined    Variant = "combined"     // ProxyMempool+PriorityMempool + MemIAVL
)

const defaultGasLimit uint64 = 500_000

// BenchConfig defines the parameters for a benchmark run.
type BenchConfig struct {
	MemIAVL      bool    `json:"memiavl"`
	NodeCount    int     `json:"node_count"`
	AccountCount int     `json:"account_count"`
	TxPerAccount int     `json:"tx_per_account"`
	GasLimit     uint64  `json:"gas_limit"`
	Label        string  `json:"label"`
	Variant      Variant `json:"variant"`
}

// GetGasLimit returns the configured gas limit, falling back to defaultGasLimit.
func (c BenchConfig) GetGasLimit() uint64 {
	if c.GasLimit == 0 {
		return defaultGasLimit
	}
	return c.GasLimit
}

// MempoolOnlyConfig returns a benchmark configuration for the mempool-only improvement layer.
// Uses ProxyMempool+PriorityMempool with standard IAVL.
func MempoolOnlyConfig() BenchConfig {
	return BenchConfig{
		MemIAVL:      false,
		NodeCount:    3,
		AccountCount: 10,
		TxPerAccount: 200,
		Label:        "proxy+priority/iavl",
		Variant:      VariantMempoolOnly,
	}
}

// CombinedConfig returns a benchmark configuration for the combined improvement layer.
// Uses ProxyMempool+PriorityMempool with MemIAVL.
func CombinedConfig() BenchConfig {
	return BenchConfig{
		MemIAVL:      true,
		NodeCount:    3,
		AccountCount: 10,
		TxPerAccount: 200,
		Label:        "proxy+priority/memiavl",
		Variant:      VariantCombined,
	}
}

// BaselineConfig returns a benchmark configuration labeled as baseline.
// This config itself runs with the current binary — to get true baseline results,
// rebuild initiad from the pre-proxy cometbft tag and pass it via E2E_INITIAD_BIN.
func BaselineConfig() BenchConfig {
	return BenchConfig{
		MemIAVL:      false,
		NodeCount:    3,
		AccountCount: 10,
		TxPerAccount: 200,
		Label:        "clist/iavl",
		Variant:      VariantBaseline,
	}
}

// DefaultConfig is an alias for MempoolOnlyConfig for backward compatibility.
func DefaultConfig() BenchConfig {
	return MempoolOnlyConfig()
}

// MemIAVLConfig is an alias for CombinedConfig for backward compatibility.
func MemIAVLConfig() BenchConfig {
	return CombinedConfig()
}

// TotalTx returns the total number of transactions for this config.
func (c BenchConfig) TotalTx() int {
	return c.AccountCount * c.TxPerAccount
}
