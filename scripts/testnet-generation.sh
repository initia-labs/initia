#!/usr/bin/env bash

mkdir -p testnet

initiad init base --home testnet/base --chain-id testnet
initiad init fullnode --home testnet/fullnode1 --chain-id testnet
initiad init fullnode --home testnet/fullnode2 --chain-id testnet

initiad keys add val --keyring-backend test --home testnet/base
initiad keys add acc1 --keyring-backend test --home testnet/base
initiad keys add acc2 --keyring-backend test --home testnet/base
initiad keys add acc3 --keyring-backend test --home testnet/base

initiad genesis add-genesis-account val 1000000000000000uinit --home testnet/base --keyring-backend test
initiad genesis add-genesis-account acc1 1000000000000000uinit --home testnet/base --keyring-backend test
initiad genesis add-genesis-account acc2 1000000000000000uinit --home testnet/base --keyring-backend test
initiad genesis add-genesis-account acc3 1000000000000000uinit --home testnet/base --keyring-backend test

initiad genesis gentx val 500000000000uinit --home testnet/base --keyring-backend test --chain-id testnet
initiad genesis collect-gentxs --home testnet/base

cp -r testnet/base/config/genesis.json testnet/fullnode1/config/genesis.json
cp -r testnet/base/config/genesis.json testnet/fullnode2/config/genesis.json

set_toml() {
  local file=$1
  local section=$2
  local key=$3
  local value=$4

  sed -i.bak -E "/^\\[${section}\\]/,/^\\[/{ s|^${key}[[:space:]]*=.*|${key} = ${value}|; }" "$file"
  rm -f "${file}.bak"
}

# Update overlapping ports
set_toml testnet/fullnode1/config/config.toml rpc laddr "\"tcp://127.0.0.1:26647\""
set_toml testnet/fullnode2/config/config.toml rpc laddr "\"tcp://127.0.0.1:26637\""

set_toml testnet/fullnode1/config/config.toml p2p laddr "\"tcp://127.0.0.1:26646\""
set_toml testnet/fullnode2/config/config.toml p2p laddr "\"tcp://127.0.0.1:26636\""

PEER_ID_VAL=$(initiad comet show-node-id --home testnet/base)
PEER_ID_FULLNODE1=$(initiad comet show-node-id --home testnet/fullnode1)
PEER_ID_FULLNODE2=$(initiad comet show-node-id --home testnet/fullnode2)

# Update persistent peers
set_toml testnet/base/config/config.toml p2p persistent_peers "\"${PEER_ID_FULLNODE1}@127.0.0.1:26646,${PEER_ID_FULLNODE2}@127.0.0.1:26636\""
set_toml testnet/fullnode1/config/config.toml p2p persistent_peers "\"${PEER_ID_VAL}@127.0.0.1:26656\""
set_toml testnet/fullnode2/config/config.toml p2p persistent_peers "\"${PEER_ID_VAL}@127.0.0.1:26656\""

# allow duplicate IPs for local testing
set_toml testnet/base/config/config.toml p2p allow_duplicate_ip true
set_toml testnet/fullnode1/config/config.toml p2p allow_duplicate_ip true
set_toml testnet/fullnode2/config/config.toml p2p allow_duplicate_ip true
set_toml testnet/base/config/config.toml p2p addr_book_strict false
set_toml testnet/fullnode1/config/config.toml p2p addr_book_strict false
set_toml testnet/fullnode2/config/config.toml p2p addr_book_strict false

# enable rest server
set_toml testnet/base/config/app.toml api enable true
set_toml testnet/fullnode1/config/app.toml api enable true
set_toml testnet/fullnode2/config/app.toml api enable true

# update rest endpoint ports
set_toml testnet/base/config/app.toml api address "\"tcp://localhost:1317\""
set_toml testnet/fullnode1/config/app.toml api address "\"tcp://localhost:1318\""
set_toml testnet/fullnode2/config/app.toml api address "\"tcp://localhost:1319\""
set_toml testnet/base/config/app.toml api swagger true
set_toml testnet/fullnode1/config/app.toml api swagger true
set_toml testnet/fullnode2/config/app.toml api swagger true

# update grpc endpoint ports
set_toml testnet/base/config/app.toml grpc address "\"localhost:9090\""
set_toml testnet/fullnode1/config/app.toml grpc address "\"localhost:9091\""
set_toml testnet/fullnode2/config/app.toml grpc address "\"localhost:9092\""

