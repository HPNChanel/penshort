.PHONY: up down test lint migrate migrate-down seed build clean dev logs

# ==================== Development ====================

## Start all services (API + PostgreSQL + Redis)
up:
	docker compose up -d

## Stop all services
down:
	docker compose down

## View logs
logs:
	docker compose logs -f api

## Run API locally without Docker (requires local Go, Postgres, Redis)
dev:
	go run ./cmd/api

# ==================== Testing ====================

## Run all tests with race detector
test:
	go test -v -race -coverprofile=coverage.out ./...

## Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html

# ==================== Code Quality ====================

## Run linter
lint:
	golangci-lint run ./...

## Format code
fmt:
	go fmt ./...
	goimports -w .

# ==================== Database ====================

## Run all pending migrations
migrate:
	migrate -path migrations -database "$(DATABASE_URL)" up

## Rollback last migration
migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

## Create a new migration file
migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

## Seed database with sample data (Phase 2)
seed:
	@echo "No seed data in Phase 1"

# ==================== Build ====================

## Build binary for current OS
build:
	go build -o bin/api ./cmd/api

## Build production binary (Linux)
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/api ./cmd/api

## Build Docker image
docker-build:
	docker build -t penshort:latest --target production .

## Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# ==================== Dependencies ====================

## Download dependencies
deps:
	go mod download

## Tidy dependencies
tidy:
	go mod tidy

# ==================== Help ====================

## Show this help
help:
	@echo "Penshort - Developer URL Shortener"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  up            Start all services (Docker Compose)"
	@echo "  down          Stop all services"
	@echo "  logs          View API logs"
	@echo "  dev           Run API locally (without Docker)"
	@echo ""
	@echo "Testing:"
	@echo "  test          Run all tests with race detector"
	@echo "  test-coverage Generate HTML coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  lint          Run golangci-lint"
	@echo "  fmt           Format code"
	@echo ""
	@echo "Database:"
	@echo "  migrate       Run pending migrations"
	@echo "  migrate-down  Rollback last migration"
	@echo "  migrate-create Create new migration"
	@echo ""
	@echo "Build:"
	@echo "  build         Build binary for current OS"
	@echo "  build-prod    Build production binary (Linux)"
	@echo "  docker-build  Build Docker image"
	@echo "  clean         Remove build artifacts"
