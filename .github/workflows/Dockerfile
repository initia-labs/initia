ARG OS
ARG ARCH
FROM --platform=$OS/$ARCH golang:1.20

ARG OS
ARG ARCH
ARG L1_VERSION
ARG L1_NETWORK_NAME
ARG MOVEVM_VERSION

WORKDIR /workspace

# 필요한 도구 설치
RUN apt-get update && apt-get install -y make tar gzip

# 소스 코드 복사
COPY . .

# 환경 변수 설정
ENV OS=$OS
ENV ARCH=$ARCH
ENV L1_VERSION=$L1_VERSION
ENV L1_NETWORK_NAME=$L1_NETWORK_NAME
ENV MOVEVM_VERSION=$MOVEVM_VERSION

# 빌드 스크립트 실행
RUN if [ "$OS" = "darwin" ]; then \
        cd initia \
        && make build \
        && cd ./build \
        && cp /go/pkg/mod/github.com/initia-labs/movevm@${MOVEVM_VERSION}/api/libmovevm.dylib ./ \
        && cp /go/pkg/mod/github.com/initia-labs/movevm@${MOVEVM_VERSION}/api/libcompiler.dylib ./ \
        && tar -czvf initia_${L1_VERSION}_Darwin_${ARCH}.tar.gz initiad libmovevm.dylib libcompiler.dylib \
        && mkdir -p ../../networks/${L1_NETWORK_NAME}/binaries/ \
        && mv ./initia_${L1_VERSION}_Darwin_${ARCH}.tar.gz ../../networks/${L1_NETWORK_NAME}/binaries/ ; \
    elif [ "$OS" = "linux" ]; then \
        cd initia \
        && make build-linux-with-shared-library \
        && cd ./build \
        && mv libmovevm.so libmovevm.${ARCH}.so \
        && mv libcompiler.so libcompiler.${ARCH}.so \
        && tar -czvf initia_${L1_VERSION}_Linux_${ARCH}.tar.gz ./initiad libmovevm.${ARCH}.so libcompiler.${ARCH}.so \
        && mkdir -p ../../networks/${L1_NETWORK_NAME}/binaries/ \
        && mv ./initia_${L1_VERSION}_Linux_${ARCH}.tar.gz ../../networks/${L1_NETWORK_NAME}/binaries/ ; \
    fi

# 결과물 복사
RUN mkdir -p /output && cp networks/${L1_NETWORK_NAME}/binaries/initia_${L1_VERSION}_*_${ARCH}.tar.gz /output/
