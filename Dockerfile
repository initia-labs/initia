# Stage 1: Build the Go project
FROM golang:1.23-alpine AS go-builder

# Use build arguments for the target architecture
ARG TARGETARCH
ARG GOARCH=${TARGETARCH}

# MoveVM Version
ENV LIBMOVEVM_VERSION=v0.6.1

# Install necessary packages
RUN set -eux; apk add --no-cache ca-certificates build-base git cmake

WORKDIR /code
COPY . /code/

# Install mimalloc
RUN git clone --depth 1 https://github.com/microsoft/mimalloc; \
    cd mimalloc; mkdir build; cd build; \
    cmake ..; make -j$(nproc); make install

ENV MIMALLOC_RESERVE_HUGE_OS_PAGES=4

# Determine architecture-specific libraries
RUN set -eux; \
    case "${TARGETARCH}" in \
        "amd64") ARCH="x86_64";; \
        "arm64") ARCH="aarch64";; \
        *) echo "Unsupported architecture: ${TARGETARCH}"; exit 1;; \
    esac; \
    echo "Using GOARCH=${GOARCH} and ARCH=${ARCH}"; \
    wget -O /lib/libmovevm_muslc.${ARCH}.a https://github.com/initia-labs/movevm/releases/download/${LIBMOVEVM_VERSION}/libmovevm_muslc.${ARCH}.a; \
    wget -O /lib/libcompiler_muslc.${ARCH}.a https://github.com/initia-labs/movevm/releases/download/${LIBMOVEVM_VERSION}/libcompiler_muslc.${ARCH}.a; \
    cp /lib/libmovevm_muslc.${ARCH}.a /lib/libmovevm_muslc.a; \
    cp /lib/libcompiler_muslc.${ARCH}.a /lib/libcompiler_muslc.a; \
    sha256sum /lib/libmovevm_muslc.${ARCH}.a | grep EXPECTED_HASH; \
    sha256sum /lib/libcompiler_muslc.${ARCH}.a | grep EXPECTED_HASH

# Build the project with architecture-specific flags
RUN set -eux; \
    CGO_ENABLED=1 BUILD_TAGS=muslc GOARCH=${GOARCH} \
    LDFLAGS="-linkmode=external -extldflags '-L/code/mimalloc/build -lmimalloc -Wl,-z,muldefs -static -static-libgcc'" \
    make build

# Stage 2: Create the final image
FROM alpine:3.19

# Add user and group for security
RUN addgroup -S initia && adduser -S -G initia -h /initia initia

WORKDIR /initia

# Copy built binary from previous stage
COPY --from=go-builder /code/build/initiad /usr/local/bin/initiad

USER initia

# Expose necessary ports
EXPOSE 1317 9090 26656 26657

# Run the application
CMD ["/usr/local/bin/initiad", "version"]
