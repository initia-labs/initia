package benchmark

import (
	"context"
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
	Code       int64     `json:"code,omitempty"`
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
	if cfg.NodeCount <= 0 {
		panic("BurstLoad: cfg.NodeCount must be > 0")
	}

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		result LoadResult
	)

	result.StartTime = time.Now()

	for _, name := range cluster.AccountNames() {
		meta := metas[name]

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < cfg.TxPerAccount; i++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				seq := meta.Sequence + uint64(i) //nolint:gosec // i is bounded by TxPerAccount
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
					Code:       res.Code,
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

// SequentialLoad submits transactions sequentially per account, waiting for each
// submission before sending the next. Though each account runs concurrently and
// is pinned to a single node (accountIndex % NodeCount).
// This ensures sequences always arrive in-order, which CListMempool requires.
func SequentialLoad(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
	if cfg.NodeCount <= 0 {
		panic("SequentialLoad: cfg.NodeCount must be > 0")
	}

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		result LoadResult
	)

	result.StartTime = time.Now()

	for accountIdx, name := range cluster.AccountNames() {
		meta := metas[name]
		viaNode := accountIdx % cfg.NodeCount

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < cfg.TxPerAccount; i++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				seq := meta.Sequence + uint64(i) //nolint:gosec // i is bounded by TxPerAccount
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
					Code:       res.Code,
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

// OutOfOrderLoad submits the first 3 txs per account with out-of-order nonces
// (seq+2, seq+0, seq+1), then the rest sequentially. TxPerAccount must be >= 3.
func OutOfOrderLoad(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
	if cfg.NodeCount <= 0 {
		panic("OutOfOrderLoad: cfg.NodeCount must be > 0")
	}
	if cfg.TxPerAccount < 3 {
		panic("OutOfOrderLoad: TxPerAccount must be >= 3 for out-of-order pattern")
	}

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		result LoadResult
	)

	result.StartTime = time.Now()

	for _, name := range cluster.AccountNames() {
		meta := metas[name]

		wg.Add(1)
		go func() {
			defer wg.Done()
			seqs := sequencePattern(meta.Sequence, cfg.TxPerAccount)
			for i, seq := range seqs {
				select {
				case <-ctx.Done():
					return
				default:
				}

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
					Code:       res.Code,
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
	if targetNode < 0 || targetNode >= cfg.NodeCount {
		panic("SingleNodeLoad: targetNode out of range")
	}

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		result LoadResult
	)

	result.StartTime = time.Now()

	for _, name := range cluster.AccountNames() {
		meta := metas[name]

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < cfg.TxPerAccount; i++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				seq := meta.Sequence + uint64(i) //nolint:gosec // i is bounded by TxPerAccount
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
					Code:       res.Code,
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

// MoveExecBurstLoad returns a LoadFn that submits Move execute transactions concurrently.
// The Move exec details (module address, name, function, args, gas) are captured via closure.
func MoveExecBurstLoad(moduleAddr, moduleName, functionName string, typeArgs, args []string, gasLimit uint64) func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
	return func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
		if cfg.NodeCount <= 0 {
			panic("MoveExecBurstLoad: cfg.NodeCount must be > 0")
		}

		var (
			mu     sync.Mutex
			wg     sync.WaitGroup
			result LoadResult
		)

		result.StartTime = time.Now()

		for _, name := range cluster.AccountNames() {
			meta := metas[name]

			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < cfg.TxPerAccount; i++ {
					select {
					case <-ctx.Done():
						return
					default:
					}

					seq := meta.Sequence + uint64(i) //nolint:gosec // i is bounded by TxPerAccount
					viaNode := i % cfg.NodeCount
					submitTime := time.Now()

					res := cluster.SendMoveExecuteJSONWithGas(
						ctx, name, moduleAddr, moduleName, functionName,
						typeArgs, args,
						meta.AccountNumber, seq, gasLimit, viaNode,
					)

					sub := TxSubmission{
						TxHash:     res.TxHash,
						Account:    name,
						Sequence:   seq,
						SubmitTime: submitTime,
						ViaNode:    viaNode,
						Code:       res.Code,
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
}

// sequencePattern generates out-of-order sequences: [base+2, base, base+1, base+3, ...].
// count must be >= 3 for the out-of-order pattern to work correctly.
func sequencePattern(base uint64, count int) []uint64 {
	seqs := []uint64{base + 2, base, base + 1}
	if count <= 3 {
		return seqs[:count]
	}

	for i := 3; i < count; i++ {
		seqs = append(seqs, base+uint64(i)) //nolint:gosec // i is bounded by count
	}

	return seqs
}

// Warmup sends a small number of transactions to warm up the cluster.
// Callers MUST refresh account metadata after Warmup returns, because
// each successful submission advances the on-chain sequence.
func Warmup(ctx context.Context, cluster *e2e.Cluster, metas map[string]e2e.AccountMeta) {
	names := cluster.AccountNames()
	for i := 0; i < 5 && i < len(names); i++ {
		name := names[i]
		meta := metas[name]
		viaNode := i % cluster.NodeCount()
		cluster.SendBankTxWithSequence(
			ctx, name, cluster.ValidatorAddress(), "1uinit",
			meta.AccountNumber, meta.Sequence, defaultGasLimit, viaNode,
		)
	}
}
