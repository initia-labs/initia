FROM golang:1.22-alpine3.19 AS go-builder
#ARG arch=x86_64

# See https://github.com/initia-labs/movevm/releases
ARG LIBMOVEVM_VERSION=v0.3.0

# this comes from standard alpine nightly file
#  https://github.com/rust-lang/docker-rust-nightly/blob/master/alpine3.12/Dockerfile
# with some changes to support our toolchain, etc
RUN set -eux; apk add --no-cache ca-certificates build-base;

RUN apk add git cmake
# NOTE: add these to run with LEDGER_ENABLED=true
# RUN apk add libusb-dev linux-headers

WORKDIR /code
COPY . /code/
# Install mimalloc
RUN git clone --depth 1 https://github.com/microsoft/mimalloc; cd mimalloc; mkdir build; cd build; cmake ..; make -j$(nproc); make install
ENV MIMALLOC_RESERVE_HUGE_OS_PAGES=4

# private ci access
ARG GITHUB_ACCESS_TOKEN
ENV GOPRIVATE=github.com/initia-labs/*
RUN git config --global url."https://${GITHUB_ACCESS_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"

# See https://github.com/initia-labs/movevm/releases
ADD https://github.com/gruntwork-io/fetch/releases/download/v0.4.6/fetch_linux_amd64 /usr/bin/fetch
RUN chmod +x /usr/bin/fetch
ENV GITHUB_OAUTH_TOKEN=$GITHUB_ACCESS_TOKEN
RUN fetch --repo="https://github.com/initia-labs/movevm" --tag="${LIBMOVEVM_VERSION}" --release-asset="libmovevm_muslc.*.a" /lib/
RUN fetch --repo="https://github.com/initia-labs/movevm" --tag="${LIBMOVEVM_VERSION}" --release-asset="libcompiler_muslc.*.a" /lib/
RUN cp /lib/libmovevm_muslc.`uname -m`.a /lib/libmovevm_muslc.a
RUN cp /lib/libcompiler_muslc.`uname -m`.a /lib/libcompiler_muslc.a

# force it to use static lib (from above) not standard libmovevm.so and libcompiler.so file
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LDFLAGS="-linkmode=external -extldflags \"-L/code/mimalloc/build -lmimalloc -Wl,-z,muldefs -static\"" make build

FROM alpine:3.19

RUN addgroup initia \
    && adduser -G initia -D -h /initia initia

WORKDIR /initia

COPY --from=go-builder /code/build/initiad /usr/local/bin/initiad

USER initia

# rest server
EXPOSE 1317
# grpc
EXPOSE 9090
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

CMD ["/usr/local/bin/initiad", "version"]
