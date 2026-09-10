# =============================================================================
# Multi-Stage Dockerfile for AI Tech Pulse Digest
# =============================================================================

# --- Stage 1: Build Binary ---
FROM golang:alpine AS builder

WORKDIR /src

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

ENV GOTOOLCHAIN=local

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary without CGO
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags '-static'" \
    -trimpath \
    -o /bin/server ./cmd/server

# --- Stage 2: Final Minimal Runtime Image ---
FROM alpine:3.20

WORKDIR /app

# Install runtime SSL certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy compiled binary from builder
COPY --from=builder /bin/server /app/server
COPY --from=builder /src/migrations /app/migrations

# Run as non-root user for security
USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/server"]
