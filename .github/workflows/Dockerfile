# 빌드 단계
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# 필요한 빌드 도구 설치
RUN apk add --no-cache git make

# 소스 코드 복사
COPY . .

# MOVEVM 버전 설정 (필요에 따라 조정)
ENV MOVEVM_VERSION=v0.2.12

# TARGETOS와 TARGETARCH에 따라 빌드 명령 실행
RUN if [ "$TARGETOS" = "linux" ]; then \
        make build-linux-with-shared-library ARCH=$TARGETARCH; \
    elif [ "$TARGETOS" = "darwin" ]; then \
        make build ARCH=$TARGETARCH; \
    fi

# 실행 단계
FROM --platform=$TARGETPLATFORM alpine

WORKDIR /app

# 빌드된 바이너리와 라이브러리 복사
COPY --from=builder /src/build/initiad /app/
COPY --from=builder /src/build/*.so /app/
COPY --from=builder /src/build/*.dylib /app/

# 실행 명령 설정
CMD ["/app/initiad", "version"]
