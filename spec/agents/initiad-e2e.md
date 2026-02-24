# initiad E2E Runbook

Last updated: 2026-02-24

## Scope

Guidance for multi-node `initiad` e2e tests around CometBFT + proxy mempool + Move tx flows.

Current scenario package path: `integration-tests/e2e/mempool`.

## Non-obvious rules

1. Keyring/home consistency is critical.

   - In this e2e setup, local keys are created under `nodes[0].home`.
   - Commands that sign or inspect keys must consistently use:
     - `--home <nodes[0].home>`
     - `--keyring-backend test`

2. Manual sequence control requires offline signing.

   - To make `--sequence` and `--account-number` deterministic, send tx with `--offline`.

3. Gas handling should be estimate-then-execute.

   - Do not hardcode large gas values in scenarios.
   - Estimate first via `--generate-only`, parse estimated gas, then execute with fixed `--gas <estimated>`.

4. Move publish must match runtime named addresses.

   - `TableGenerator` (and similar modules) can fail if built with a mismatched named address.
   - In e2e, build module at runtime with `initiad move build` and pass e2e account VM address through `--named-addresses`.

5. Multi-node tx ingress should be distributed.

   - For mempool behavior tests, do not pin all tx to one RPC node.
   - Randomize target RPC node per tx (`viaNode` per send).

6. Queue-clear success criteria should be explicit.

   - Wait for mempool drain (`num_unconfirmed_txs == 0`).
   - Verify final account sequence converges to `initial + accepted_tx_count` for each sender.

## Scenario checklist

- [ ] Cluster starts and produces blocks (`WaitForReady`).
- [ ] Workload sends out-of-order sequence pattern (e.g. `base+2, base, base+1, ...`).
- [ ] Broadcast results captured per tx (`code`, `raw_log`, `txhash`).
- [ ] Mempool drained before final assertions.
- [ ] Final sequence invariants validated for all accounts.

## Useful commands

Compile-only e2e packages:

```bash
go test ./integration-tests/e2e/... -tags e2e -run '^$'
```

Run queue-clear bank scenario:

```bash
E2E_NODE_COUNT=4 E2E_ACCOUNT_COUNT=4 E2E_TX_PER_ACCOUNT=3 \
  go test ./integration-tests/e2e/mempool -tags e2e -run TestQueueClearOrdering -count=1
```

## Known failure signatures

- `key not found`
  - Usually `--home` or keyring mismatch; verify key lookup against `nodes[0].home`.

- `tx nonce X is stale ... expected >= Y`
  - Sequence/account-number snapshot is stale or previous tx already committed.
  - Re-query account meta before building next deterministic sequence batch.

- `tx already exists` / duplicate tx hash
  - Same signed tx bytes re-broadcast; often from duplicated sequence+payload under offline mode.
  - Ensure unique tx payload when intentionally testing same sender sequence edges.
