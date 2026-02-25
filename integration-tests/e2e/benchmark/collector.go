package benchmark

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/initia-labs/initia/integration-tests/e2e"
)

// BlockStat holds per-block statistics.
type BlockStat struct {
	Height  int64     `json:"height"`
	TxCount int       `json:"tx_count"`
	Time    time.Time `json:"time"`
}

// BenchResult holds the aggregated benchmark results.
type BenchResult struct {
	Config          BenchConfig `json:"config"`
	TotalSubmitted  int         `json:"total_submitted"`
	TotalIncluded   int         `json:"total_included"`
	DurationSec     float64     `json:"duration_sec"`
	TxPerSecond     float64     `json:"tx_per_second"`
	AvgLatencyMs    float64     `json:"avg_latency_ms"`
	P50LatencyMs    float64     `json:"p50_latency_ms"`
	P95LatencyMs    float64     `json:"p95_latency_ms"`
	P99LatencyMs    float64     `json:"p99_latency_ms"`
	MaxLatencyMs    float64     `json:"max_latency_ms"`
	PeakMempoolSize int         `json:"peak_mempool_size"`
	BlockStats      []BlockStat `json:"block_stats"`
}

// MempoolPoller polls mempool size at regular intervals and tracks the peak.
type MempoolPoller struct {
	cluster  *e2e.Cluster
	interval time.Duration
	peak     atomic.Int64
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewMempoolPoller creates and starts a mempool size poller.
func NewMempoolPoller(ctx context.Context, cluster *e2e.Cluster, interval time.Duration) *MempoolPoller {
	pollCtx, cancel := context.WithCancel(ctx)
	p := &MempoolPoller{
		cluster:  cluster,
		interval: interval,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
	go p.run(pollCtx)

	return p
}

func (p *MempoolPoller) run(ctx context.Context) {
	defer close(p.done)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// we poll all nodes and take the max
			var maxSize int64
			for i := 0; i < p.cluster.NodeCount(); i++ {
				n, err := p.cluster.UnconfirmedTxCount(ctx, i)
				if err != nil {
					continue
				}
				if n > maxSize {
					maxSize = n
				}
			}
			for {
				old := p.peak.Load()
				if maxSize <= old {
					break
				}
				if p.peak.CompareAndSwap(old, maxSize) {
					break
				}
			}
		}
	}
}

// Stop stops the poller and returns the peak mempool size observed.
func (p *MempoolPoller) Stop() int {
	p.cancel()
	<-p.done

	return int(p.peak.Load())
}

// CollectResults iterates blocks from startHeight to endHeight, matches, and computes aggregate metrics.
func CollectResults(
	ctx context.Context,
	cluster *e2e.Cluster,
	cfg BenchConfig,
	load LoadResult,
	startHeight, endHeight int64,
	peakMempool int,
) (BenchResult, error) {
	submitMap := make(map[string]TxSubmission, len(load.Submissions))
	for _, sub := range load.Submissions {
		if sub.TxHash != "" {
			submitMap[sub.TxHash] = sub
		}
	}

	var (
		blockStats []BlockStat
		latencies  []float64
		included   int
	)

	for h := startHeight; h <= endHeight; h++ {
		block, err := cluster.QueryBlock(ctx, 0, h)
		if err != nil {
			return BenchResult{}, fmt.Errorf("query block %d: %w", h, err)
		}

		bs := BlockStat{
			Height:  h,
			TxCount: len(block.TxHashes),
			Time:    block.BlockTime,
		}
		blockStats = append(blockStats, bs)

		for _, txHash := range block.TxHashes {
			sub, ok := submitMap[txHash]
			if !ok {
				continue
			}
			included++
			latencyMs := float64(block.BlockTime.Sub(sub.SubmitTime).Milliseconds())
			if latencyMs < 0 {
				latencyMs = 0
			}
			latencies = append(latencies, latencyMs)
		}
	}

	totalDuration := load.EndTime.Sub(load.StartTime).Seconds()
	if totalDuration <= 0 {
		totalDuration = 1
	}

	// use span from first to last block that actually included benchmark txs
	var firstTxTime, lastTxTime time.Time
	for _, bs := range blockStats {
		if bs.TxCount == 0 {
			continue
		}
		if firstTxTime.IsZero() {
			firstTxTime = bs.Time
		}
		lastTxTime = bs.Time
	}
	if !firstTxTime.IsZero() {
		blockDuration := lastTxTime.Sub(firstTxTime).Seconds()
		if blockDuration > 0 {
			totalDuration = blockDuration
		}
	}

	tps := float64(included) / totalDuration

	sort.Float64s(latencies)

	result := BenchResult{
		Config:          cfg,
		TotalSubmitted:  len(load.Submissions),
		TotalIncluded:   included,
		DurationSec:     math.Round(totalDuration*100) / 100,
		TxPerSecond:     math.Round(tps*10) / 10,
		PeakMempoolSize: peakMempool,
		BlockStats:      blockStats,
	}

	if len(latencies) > 0 {
		result.AvgLatencyMs = math.Round(avg(latencies)*10) / 10
		result.P50LatencyMs = math.Round(percentile(latencies, 50)*10) / 10
		result.P95LatencyMs = math.Round(percentile(latencies, 95)*10) / 10
		result.P99LatencyMs = math.Round(percentile(latencies, 99)*10) / 10
		result.MaxLatencyMs = latencies[len(latencies)-1]
	}

	return result, nil
}

// WaitForAllIncluded waits until the mempool is empty and all submitted txs
// are included in blocks, returning the end height.
func WaitForAllIncluded(ctx context.Context, cluster *e2e.Cluster, timeout time.Duration) (int64, error) {
	if err := cluster.WaitForMempoolEmpty(ctx, timeout); err != nil {
		return 0, err
	}
	// Wait one more block to ensure finality
	time.Sleep(3 * time.Second)
	h, err := cluster.LatestHeight(ctx, 0)
	if err != nil {
		return 0, err
	}
	return h, nil
}

// CollectInitialMetas queries account metadata for all accounts.
func CollectInitialMetas(ctx context.Context, cluster *e2e.Cluster) (map[string]e2e.AccountMeta, error) {
	metas := make(map[string]e2e.AccountMeta)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error

	for _, name := range cluster.AccountNames() {
		name := name
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr, err := cluster.AccountAddress(name)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			meta, err := cluster.QueryAccountMeta(ctx, 0, addr)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			mu.Lock()
			metas[name] = meta
			mu.Unlock()
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	return metas, nil
}

func avg(sorted []float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	var sum float64
	for _, v := range sorted {
		sum += v
	}

	return sum / float64(len(sorted))
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}
