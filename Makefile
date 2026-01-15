.PHONY: setup doctor up down down-clean logs dev
.PHONY: migrate migrate-down migrate-create seed
.PHONY: test test-unit test-integration test-contract test-e2e test-bench bench
.PHONY: lint fmt security test-security docs-check
.PHONY: verify verify-full build build-prod docker-build clean deps tidy help

# ==================== Environment ====================

## Diagnose environment issues

doctor:
	@chmod +x ./scripts/doctor.sh 2>/dev/null || true
	@./scripts/doctor.sh

## One-time setup (download dependencies)
setup:
	go mod download
	@echo "Dependencies downloaded."
	@echo ""
	@echo "Required tools for verify:"
	@echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
	@echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
	@echo "  go install golang.org/x/vuln/cmd/govulncheck@latest"
	@echo "  go install github.com/securego/gosec/v2/cmd/gosec@latest"
	@echo "  go install github.com/zricethezav/gitleaks/v8@latest"
	@echo ""
	@echo "Optional tools:"
	@echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
	@echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
	@echo "  go install golang.org/x/vuln/cmd/govulncheck@latest"

# ==================== Development ====================

## Start all services (API + PostgreSQL + Redis)
up:
	docker compose up -d

## Stop all services

down:
	docker compose down

## Stop all services and remove volumes
down-clean:
	docker compose down -v

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

## Run unit tests only (fast, no Docker required)
test-unit:
	go test -v -race -short -coverprofile=coverage.out ./...

## Run integration tests (requires Docker services)
test-integration:
	go test -v -race -tags=integration -run Integration ./...

## Run contract tests (OpenAPI schema validation)
test-contract:
	@if [ -d "internal/contract" ]; then \
		go test -v -tags=contract ./internal/contract/...; \
	else \
		echo "No contract tests found. Skipping."; \
	fi

## Run end-to-end tests (full stack)
test-e2e:
	@chmod +x ./scripts/run-e2e.sh 2>/dev/null || true
	@./scripts/run-e2e.sh

## Run performance benchmarks
test-bench:
	@if [ -d "internal/benchmark" ]; then \
		go test -v -tags=bench -bench=. -benchmem ./internal/benchmark/...; \
	else \
		echo "No benchmark tests found. Skipping."; \
	fi

## Run performance benchmarks (alias)
bench: test-bench

## Run security checks (secrets + dependencies + SAST)
security:
	@chmod +x ./scripts/security.sh 2>/dev/null || true
	@./scripts/security.sh

## Run security checks (alias)
test-security: security

## Validate documentation examples
docs-check:
	@chmod +x ./scripts/validate-docs.sh 2>/dev/null || true
	@./scripts/validate-docs.sh

## Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ==================== Verification ====================

## Full verification pipeline (run this before submitting PRs)
verify:
	@chmod +x ./scripts/verify.sh 2>/dev/null || true
	@./scripts/verify.sh

## Full verification including E2E (alias)
verify-full: verify

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
	@echo "Environment:"
	@echo "  doctor        Diagnose environment issues"
	@echo "  setup         One-time setup (download dependencies)"
	@echo ""
	@echo "Development:"
	@echo "  up            Start all services (Docker Compose)"
	@echo "  down          Stop all services"
	@echo "  down-clean    Stop all services and remove volumes"
	@echo "  logs          View API logs"
	@echo "  dev           Run API locally (without Docker)"
	@echo ""
	@echo "Testing:"
	@echo "  test          Run all tests with race detector"
	@echo "  test-unit     Run unit tests only (no Docker)"
	@echo "  test-integration  Run integration tests"
	@echo "  test-contract     Run contract tests"
	@echo "  test-e2e          Run end-to-end tests"
	@echo "  test-bench        Run performance benchmarks"
	@echo "  bench             Run performance benchmarks"
	@echo "  test-security     Run security checks"
	@echo "  security          Run security checks"
	@echo "  docs-check        Validate documentation examples"
	@echo "  test-coverage     Generate HTML coverage report"
	@echo ""
	@echo "Verification:"
	@echo "  verify        Full verification pipeline (before PRs)"
	@echo "  verify-full   Full verification including E2E"
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
