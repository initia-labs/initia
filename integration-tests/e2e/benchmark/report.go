package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// WriteResult writes a BenchResult as JSON to the results directory.
func WriteResult(t *testing.T, result BenchResult, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	safe := strings.NewReplacer("/", "_", " ", "_", "+", "_").Replace(result.Config.Label)
	path := filepath.Join(dir, safe+".json")

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	t.Logf("Writing benchmark result to %s", path)
	return os.WriteFile(path, data, 0o600)
}

// PrintComparisonTable prints a formatted comparison table for multiple results.
func PrintComparisonTable(t *testing.T, results []BenchResult) {
	header := fmt.Sprintf("%-25s | %8s | %7s | %7s | %7s | %7s | %12s",
		"Config", "Variant", "TPS", "P50ms", "P95ms", "P99ms", "Peak Mempool")
	sep := strings.Repeat("-", len(header))

	t.Log(sep)
	t.Log(header)
	t.Log(sep)

	for _, r := range results {
		t.Logf("%-25s | %8s | %7.1f | %7.0f | %7.0f | %7.0f | %12d",
			r.Config.Label, r.Config.Variant, r.TxPerSecond,
			r.P50LatencyMs, r.P95LatencyMs, r.P99LatencyMs,
			r.PeakMempoolSize)
	}
	t.Log(sep)
}

// PrintImprovementTable prints a comparison table with delta percentages against a baseline.
// The baseline is the first result with VariantBaseline, or the first result if none is marked.
//
// Output format:
//
//	Config                    | Variant  |     TPS | vs base |   P50ms | vs base |   P95ms | Peak Mempool
//	clist/iavl                | baseline |   120.5 |       - |    2500 |       - |    4800 |         1950
//	proxy+priority/iavl       | mempool  |   245.3 | +103.6% |    1823 |  -27.1% |    3412 |         1847
//	proxy+priority/memiavl    | combined |   312.7 | +159.5% |    1401 |  -44.0% |    2845 |         1823
func PrintImprovementTable(t *testing.T, results []BenchResult) {
	if len(results) == 0 {
		return
	}

	var baseline *BenchResult
	for i := range results {
		if results[i].Config.Variant == VariantBaseline {
			baseline = &results[i]
			break
		}
	}
	if baseline == nil {
		baseline = &results[0]
	}

	header := fmt.Sprintf("%-25s | %-12s | %7s | %8s | %7s | %8s | %7s | %8s | %12s",
		"Config", "Variant", "TPS", "vs base", "P50ms", "vs base", "P95ms", "vs base", "Peak Mempool")
	sep := strings.Repeat("-", len(header))

	t.Log("")
	t.Log(sep)
	t.Log(header)
	t.Log(sep)

	for _, r := range results {
		tpsDelta := deltaStr(baseline.TxPerSecond, r.TxPerSecond)
		p50Delta := deltaStr(baseline.P50LatencyMs, r.P50LatencyMs)
		p95Delta := deltaStr(baseline.P95LatencyMs, r.P95LatencyMs)

		if r.Config.Variant == baseline.Config.Variant {
			tpsDelta = "-"
			p50Delta = "-"
			p95Delta = "-"
		}

		t.Logf("%-25s | %-12s | %7.1f | %8s | %7.0f | %8s | %7.0f | %8s | %12d",
			r.Config.Label, r.Config.Variant, r.TxPerSecond, tpsDelta,
			r.P50LatencyMs, p50Delta, r.P95LatencyMs, p95Delta,
			r.PeakMempoolSize)
	}
	t.Log(sep)
	t.Log("")
}

// deltaStr computes a percentage change string.
func deltaStr(base, current float64) string {
	if base == 0 {
		return "N/A"
	}
	pct := (current - base) / base * 100
	sign := "+"
	if pct < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.1f%%", sign, pct)
}

// LoadResults reads all JSON result files from a directory.
func LoadResults(dir string) ([]BenchResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var results []BenchResult
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		var r BenchResult
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		results = append(results, r)
	}

	return results, nil
}

// LoadBaselineResults loads baseline results from a directory if available.
// Returns nil if the directory doesn't exist or contains no baseline results.
func LoadBaselineResults(dir string) []BenchResult {
	results, err := LoadResults(dir)
	if err != nil {
		return nil
	}

	var baselines []BenchResult
	for _, r := range results {
		if r.Config.Variant == VariantBaseline {
			baselines = append(baselines, r)
		}
	}

	return baselines
}

// LoadBaselineResultsByLabel loads baseline results matching a specific label.
// This allows distinguishing between different baseline runs (e.g., "clist/iavl/seq" vs "clist/iavl/burst").
func LoadBaselineResultsByLabel(dir, label string) []BenchResult {
	results, err := LoadResults(dir)
	if err != nil {
		return nil
	}

	var matched []BenchResult
	for _, r := range results {
		if r.Config.Variant == VariantBaseline && r.Config.Label == label {
			matched = append(matched, r)
		}
	}

	return matched
}
