#!/usr/bin/env sh

set -eu

ROOT_DIR=$(pwd)
TMP_SWAGGER_DIR="${ROOT_DIR}/tmp-swagger-gen"
TMP_SWAGGER_TEMPLATE="${TMP_SWAGGER_DIR}/buf.gen.swagger.yaml"
THIRD_PARTY_DIR="${ROOT_DIR}/third_party"
SUCCESS=0

PATH="$(go env GOPATH)/bin:${PATH}"
export PATH

clone_if_missing_or_empty() {
  target_dir="$1"
  branch="$2"
  remote_url="$3"
  clone_dir="$4"

  if [ -d "${target_dir}" ] && [ -n "$(find "${target_dir}" -mindepth 1 -print -quit 2>/dev/null)" ]; then
    echo "Skipping clone for ${target_dir}; directory already exists and is not empty."
    return
  fi

  rm -rf "${target_dir}"
  git clone -b "${branch}" "https://${remote_url}" "${clone_dir}"
}

generate_swagger() {
  module_root="$1"
  shift

  (
    cd "${module_root}"
    proto_dirs=$(find "$@" -name '*.proto' -print | xargs -n1 dirname | sort | uniq)

    for dir in ${proto_dirs}; do
      query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \) | head -n 1)
      if [ -n "${query_file}" ]; then
        buf generate --template "${TMP_SWAGGER_TEMPLATE}" "${query_file}"
      fi
    done
  )
}

cleanup() {
  rm -rf "${TMP_SWAGGER_DIR}"

  if [ "${SUCCESS}" -eq 1 ]; then
    rm -rf "${THIRD_PARTY_DIR}"
  fi
}

trap cleanup EXIT

COSMOS_URL=github.com/cosmos/cosmos-sdk
IBC_URL=github.com/cosmos/ibc-go
IBC_V=v8
IBC_APP_URL=github.com/cosmos/ibc-apps
IBC_RATE_LIMITING_PATH=modules/rate-limiting
IBC_RATE_LIMITING_URL=$IBC_APP_URL/$IBC_RATE_LIMITING_PATH
OPINIT_URL=github.com/initia-labs/OPinit
CONNECT_URL=github.com/skip-mev/connect
CONNECT_V=v2

COSMOS_SDK_VERSION=$(grep "$COSMOS_URL v" ./go.mod | sed -n -e "s/^.* //p")
IBC_VERSION=$(grep "$IBC_URL/$IBC_V v" ./go.mod | sed -n -e "s/^.* //p")
IBC_RATE_LIMITING_VERSION=$(grep "$IBC_RATE_LIMITING_URL/$IBC_V v" ./go.mod | sed -n -e "s/^.* //p")
OPINIT_VERSION=$(grep "$OPINIT_URL v" ./go.mod | sed -n -e "s/^.* //p")
CONNECT_VERSION=$(grep "$CONNECT_URL/$CONNECT_V v" ./go.mod | sed -n -e "s/^.* //p")

mkdir -p "${THIRD_PARTY_DIR}"
mkdir -p "${TMP_SWAGGER_DIR}"

cd "${THIRD_PARTY_DIR}"
clone_if_missing_or_empty "${THIRD_PARTY_DIR}/cosmos-sdk" "${COSMOS_SDK_VERSION}" "${COSMOS_URL}" "cosmos-sdk"
clone_if_missing_or_empty "${THIRD_PARTY_DIR}/ibc-go" "${IBC_VERSION}" "${IBC_URL}" "ibc-go"
clone_if_missing_or_empty "${THIRD_PARTY_DIR}/ibc-rate-limiting" "${IBC_RATE_LIMITING_PATH}/${IBC_RATE_LIMITING_VERSION}" "${IBC_APP_URL}" "ibc-rate-limiting"
clone_if_missing_or_empty "${THIRD_PARTY_DIR}/OPinit" "${OPINIT_VERSION}" "${OPINIT_URL}" "OPinit"
clone_if_missing_or_empty "${THIRD_PARTY_DIR}/connect" "${CONNECT_VERSION}" "${CONNECT_URL}" "connect"
cd "${ROOT_DIR}"

cat > "${TMP_SWAGGER_TEMPLATE}" <<EOF
version: v1
plugins:
  - name: swagger
    out: ${TMP_SWAGGER_DIR}
    opt: logtostderr=true,fqn_for_swagger_name=true,simple_operation_ids=true
EOF

generate_swagger "${ROOT_DIR}/proto" initia ibc
generate_swagger "${ROOT_DIR}/third_party/cosmos-sdk/proto" cosmos
generate_swagger "${ROOT_DIR}/third_party/ibc-go/proto" ibc
generate_swagger "${ROOT_DIR}/third_party/ibc-rate-limiting/modules/rate-limiting/proto" ratelimit
generate_swagger "${ROOT_DIR}/third_party/OPinit/proto" opinit
generate_swagger "${ROOT_DIR}/third_party/connect/proto" connect

swagger-combine ./client/docs/config-connect.json -o ./client/docs/swagger-ui/swagger-connect.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true
swagger-combine ./client/docs/config-cosmos.json -o ./client/docs/swagger-ui/swagger-cosmos.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true
swagger-combine ./client/docs/config-ibc.json -o ./client/docs/swagger-ui/swagger-ibc.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true
swagger-combine ./client/docs/config-initia.json -o ./client/docs/swagger-ui/swagger-initia.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true
swagger-combine ./client/docs/config-opinit.json -o ./client/docs/swagger-ui/swagger-opinit.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true

SUCCESS=1
