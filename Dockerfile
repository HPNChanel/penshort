# syntax=docker/dockerfile:1

# ==================== Build Stage ====================
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api ./cmd/api

# ==================== Development Stage ====================
FROM golang:1.22-alpine AS development

WORKDIR /app

RUN apk add --no-cache git ca-certificates

# Install air for hot reload
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

# Source will be mounted via volume
CMD ["air", "-c", ".air.toml"]

# ==================== Production Stage ====================
FROM alpine:3.19 AS production

WORKDIR /app

# Install CA certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Add non-root user
RUN adduser -D -g '' appuser
USER appuser

# Copy binary from builder
COPY --from=builder /api .

EXPOSE 8080

CMD ["./api"]
