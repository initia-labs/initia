FROM golang:1.21-bullseye AS go-builder

# Install minimum necessary dependencies, build Cosmos SDK, remove packages
RUN apt update
RUN apt install -y curl git build-essential
# debug: for live editting in the image
RUN apt install -y vim

WORKDIR /code
COPY . /code/

RUN LEDGER_ENABLED=false make build

RUN cp /go/pkg/mod/github.com/initia\-labs/initiavm@v*/api/libmovevm.`uname -m`.so /lib/libmovevm.so

FROM ubuntu:20.04

WORKDIR /root

COPY --from=go-builder /code/build/initiad /usr/local/bin/initiad
COPY --from=go-builder /lib/libmovevm.so /lib/libmovevm.so

# rest server
EXPOSE 1317
# grpc
EXPOSE 9090
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

CMD ["/usr/local/bin/initiad", "version"]
