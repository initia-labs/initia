# 빌드 단계
FROM --platform=$BUILDPLATFORM golang:1.22 AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# 필요한 빌드 도구 설치
RUN apt-get update && apt-get install -y git make

# 소스 코드 복사
COPY . .

# MOVEVM 버전 설정 (필요에 따라 조정)
ENV MOVEVM_VERSION=v0.2.12

# 디버깅을 위한 환경 출력
RUN echo "TARGETOS: $TARGETOS" && \
    echo "TARGETARCH: $TARGETARCH" && \
    echo "Current directory:" && \
    pwd && \
    echo "Directory contents:" && \
    ls -la

# TARGETOS와 TARGETARCH에 따라 빌드 명령 실행
RUN if [ "$TARGETOS" = "linux" ]; then \
        echo "Building for Linux" && \
        make build-linux-with-shared-library ARCH=$TARGETARCH || (echo "Make failed" && cat Makefile && exit 1); \
    elif [ "$TARGETOS" = "darwin" ]; then \
        echo "Building for Darwin" && \
        make build ARCH=$TARGETARCH || (echo "Make failed" && cat Makefile && exit 1); \
    else \
        echo "Unknown OS: $TARGETOS" && exit 1; \
    fi

# 실행 단계
FROM --platform=$TARGETPLATFORM golang:1.22

WORKDIR /app

# 빌드된 바이너리와 라이브러리 복사
COPY --from=builder /src/build/initiad /app/
COPY --from=builder /src/build/*.so /app/
COPY --from=builder /src/build/*.dylib /app/

# 실행 명령 설정
CMD ["/app/initiad", "version"]
