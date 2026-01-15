# syntax=docker/dockerfile:1

# ==================== Build Arguments ====================
ARG GO_VERSION=1.22
ARG ALPINE_VERSION=3.19

# ==================== Build Stage ====================
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source and build with security flags
COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -trimpath \
    -o /app/api \
    ./cmd/api

# ==================== Development Stage ====================
FROM golang:${GO_VERSION}-alpine AS development

WORKDIR /app

RUN apk add --no-cache git ca-certificates

# Install air for hot reload
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

# Source will be mounted via volume
CMD ["air", "-c", ".air.toml"]

# ==================== Production Stage ====================
# Using scratch for minimal attack surface (~10MB final image)
FROM scratch AS production

# Import CA certificates and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy passwd file for non-root user (UID 65534 = nobody)
COPY --from=builder /etc/passwd /etc/passwd

# Copy binary
COPY --from=builder /app/api /api

# Use non-root user
USER 65534:65534

# Metadata labels
LABEL org.opencontainers.image.title="Penshort" \
      org.opencontainers.image.description="Developer-focused URL shortener" \
      org.opencontainers.image.source="https://github.com/HPNChanel/penshort" \
      org.opencontainers.image.vendor="Penshort" \
      org.opencontainers.image.licenses="MIT"

EXPOSE 8080

ENTRYPOINT ["/api"]

