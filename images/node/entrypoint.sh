#!/usr/bin/env sh

set -xe

INITIAD_HOME="${INITIAD_HOME:-"${HOME}/.initia"}"
INITIAD_MONIKER="$(hostname)"

CONFIG_DIR="${INITIAD_HOME:?}/config"
GENESIS_URL="${GENESIS_URL:?}"

if [ -f "${CONFIG_DIR}/init.ok" ]; then
    echo "skipping init: config already present"
    exec "$@"
fi;

mkdir -p "${INITIAD_HOME}"

# `initiad init` does NOT take cmd env vars into account for some reason
# so we have to manually insert flags with the corresponding values
initiad --home "${INITIAD_HOME}" init "${INITIAD_MONIKER}"

wget -O "${CONFIG_DIR}/genesis.json" "${GENESIS_URL}"

touch "${CONFIG_DIR}/init.ok" 

exec "$@"
