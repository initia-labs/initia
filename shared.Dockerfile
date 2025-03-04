FROM golang:1.23-bullseye AS go-builder

# Install minimum necessary dependencies, build Cosmos SDK, remove packages
RUN apt update
RUN apt install -y curl git build-essential
# debug: for live editing in the image
RUN apt install -y vim

WORKDIR /code
COPY . /code/

RUN VERSION=${VERSION} LEDGER_ENABLED=false make build

RUN cp /go/pkg/mod/github.com/initia\-labs/movevm@v*/api/libmovevm.`uname -m`.so /lib/libmovevm.so
RUN cp /go/pkg/mod/github.com/initia\-labs/movevm@v*/api/libcompiler.`uname -m`.so /lib/libcompiler.so

FROM ubuntu:20.04

WORKDIR /root

COPY --from=go-builder /code/build/initiad /usr/local/bin/initiad
COPY --from=go-builder /lib/libmovevm.so /lib/libmovevm.so
COPY --from=go-builder /lib/libcompiler.so /lib/libcompiler.so

# rest server
EXPOSE 1317
# grpc
EXPOSE 9090
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

CMD ["/usr/local/bin/initiad", "version"]
