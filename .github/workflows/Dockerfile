# Stage 1: Build the Go project
FROM golang:1.22-alpine AS go-builder

# Use build arguments for the target architecture
ARG TARGETARCH

# Set environment variables
ENV GOOS=linux
ENV GOARCH=${TARGETARCH}

# Set the working directory
WORKDIR /code

# Copy the source code
COPY . .

# Download dependencies and build the project
RUN go mod download
RUN go build -o build/initiad ./cmd/initiad

# Stage 2: Prepare the final image
FROM alpine:3.19

# Copy the binary from the builder stage
COPY --from=go-builder /code/build/initiad /usr/local/bin/initiad

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/initiad"]
