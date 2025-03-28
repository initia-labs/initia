# Stage 1: Build the Go project
FROM golang:1.23-alpine AS go-builder

# Use build arguments for the target architecture
ARG TARGETARCH
ARG GOARCH
ARG VERSION
ARG COMMIT

# See https://github.com/initia-labs/movevm/releases
ENV LIBMOVEVM_VERSION=v1.0.0-rc.1
ENV MIMALLOC_VERSION=v2.2.2

# Install necessary packages
RUN set -eux; apk add --no-cache ca-certificates build-base git cmake

WORKDIR /code
COPY . /code/

# Install mimalloc
RUN git clone -b ${MIMALLOC_VERSION} --depth 1 https://github.com/microsoft/mimalloc; cd mimalloc; mkdir build; cd build; cmake ..; make -j$(nproc); make install
ENV MIMALLOC_RESERVE_HUGE_OS_PAGES=4

# Determine GOARCH and download the appropriate libraries
RUN set -eux; \
    case "${TARGETARCH}" in \
        "amd64") export GOARCH="amd64"; ARCH="x86_64";; \
        "arm64") export GOARCH="arm64"; ARCH="aarch64";; \
        *) echo "Unsupported architecture: ${TARGETARCH}"; exit 1;; \
    esac; \
    echo "Using GOARCH=${GOARCH} and ARCH=${ARCH}"; \
    wget -O /lib/libmovevm_muslc.${ARCH}.a https://github.com/initia-labs/movevm/releases/download/${LIBMOVEVM_VERSION}/libmovevm_muslc.${ARCH}.a; \
    wget -O /lib/libcompiler_muslc.${ARCH}.a https://github.com/initia-labs/movevm/releases/download/${LIBMOVEVM_VERSION}/libcompiler_muslc.${ARCH}.a; \
    cp /lib/libmovevm_muslc.${ARCH}.a /lib/libmovevm_muslc.a; \
    cp /lib/libcompiler_muslc.${ARCH}.a /lib/libcompiler_muslc.a

# Verify the library hashes (optional, uncomment if needed)
# RUN sha256sum /lib/libmovevm_muslc.${ARCH}.a | grep ...
# RUN sha256sum /lib/libcompiler_muslc.${ARCH}.a | grep ...

# Build the project with the specified architecture and linker flags
RUN VERSION=${VERSION} COMMIT=${COMMIT} LEDGER_ENABLED=false BUILD_TAGS=muslc GOARCH=${GOARCH} LDFLAGS="-linkmode=external -extldflags \"-L/code/mimalloc/build -lmimalloc -Wl,-z,muldefs -static\"" make build

# Stage 2: Create the final image
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
