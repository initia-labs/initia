ARG ARCH
FROM --platform=linux/${ARCH} golang:1.20

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
ENV ARCH=$ARCH
ENV L1_VERSION=$L1_VERSION
ENV L1_NETWORK_NAME=$L1_NETWORK_NAME
ENV MOVEVM_VERSION=$MOVEVM_VERSION

# 빌드 스크립트 실행
RUN cd initia \
    && make build-linux-with-shared-library \
    && cd ./build \
    && mv libmovevm.so libmovevm.${ARCH}.so \
    && mv libcompiler.so libcompiler.${ARCH}.so \
    && tar -czvf initia_${L1_VERSION}_Linux_${ARCH}.tar.gz ./initiad libmovevm.${ARCH}.so libcompiler.${ARCH}.so \
    && mkdir -p ../../networks/${L1_NETWORK_NAME}/binaries/ \
    && mv ./initia_${L1_VERSION}_Linux_${ARCH}.tar.gz ../../networks/${L1_NETWORK_NAME}/binaries/

# 결과물 복사
RUN mkdir -p /output && cp networks/${L1_NETWORK_NAME}/binaries/initia_${L1_VERSION}_Linux_${ARCH}.tar.gz /output/
