#!/bin/bash

BINARY=$1
CONTINUE=${CONTINUE:-"false"}
HOME_DIR=mytestnet
ENV=${ENV:-""}

if [ "$CONTINUE" == "true" ]; then
    $BINARY start --home $HOME_DIR --log_level debug
    exit 0
fi

rm -rf mytestnet
pkill initiad

# check DENOM is set. If not, set to uinit
DENOM=${2:-uinit}

COMMISSION_RATE=0.01
COMMISSION_MAX_RATE=0.02

SED_BINARY=sed
# check if this is OS X
if [[ "$OSTYPE" == "darwin"* ]]; then
    # check if gsed is installed
    if ! command -v gsed &> /dev/null
    then
        echo "gsed could not be found. Please install it with 'brew install gnu-sed'"
        exit
    else
        SED_BINARY=gsed
    fi
fi

# check BINARY is set. If not, build picad and set BINARY
if [ -z "$BINARY" ]; then
    make build
    BINARY=build/picad
fi

CHAIN_ID="localinit"
KEYRING="test"
KEY="test0"
KEY1="test1"
KEY2="test2"

# Function updates the config based on a jq argument as a string
update_test_genesis () {
    # update_test_genesis '.consensus_params["block"]["max_gas"]="100000000"'
    cat $HOME_DIR/config/genesis.json | jq "$1" > $HOME_DIR/config/tmp_genesis.json && mv $HOME_DIR/config/tmp_genesis.json $HOME_DIR/config/genesis.json
}

$BINARY init --chain-id $CHAIN_ID moniker --home $HOME_DIR

$BINARY keys add $KEY --keyring-backend $KEYRING --home $HOME_DIR
$BINARY keys add $KEY1 --keyring-backend $KEYRING --home $HOME_DIR
$BINARY keys add $KEY2 --keyring-backend $KEYRING --home $HOME_DIR

# Allocate genesis accounts (cosmos formatted addresses)
$BINARY genesis add-genesis-account $KEY "1000000000${DENOM}" --keyring-backend $KEYRING --home $HOME_DIR
$BINARY genesis add-genesis-account $KEY1 "1000000000${DENOM}" --keyring-backend $KEYRING --home $HOME_DIR
$BINARY genesis add-genesis-account $KEY2 "1000000000${DENOM}" --keyring-backend $KEYRING --home $HOME_DIR

# update_test_genesis '.app_state["gov"]["params"]["voting_period"]="20s"'
update_test_genesis '.app_state["mint"]["params"]["mint_denom"]="'$DENOM'"'
update_test_genesis '.app_state["gov"]["params"]["min_deposit"]=[{"denom":"'$DENOM'","amount": "1000000"}]'
update_test_genesis '.app_state["crisis"]["constant_fee"]={"denom":"'$DENOM'","amount":"1000"}'
update_test_genesis '.app_state["staking"]["params"]["bond_denom"]="'$DENOM'"'

# enable rest server and swagger
$SED_BINARY -i '0,/enable = false/s//enable = true/' $HOME_DIR/config/app.toml
$SED_BINARY -i 's/swagger = false/swagger = true/' $HOME_DIR/config/app.toml
$SED_BINARY -i -e 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' $HOME_DIR/config/app.toml
$SED_BINARY -i 's/minimum-gas-prices = "0.uinit"/minimum-gas-prices = "0.uinit"/' $HOME_DIR/config/app.toml


# Sign genesis transaction
$BINARY genesis gentx $KEY "100000000${DENOM}" --commission-rate=$COMMISSION_RATE --commission-max-rate=$COMMISSION_MAX_RATE --keyring-backend $KEYRING --chain-id $CHAIN_ID --home $HOME_DIR

# Collect genesis tx
$BINARY genesis collect-gentxs --home $HOME_DIR

# Run this to ensure everything worked and that the genesis file is setup correctly
$BINARY validate-genesis --home $HOME_DIR
$BINARY start --home $HOME_DIR

