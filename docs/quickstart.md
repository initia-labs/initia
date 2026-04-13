# Quickstart Guide

`initiad quickstart` configures your Initia node after running `initiad init`. It downloads the genesis file, address book, and sets up sync method, pruning, indexing, and other settings automatically.

## Prerequisites

```bash
initiad init <moniker>
```

## Interactive Mode

The easiest way to set up your node:

```bash
initiad quickstart --interactive
```

This walks you through each setting with guided prompts:

1. **Network** - mainnet or testnet
2. **Sync method** - statesync (fast) or snapshot (requires download URL)
3. **App state pruning** - default, nothing, everything, or custom
4. **Block pruning** - minimum blocks to retain (default: 1,000,000)
5. **TX indexing** - disable, default (IBC-related keys), or custom
6. **MemIAVL** - enable/disable (not available with snapshot sync)
7. **REST API** - enable and set listen address
8. **RPC address** - listen address for RPC server

## Flag Mode

For scripting and automation, pass all options as flags:

### Mainnet with State Sync

```bash
initiad quickstart \
  --network=mainnet \
  --sync-method=statesync \
  --tx-indexing=default \
  --pruning=default \
  --min-retain-blocks=1000000 \
  --memiavl-enable \
  --api-address=tcp://0.0.0.0:1317 \
  --rpc-address=tcp://0.0.0.0:26657
```

### Testnet with State Sync (Minimal)

```bash
initiad quickstart \
  --network=testnet \
  --sync-method=statesync \
  --tx-indexing=null \
  --pruning=everything
```

### Mainnet with Snapshot

Find the latest snapshot URL at <https://polkachu.com/tendermint_snapshots/initia>

```bash
initiad quickstart \
  --network=mainnet \
  --sync-method=snapshot \
  --snapshot-url=https://snapshots.polkachu.com/tendermint_snapshots/initia/initia_12345678.tar.lz4 \
  --tx-indexing=default \
  --pruning=default
```

> Note: Snapshot sync requires `curl`, `lz4`, and `tar` to be installed. MemIAVL cannot be used with snapshot sync.

### Custom Pruning

```bash
initiad quickstart \
  --network=mainnet \
  --sync-method=statesync \
  --tx-indexing=default \
  --pruning=custom \
  --pruning-keep-recent=362880 \
  --pruning-interval=100
```

### Custom TX Indexing Keys

```bash
initiad quickstart \
  --network=mainnet \
  --sync-method=statesync \
  --tx-indexing=custom \
  --tx-indexing-keys="tx.height,tx.hash,send_packet.packet_sequence" \
  --pruning=default
```

## Aliases

`quickstart` can also be invoked as `qstart` or `qs`:

```bash
initiad qs --interactive
initiad qstart --network=mainnet --sync-method=statesync --tx-indexing=default --pruning=default
```

## Flags Reference

| Flag | Description | Values | Default |
|------|-------------|--------|---------|
| `--interactive` | Run in interactive mode | - | false |
| `--network` | Target network | `mainnet`, `testnet` | required |
| `--sync-method` | How to sync the node | `statesync`, `snapshot` | required |
| `--snapshot-url` | Snapshot download URL | URL | required for snapshot |
| `--tx-indexing` | TX indexing mode | `null`, `default`, `custom` | required |
| `--tx-indexing-keys` | Custom indexing keys | comma-separated | required for custom |
| `--pruning` | App state pruning | `default`, `nothing`, `everything`, `custom` | required |
| `--pruning-keep-recent` | States to keep (custom) | integer | - |
| `--pruning-interval` | Pruning interval (custom) | integer | - |
| `--min-retain-blocks` | Minimum blocks to retain | integer | 1000000 |
| `--memiavl-enable` | Enable MemIAVL | - | false |
| `--api-address` | REST API address (enables API) | `tcp://ip:port` | - |
| `--rpc-address` | RPC listen address | `tcp://ip:port` | `tcp://127.0.0.1:26657` |

## What It Does

1. **Downloads genesis.json** from Polkachu (with RPC fallback)
2. **Downloads addrbook.json** from Polkachu (with RPC `/net_info` fallback)
3. **Configures config.toml** - RPC address, TX indexing, retain height
4. **Configures app.toml** - pruning, block retention, API, MemIAVL, index events
5. **Sets up sync method:**
   - **State sync**: fetches trust height/hash from RPC, configures state sync peer
   - **Snapshot**: downloads and extracts `.tar.lz4` snapshot

## TX Indexing Presets

- **disable** (`null`): no transaction indexing
- **default**: IBC-related keys optimized for relayer operations:
  - `tx.height`, `tx.hash`
  - `send_packet.packet_sequence`, `recv_packet.packet_sequence`
  - `write_acknowledgement.packet_sequence`, `acknowledge_packet.packet_sequence`
  - `timeout_packet.packet_sequence`, `finalize_token_deposit.l1_sequence`
- **custom**: specify your own comma-separated keys

## After Quickstart

Start your node:

```bash
initiad start
```
