# ─────────────────────────────────────────────────────────────
# Stage 1: Builder — generate proto + compile Go binary
# Menggunakan golang:1.21-bookworm (Debian) agar apt tersedia
# ─────────────────────────────────────────────────────────────
FROM golang:1.23-bookworm AS builder

# Install protoc via apt (lebih stabil dari Alpine apk)
RUN apt-get update && apt-get install -y --no-install-recommends \
        protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

# Pin versi eksplisit agar tidak fetch @latest
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

WORKDIR /build

# Copy proto
COPY proto/ ./proto/

# Copy go.mod gateway agar go mod tidy bisa berjalan
COPY gateway/go.mod ./gateway/
WORKDIR /build/gateway
RUN go mod tidy

# Copy seluruh source gateway (setelah tidy agar cache layer optimal)
WORKDIR /build
COPY gateway/ ./gateway/

# Generate Go code dari proto ke gateway/generated
RUN mkdir -p gateway/generated && \
    protoc \
      --proto_path=proto \
      --go_out=gateway/generated \
      --go_opt=paths=source_relative \
      --go-grpc_out=gateway/generated \
      --go-grpc_opt=paths=source_relative \
      proto/finance.proto

# Build binary statis
WORKDIR /build/gateway
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/gateway ./main.go

# ─────────────────────────────────────────────────────────────
# Stage 2: Runtime — image minimal
# ─────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/gateway ./gateway

EXPOSE 8000

CMD ["./gateway"]
