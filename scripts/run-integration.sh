#!/usr/bin/env bash
# run-integration.sh - Integration test runner with automatic dependency management
# This script ensures Postgres and Redis are running before running integration tests

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi
}

# Start services if not running
start_services() {
    log_info "Checking Docker Compose services..."
    
    cd "$PROJECT_ROOT"
    
    # Check if services are already running
    if docker compose ps --services --filter "status=running" 2>/dev/null | grep -q postgres; then
        log_info "PostgreSQL is already running"
    else
        log_info "Starting PostgreSQL..."
        docker compose up -d postgres
    fi
    
    if docker compose ps --services --filter "status=running" 2>/dev/null | grep -q redis; then
        log_info "Redis is already running"
    else
        log_info "Starting Redis..."
        docker compose up -d redis
    fi
}

# Wait for services to be healthy
wait_for_services() {
    log_info "Waiting for services to be healthy..."
    
    local max_attempts=30
    local attempt=0
    
    # Wait for Postgres
    while [ $attempt -lt $max_attempts ]; do
        if docker compose exec -T postgres pg_isready -U penshort &> /dev/null; then
            log_info "PostgreSQL is ready"
            break
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    
    if [ $attempt -eq $max_attempts ]; then
        log_error "PostgreSQL did not become ready in time"
        exit 1
    fi
    
    # Wait for Redis
    attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if docker compose exec -T redis redis-cli ping &> /dev/null; then
            log_info "Redis is ready"
            break
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    
    if [ $attempt -eq $max_attempts ]; then
        log_error "Redis did not become ready in time"
        exit 1
    fi
}

# Apply migrations
apply_migrations() {
    log_info "Applying database migrations..."
    
    if ! command -v migrate &> /dev/null; then
        log_warn "migrate tool not found, skipping migrations"
        log_warn "Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
        return 0
    fi
    
    export DATABASE_URL="postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable"
    migrate -path "$PROJECT_ROOT/migrations" -database "$DATABASE_URL" up 2>/dev/null || {
        log_warn "Migrations may have already been applied"
    }
}

# Run integration tests
run_tests() {
    log_info "Running integration tests..."
    
    export DATABASE_URL="postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable"
    export REDIS_URL="redis://localhost:6379"
    export APP_ENV="test"
    
    cd "$PROJECT_ROOT"
    
    if [ -n "$1" ]; then
        # Run specific package
        go test -v -race -tags=integration -run Integration "$@"
    else
        # Run all integration tests
        go test -v -race -tags=integration -run Integration ./...
    fi
}

# Cleanup (optional, controlled by flag)
cleanup() {
    if [ "${INTEGRATION_CLEANUP:-false}" = "true" ]; then
        log_info "Cleaning up..."
        docker compose exec -T redis redis-cli FLUSHDB &> /dev/null || true
    fi
}

# Main
main() {
    log_info "=== Penshort Integration Test Runner ==="
    
    # Skip compose if flag is set
    if [ "${INTEGRATION_SKIP_COMPOSE:-false}" != "true" ]; then
        check_docker
        start_services
        wait_for_services
        apply_migrations
    else
        log_info "Skipping Docker Compose (INTEGRATION_SKIP_COMPOSE=true)"
    fi
    
    run_tests "$@"
    local exit_code=$?
    
    cleanup
    
    if [ $exit_code -eq 0 ]; then
        log_info "=== Integration tests passed ==="
    else
        log_error "=== Integration tests failed ==="
    fi
    
    exit $exit_code
}

main "$@"
