# E2E Multi-node Framework

This package provides reusable tools to run `initiad` as a real multi-node cluster (1-10 nodes) and execute high-concurrency transaction ordering scenarios.

## Structure

- `cluster/`: node lifecycle, CLI helpers, Move tx/query helpers
- `mempool/queue_clear_e2e_test.go`: bank queue-clear scenario + shared helpers
- `mempool/move_account_queue_clear_e2e_test.go`: stdlib account creation queue-clear scenario
- `mempool/move_table_generator_queue_clear_e2e_test.go`: TableGenerator resource + queue-clear scenario

## Run

Compile-only:

```bash
go test ./integration-tests/e2e/... -tags e2e -run '^$'
```

Run queue-clear bank scenario:

```bash
E2E_NODE_COUNT=4 E2E_ACCOUNT_COUNT=4 E2E_TX_PER_ACCOUNT=3 \
  go test ./integration-tests/e2e/mempool -tags e2e -run TestQueueClearOrdering -count=1
```

Optional env vars:

- `E2E_NODE_COUNT` (default `5`, max `10`)
- `E2E_ACCOUNT_COUNT` (default `5`)
- `E2E_TX_PER_ACCOUNT` (default `10`)
- `E2E_INITIAD_BIN` (optional prebuilt `initiad` path)
