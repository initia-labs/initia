# this script is for generating protobuf files for the new google.golang.org/protobuf API

set -eo pipefail

protoc_install_gopulsar() {
  go install github.com/cosmos/cosmos-proto/cmd/protoc-gen-go-pulsar@latest
  go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1
}

protoc_install_gopulsar

mkdir -p ./api

echo "Cleaning API directory"
(
  cd api
  find ./ -type f \( -iname \*.pulsar.go -o -iname \*.pb.go -o -iname \*.cosmos_orm.go -o -iname \*.pb.gw.go \) -delete
  find . -empty -type d -delete
  cd ..
)

echo "Generating API module"

# clone dependency proto files
IBC_URL=github.com/cosmos/ibc-go
IBC_V=v8
ICS23_URL=github.com/cosmos/ics23

IBC_VERSION=$(cat ./go.mod | grep "$IBC_URL/$IBC_V v" | sed -nE 's/.* (v[0-9][^[:space:]]*).*/\1/p')
ICS23_VERSION=$(cat ./go.mod | grep "$ICS23_URL/go v" | sed -nE 's/.* (v[0-9][^[:space:]]*).*/\1/p')

mkdir -p ./third_party
cd third_party
git clone -b $IBC_VERSION https://$IBC_URL
git clone -b go/$ICS23_VERSION https://$ICS23_URL
cd ..

# exclude ibc modules
cd proto
proto_dirs=$(find \
./initia \
./ibc/applications \
../third_party/ibc-go/proto/ibc \
../third_party/ics23/proto/cosmos/ics23 \
-path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    buf generate --template buf.gen.pulsar.yaml $file
  done
done
cd ..

# clean third party files
rm -rf ./third_party