initiad start --home testnet/base > testnet/base/node.log 2>&1 &
PID_BASE=$!
initiad start --home testnet/fullnode1 > testnet/fullnode1/node.log 2>&1 &
PID_FULLNODE1=$!
initiad start --home testnet/fullnode2 > testnet/fullnode2/node.log 2>&1 &
PID_FULLNODE2=$!

PIDS=("$PID_BASE" "$PID_FULLNODE1" "$PID_FULLNODE2")

cleanup() {
  if [ -n "${CLEANUP_DONE:-}" ]; then
    return 0
  fi
  CLEANUP_DONE=1
  trap '' INT TERM
  echo "Stopping testnet nodes..."
  kill -TERM "${PIDS[@]}" 2>/dev/null
  for _ in 1 2 3 4 5; do
    local any_alive=false
    for pid in "${PIDS[@]}"; do
      if kill -0 "$pid" 2>/dev/null; then
        any_alive=true
        break
      fi
    done
    if [ "$any_alive" = false ]; then
      break
    fi
    sleep 1
  done
  kill -KILL "${PIDS[@]}" 2>/dev/null
  wait "${PIDS[@]}" 2>/dev/null
  rm -rf testnet
}

trap cleanup INT TERM EXIT

show_logs() {
  local node=$1
  local log_file=
  case "$node" in
    base) log_file="testnet/base/node.log" ;;
    fullnode1) log_file="testnet/fullnode1/node.log" ;;
    fullnode2) log_file="testnet/fullnode2/node.log" ;;
    *) echo "Unknown node: $node"; return 1 ;;
  esac

  echo "Tailing $log_file (press 'q' or Esc to go back)"
  tail -f "$log_file" &
  local tail_pid=$!
  while IFS= read -rsn1 key; do
    if [[ $key == $'q' || $key == $'\e' ]]; then
      break
    fi
  done
  kill "$tail_pid" 2>/dev/null
  wait "$tail_pid" 2>/dev/null
}

select_account() {
  local choice
  echo "Select account:" >&2
  echo "1) acc1" >&2
  echo "2) acc2" >&2
  echo "3) acc3" >&2
  echo "4) Back" >&2
  read -r -p "> " choice >&2
  case "$choice" in
    1) echo "acc1" ;;
    2) echo "acc2" ;;
    3) echo "acc3" ;;
    4) echo "" ;;
    *) echo "__invalid__" ;;
  esac
}

send_from_account() {
  local account receiver amount
  account=$(select_account)
  if [ "$account" = "__invalid__" ]; then
    echo "Invalid choice."
    return 1
  fi
  if [ -z "$account" ]; then
    return 0
  fi

  read -r -p "Receiver address: " receiver
  read -r -p "Amount (e.g. 1000000uinit): " amount
  if [ -z "$receiver" ] || [ -z "$amount" ]; then
    echo "Receiver and amount are required."
    return 1
  fi

  initiad tx bank send "$account" "$receiver" "$amount" \
    --chain-id testnet \
    --gas-prices 0.015uinit \
    --gas auto \
    --gas-adjustment 1.4 \
    --node http://127.0.0.1:26647 \
    --keyring-backend test \
    -y \
    --home testnet/base
}

show_account_info() {
  local account
  account=$(select_account)
  if [ "$account" = "__invalid__" ]; then
    echo "Invalid choice."
    return 1
  fi
  if [ -z "$account" ]; then
    return 0
  fi

  initiad keys show "$account" --keyring-backend test --home testnet/base
}

echo "Testnet nodes started (pids: ${PIDS[*]})."
while true; do
  echo ""
  echo "Select an option:"
  echo "1) View logs"
  echo "2) Send bank tx"
  echo "3) Show account info"
  echo "4) Exit (stop nodes)"
  read -r -p "> " choice

  case "$choice" in
    1)
      echo "Select node to view logs:"
      echo "1) base"
      echo "2) fullnode1"
      echo "3) fullnode2"
      echo "4) Back"
      read -r -p "> " node_choice
      case "$node_choice" in
        1) show_logs base ;;
        2) show_logs fullnode1 ;;
        3) show_logs fullnode2 ;;
        4) ;;
        *) echo "Invalid choice." ;;
      esac
      ;;
    2) send_from_account ;;
    3) show_account_info ;;
    4) cleanup; exit 0 ;;
    *) echo "Invalid choice." ;;
  esac
done
