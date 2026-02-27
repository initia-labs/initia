# abcipp

`abcipp` contains Initia's ABCI++ integration pieces:

- priority mempool implementation
- CheckTx alignment logic
- Prepare/Process proposal handlers
- mempool query server

## Documents

- [spec.md](./docs/spec.md): behavioral spec and architecture overview.
- [invariant.md](./docs/invariant.md): core mempool invariants and guarantees.

## Key Files

- `mempool.go`: core struct definition and shared entry points. Mempool logic is split across `mempool_insert.go`, `mempool_remove.go`, `mempool_sender_state.go`, `mempool_cleanup.go`, `mempool_invariant.go`, `mempool_event.go`, `mempool_tier.go`, `mempool_query.go`. See [spec.md](./docs/spec.md) for a full breakdown.
- `checktx.go`: CheckTx/recheck alignment path.
- `proposals.go`: PrepareProposal/ProcessProposal logic.
- `query_server.go`: mempool query endpoints.

## Tests

See [invariant.md](./docs/invariant.md) for the full test contract. Key files:

- `mempool_test.go`: end-to-end mempool behavior scenarios.
- `mempool_remove_test.go`, `mempool_sender_state_test.go`: removal and sender-state invariants.
- `mempool_cleanup_test.go`, `mempool_event_test.go`, `mempool_query_test.go`, `mempool_tier_test.go`: per-subsystem coverage.
- `mempool_test_utils_test.go`: shared test helpers.
- `proposal_test.go`, `query_server_test.go`: proposal/query tests.
