package benchmark

import (
	"context"
	"sync"
	"testing"
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
				if cfg.ValidatorCount > 0 && cfg.ValidatorCount < cfg.NodeCount {
					viaNode = edgeNodeIndex(i, cfg.NodeCount, cfg.ValidatorCount)
				}
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
		if cfg.ValidatorCount > 0 && cfg.ValidatorCount < cfg.NodeCount {
			viaNode = edgeNodeIndex(accountIdx, cfg.NodeCount, cfg.ValidatorCount)
		}

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
				if cfg.ValidatorCount > 0 && cfg.ValidatorCount < cfg.NodeCount {
					viaNode = edgeNodeIndex(i, cfg.NodeCount, cfg.ValidatorCount)
				}
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

// MoveExecSequentialLoad returns a LoadFn that submits Move execute transactions
// sequentially per account. Each account is pinned to a single node and waits for
// each submission before sending the next, ensuring sequences arrive in-order.
func MoveExecSequentialLoad(moduleAddr, moduleName, functionName string, typeArgs, args []string, gasLimit uint64) func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
	return func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
		if cfg.NodeCount <= 0 {
			panic("MoveExecSequentialLoad: cfg.NodeCount must be > 0")
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
			if cfg.ValidatorCount > 0 && cfg.ValidatorCount < cfg.NodeCount {
				viaNode = edgeNodeIndex(accountIdx, cfg.NodeCount, cfg.ValidatorCount)
			}

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
					if cfg.ValidatorCount > 0 && cfg.ValidatorCount < cfg.NodeCount {
						viaNode = edgeNodeIndex(i, cfg.NodeCount, cfg.ValidatorCount)
					}
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

// QueuedFloodLoad submits txs with nonces [base+1..base+N] (skipping base+0),
// flooding the queued pool. After all future-nonce txs are submitted, it sends the
// gap-filling base+0 tx for each account to trigger a promotion cascade.
// This tests the queued pool's ability to hold and promote a large batch.
func QueuedFloodLoad(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
	if cfg.NodeCount <= 0 {
		panic("QueuedFloodLoad: cfg.NodeCount must be > 0")
	}
	if cfg.TxPerAccount < 2 {
		panic("QueuedFloodLoad: TxPerAccount must be >= 2")
	}

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		result LoadResult
	)

	result.StartTime = time.Now()

	// Phase 1: submit future-nonce txs [base+1..base+N-1] for all accounts concurrently.
	for _, name := range cluster.AccountNames() {
		meta := metas[name]

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 1; i < cfg.TxPerAccount; i++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				seq := meta.Sequence + uint64(i) //nolint:gosec // i is bounded by TxPerAccount
				viaNode := i % cfg.NodeCount
				if cfg.ValidatorCount > 0 && cfg.ValidatorCount < cfg.NodeCount {
					viaNode = edgeNodeIndex(i, cfg.NodeCount, cfg.ValidatorCount)
				}
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

	// Brief pause to let queued txs propagate across the cluster.
	time.Sleep(2 * time.Second)

	// Phase 2: send the gap-filling base+0 tx for each account.
	for _, name := range cluster.AccountNames() {
		meta := metas[name]
		viaNode := 0
		if cfg.ValidatorCount > 0 && cfg.ValidatorCount < cfg.NodeCount {
			viaNode = edgeNodeIndex(0, cfg.NodeCount, cfg.ValidatorCount)
		}
		submitTime := time.Now()

		res := cluster.SendBankTxWithSequence(
			ctx, name, cluster.ValidatorAddress(), "1uinit",
			meta.AccountNumber, meta.Sequence, cfg.GetGasLimit(), viaNode,
		)

		sub := TxSubmission{
			TxHash:     res.TxHash,
			Account:    name,
			Sequence:   meta.Sequence,
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

	result.EndTime = time.Now()

	return result
}

// QueuedGapLoad submits txs with nonces [base+1..base+N] (skipping base+0),
// flooding the queued pool but never filling the gap. The queued txs should
// eventually be evicted by the gap TTL (default 60s).
func QueuedGapLoad(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
	if cfg.NodeCount <= 0 {
		panic("QueuedGapLoad: cfg.NodeCount must be > 0")
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
			for i := 1; i <= cfg.TxPerAccount; i++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				seq := meta.Sequence + uint64(i) //nolint:gosec // i is bounded by TxPerAccount
				viaNode := i % cfg.NodeCount
				if cfg.ValidatorCount > 0 && cfg.ValidatorCount < cfg.NodeCount {
					viaNode = edgeNodeIndex(i, cfg.NodeCount, cfg.ValidatorCount)
				}
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

// PreSignBankTxs generates and signs all bank send transactions offline in parallel.
// This is done before the benchmark starts so the signing overhead doesn't affect measurements.
func PreSignBankTxs(ctx context.Context, t *testing.T, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) []e2e.SignedTx {
	t.Helper()
	total := cfg.TotalTx()
	txs := make([]e2e.SignedTx, 0, total)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, name := range cluster.AccountNames() {
		meta := metas[name]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < cfg.TxPerAccount; i++ {
				seq := meta.Sequence + uint64(i) //nolint:gosec // i is bounded by TxPerAccount
				signed, err := cluster.GenerateSignedBankTx(
					ctx, name, cluster.ValidatorAddress(), "1uinit",
					meta.AccountNumber, seq, cfg.GetGasLimit(),
				)
				if err != nil {
					t.Logf("[pre-sign] failed from=%s seq=%d err=%v", name, seq, err)
					continue
				}
				mu.Lock()
				txs = append(txs, signed)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	t.Logf("Pre-signed %d/%d bank txs", len(txs), total)
	return txs
}

// PreSignMoveExecTxs generates and signs all Move execute transactions offline in parallel.
func PreSignMoveExecTxs(
	ctx context.Context, t *testing.T, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta,
	moduleAddr, moduleName, functionName string, typeArgs, args []string, gasLimit uint64,
) []e2e.SignedTx {
	t.Helper()
	total := cfg.TotalTx()
	txs := make([]e2e.SignedTx, 0, total)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, name := range cluster.AccountNames() {
		meta := metas[name]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < cfg.TxPerAccount; i++ {
				seq := meta.Sequence + uint64(i) //nolint:gosec // i is bounded by TxPerAccount
				signed, err := cluster.GenerateSignedMoveExecTx(
					ctx, name, moduleAddr, moduleName, functionName,
					typeArgs, args,
					meta.AccountNumber, seq, gasLimit,
				)
				if err != nil {
					t.Logf("[pre-sign] failed from=%s seq=%d err=%v", name, seq, err)
					continue
				}
				mu.Lock()
				txs = append(txs, signed)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	t.Logf("Pre-signed %d/%d move exec txs", len(txs), total)
	return txs
}

// PreSignedBurstLoad broadcasts pre-signed transactions via HTTP as fast as possible.
// Transactions are distributed round-robin across edge nodes.
func PreSignedBurstLoad(signedTxs []e2e.SignedTx) func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
	return func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, _ map[string]e2e.AccountMeta) LoadResult {
		if cfg.NodeCount <= 0 {
			panic("PreSignedBurstLoad: cfg.NodeCount must be > 0")
		}

		byAccount := make(map[string][]e2e.SignedTx)
		for _, tx := range signedTxs {
			byAccount[tx.Account] = append(byAccount[tx.Account], tx)
		}

		for acct := range byAccount {
			txs := byAccount[acct]
			for i := 1; i < len(txs); i++ {
				for j := i; j > 0 && txs[j].Sequence < txs[j-1].Sequence; j-- {
					txs[j], txs[j-1] = txs[j-1], txs[j]
				}
			}
		}

		var (
			mu     sync.Mutex
			wg     sync.WaitGroup
			result LoadResult
		)

		result.StartTime = time.Now()

		accountIdx := 0
		for acct, txs := range byAccount {
			_ = acct
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i, stx := range txs {
					select {
					case <-ctx.Done():
						return
					default:
					}

					viaNode := i % cfg.NodeCount
					if cfg.ValidatorCount > 0 && cfg.ValidatorCount < cfg.NodeCount {
						viaNode = edgeNodeIndex(i, cfg.NodeCount, cfg.ValidatorCount)
					}
					submitTime := time.Now()

					res, err := cluster.BroadcastTxSync(ctx, viaNode, stx.TxBase64)

					sub := TxSubmission{
						TxHash:     stx.TxHash,
						Account:    stx.Account,
						Sequence:   stx.Sequence,
						SubmitTime: submitTime,
						ViaNode:    viaNode,
					}
					if err == nil {
						sub.Code = res.Code
						if res.TxHash != "" {
							sub.TxHash = res.TxHash
						}
					}

					mu.Lock()
					result.Submissions = append(result.Submissions, sub)
					if err != nil {
						result.Errors = append(result.Errors, err)
					} else if res.Err != nil {
						result.Errors = append(result.Errors, res.Err)
					}
					mu.Unlock()
				}
			}()
			accountIdx++
		}

		wg.Wait()
		result.EndTime = time.Now()

		return result
	}
}

// PreSignedSequentialLoad broadcasts pre-signed transactions via HTTP, one at a time
// per account, with each account pinned to a single edge node.
func PreSignedSequentialLoad(signedTxs []e2e.SignedTx) func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, metas map[string]e2e.AccountMeta) LoadResult {
	return func(ctx context.Context, cluster *e2e.Cluster, cfg BenchConfig, _ map[string]e2e.AccountMeta) LoadResult {
		if cfg.NodeCount <= 0 {
			panic("PreSignedSequentialLoad: cfg.NodeCount must be > 0")
		}

		byAccount := make(map[string][]e2e.SignedTx)
		for _, tx := range signedTxs {
			byAccount[tx.Account] = append(byAccount[tx.Account], tx)
		}

		for acct := range byAccount {
			txs := byAccount[acct]
			for i := 1; i < len(txs); i++ {
				for j := i; j > 0 && txs[j].Sequence < txs[j-1].Sequence; j-- {
					txs[j], txs[j-1] = txs[j-1], txs[j]
				}
			}
		}

		var (
			mu     sync.Mutex
			wg     sync.WaitGroup
			result LoadResult
		)

		result.StartTime = time.Now()

		accountIdx := 0
		for acct, txs := range byAccount {
			_ = acct
			viaNode := accountIdx % cfg.NodeCount
			if cfg.ValidatorCount > 0 && cfg.ValidatorCount < cfg.NodeCount {
				viaNode = edgeNodeIndex(accountIdx, cfg.NodeCount, cfg.ValidatorCount)
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				for i, stx := range txs {
					select {
					case <-ctx.Done():
						return
					default:
					}

					submitTime := time.Now()
					res, err := cluster.BroadcastTxSync(ctx, viaNode, stx.TxBase64)

					sub := TxSubmission{
						TxHash:     stx.TxHash,
						Account:    stx.Account,
						Sequence:   stx.Sequence,
						SubmitTime: submitTime,
						ViaNode:    viaNode,
					}
					if err == nil {
						sub.Code = res.Code
						if res.TxHash != "" {
							sub.TxHash = res.TxHash
						}
					}

					mu.Lock()
					result.Submissions = append(result.Submissions, sub)
					if err != nil {
						result.Errors = append(result.Errors, err)
					} else if res.Err != nil {
						result.Errors = append(result.Errors, res.Err)
					}
					mu.Unlock()

					// Throttle to avoid overwhelming CheckTx on the receiving node.
					// 5ms per tx × N accounts ≈ N*200 tx/s total, well above chain TPS.
					if i < len(txs)-1 {
						time.Sleep(5 * time.Millisecond)
					}
				}
			}()
			accountIdx++
		}

		wg.Wait()
		result.EndTime = time.Now()

		return result
	}
}

// edgeNodeIndex returns a non-validator node index for round-robin distribution.
// With NodeCount=8 and ValidatorCount=5, edge nodes are indices 5,6,7.
func edgeNodeIndex(i, nodeCount, validatorCount int) int {
	edgeCount := nodeCount - validatorCount
	if edgeCount <= 0 {
		return i % nodeCount // fallback: no edge nodes
	}
	return validatorCount + (i % edgeCount)
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
