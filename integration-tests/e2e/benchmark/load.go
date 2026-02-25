package benchmark

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/initia-labs/initia/integration-tests/e2e"
)

// TxSubmission records a single submitted transaction.
type TxSubmission struct {
	TxHash     string    `json:"tx_hash"`
	Account    string    `json:"account"`
	Sequence   uint64    `json:"sequence"`
	SubmitTime time.Time `json:"submit_time"`
	ViaNode    int       `json:"via_node"`
}

// LoadResult holds the outcome of a load generation run.
type LoadResult struct {
	Submissions []TxSubmission
	Errors      []error
	StartTime   time.Time
	EndTime     time.Time
}

// BurstLoad submits all transactions concurrently across accounts with sequential nonces.
func BurstLoad(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		result LoadResult
	)

	result.StartTime = time.Now()

	for _, name := range cluster.AccountNames() {
		name := name
		meta := metas[name]

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < cfg.TxPerAccount; i++ {
				seq := meta.Sequence + uint64(i)
				viaNode := i % cfg.NodeCount
				submitTime := time.Now()

				res := cluster.SendBankTxWithSequence(
					ctx, name, cluster.ValidatorAddress(), "1uinit",
					meta.AccountNumber, seq, cfg.GetGasLimit(), viaNode,
				)

				sub := TxSubmission{
					TxHash:     res.TxHash,
					Account:    name,
					Sequence:   seq,
					SubmitTime: submitTime,
					ViaNode:    viaNode,
				}

				mu.Lock()
				result.Submissions = append(result.Submissions, sub)
				if res.Err != nil {
					result.Errors = append(result.Errors, res.Err)
				} else if res.Code != 0 {
					// still record but note the non-zero code
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	result.EndTime = time.Now()

	return result
}

// OutOfOrderLoad submits the first 3 txs per account with out-of-order nonces
// (seq+2, seq+0, seq+1), then the rest sequentially.
func OutOfOrderLoad(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		result LoadResult
	)

	result.StartTime = time.Now()

	for _, name := range cluster.AccountNames() {
		name := name
		meta := metas[name]

		wg.Add(1)
		go func() {
			defer wg.Done()
			seqs := sequencePattern(meta.Sequence, cfg.TxPerAccount)
			for i, seq := range seqs {
				viaNode := i % cfg.NodeCount
				submitTime := time.Now()

				res := cluster.SendBankTxWithSequence(
					ctx, name, cluster.ValidatorAddress(), "1uinit",
					meta.AccountNumber, seq, cfg.GetGasLimit(), viaNode,
				)

				sub := TxSubmission{
					TxHash:     res.TxHash,
					Account:    name,
					Sequence:   seq,
					SubmitTime: submitTime,
					ViaNode:    viaNode,
				}

				mu.Lock()
				result.Submissions = append(result.Submissions, sub)
				if res.Err != nil {
					result.Errors = append(result.Errors, res.Err)
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	result.EndTime = time.Now()

	return result
}

// SingleNodeLoad submits all transactions to a single specified node.
func SingleNodeLoad(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta, targetNode int) LoadResult {
	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		result LoadResult
	)

	result.StartTime = time.Now()

	for _, name := range cluster.AccountNames() {
		name := name
		meta := metas[name]

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < cfg.TxPerAccount; i++ {
				seq := meta.Sequence + uint64(i)
				submitTime := time.Now()

				res := cluster.SendBankTxWithSequence(
					ctx, name, cluster.ValidatorAddress(), "1uinit",
					meta.AccountNumber, seq, cfg.GetGasLimit(), targetNode,
				)

				sub := TxSubmission{
					TxHash:     res.TxHash,
					Account:    name,
					Sequence:   seq,
					SubmitTime: submitTime,
					ViaNode:    targetNode,
				}

				mu.Lock()
				result.Submissions = append(result.Submissions, sub)
				if res.Err != nil {
					result.Errors = append(result.Errors, res.Err)
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	result.EndTime = time.Now()

	return result
}

// sequencePattern generates out-of-order sequences: [base+2, base, base+1, base+3, ...].
func sequencePattern(base uint64, count int) []uint64 {
	if count <= 3 {
		seqs := []uint64{base + 2, base, base + 1}
		return seqs[:count]
	}

	seqs := []uint64{base + 2, base, base + 1}
	for i := 3; i < count; i++ {
		seqs = append(seqs, base+uint64(i))
	}

	return seqs
}

// Warmup sends a small number of transactions to warm up the cluster.
func Warmup(ctx context.Context, cluster *e2e.Cluster, metas map[string]e2e.AccountMeta) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	names := cluster.AccountNames()
	for i := 0; i < 5 && i < len(names); i++ {
		name := names[i]
		meta := metas[name]
		viaNode := rng.Intn(cluster.NodeCount())
		cluster.SendBankTxWithSequence(
			ctx, name, cluster.ValidatorAddress(), "1uinit",
			meta.AccountNumber, meta.Sequence, defaultGasLimit, viaNode,
		)
	}
}
