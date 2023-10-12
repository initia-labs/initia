# this script is for generating protobuf files for the new google.golang.org/protobuf API

set -eo pipefail

protoc_install_gopulsar() {
  go install github.com/cosmos/cosmos-proto/cmd/protoc-gen-go-pulsar@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
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

# exclude ibc modules
cd proto
proto_dirs=$(find ./initia -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    buf generate --template buf.gen.pulsar.yaml $file
  done
done
cd ..
