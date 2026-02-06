.PHONY: all build run dev clean test lint help
.PHONY: backend-build backend-run backend-dev backend-test backend-lint
.PHONY: frontend-build frontend-run frontend-dev frontend-lint
.PHONY: docker-build docker-up docker-down

# Default target
all: build

# =============================================================================
# Main Commands
# =============================================================================

## build: Build both frontend and backend (production binary)
build:
	./build.sh

## run: Run the production binary
run: build
	./codesentry

## dev: Run both frontend and backend in development mode (requires two terminals)
dev:
	@echo "Run these commands in separate terminals:"
	@echo "  make backend-dev   # Terminal 1"
	@echo "  make frontend-dev  # Terminal 2"

## clean: Clean build artifacts
clean:
	rm -f codesentry
	rm -rf frontend/dist
	rm -rf backend/tmp

## test: Run all tests
test: backend-test

## lint: Run all linters
lint: backend-lint frontend-lint

# =============================================================================
# Backend Commands
# =============================================================================

## backend-build: Build the backend binary
backend-build:
	cd backend && go build -o ../codesentry ./cmd/server

## backend-run: Run the backend server
backend-run:
	cd backend && go run ./cmd/server

## backend-dev: Run backend with hot reload (requires air)
backend-dev:
	cd backend && air

## backend-test: Run backend tests
backend-test:
	cd backend && go test ./...

## backend-lint: Run backend linters
backend-lint:
	cd backend && go vet ./...
	@which golangci-lint > /dev/null && cd backend && golangci-lint run || echo "golangci-lint not installed, skipping"

## backend-tidy: Tidy Go modules
backend-tidy:
	cd backend && go mod tidy

# =============================================================================
# Frontend Commands
# =============================================================================

## frontend-build: Build the frontend for production
frontend-build:
	cd frontend && npm run build

## frontend-dev: Run frontend development server
frontend-dev:
	cd frontend && npm run dev

## frontend-lint: Run frontend linter
frontend-lint:
	cd frontend && npm run lint

## frontend-install: Install frontend dependencies
frontend-install:
	cd frontend && npm install

# =============================================================================
# Docker Commands
# =============================================================================

## docker-build: Build Docker image
docker-build:
	docker build -t codesentry:latest .

## docker-up: Start with Docker Compose (MySQL)
docker-up:
	docker-compose up -d

## docker-up-sqlite: Start with Docker Compose (SQLite)
docker-up-sqlite:
	docker-compose -f docker-compose.sqlite.yml up -d

## docker-up-postgres: Start with Docker Compose (PostgreSQL)
docker-up-postgres:
	docker-compose -f docker-compose.postgres.yml up -d

## docker-down: Stop Docker Compose
docker-down:
	docker-compose down

## docker-logs: Show Docker Compose logs
docker-logs:
	docker-compose logs -f

# =============================================================================
# Database Commands
# =============================================================================

## db-reset: Reset the database (SQLite only, development)
db-reset:
	rm -f backend/data/codesentry.db
	@echo "Database reset. Run 'make backend-run' to recreate."

# =============================================================================
# Help
# =============================================================================

## help: Show this help message
help:
	@echo "CodeSentry - AI-powered Code Review Platform"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Main Commands:"
	@grep -E '^## ' $(MAKEFILE_LIST) | grep -E '(build|run|dev|clean|test|lint):' | sed 's/## /  /' | column -t -s ':'
	@echo ""
	@echo "Backend Commands:"
	@grep -E '^## backend' $(MAKEFILE_LIST) | sed 's/## /  /' | column -t -s ':'
	@echo ""
	@echo "Frontend Commands:"
	@grep -E '^## frontend' $(MAKEFILE_LIST) | sed 's/## /  /' | column -t -s ':'
	@echo ""
	@echo "Docker Commands:"
	@grep -E '^## docker' $(MAKEFILE_LIST) | sed 's/## /  /' | column -t -s ':'
	@echo ""
	@echo "Database Commands:"
	@grep -E '^## db' $(MAKEFILE_LIST) | sed 's/## /  /' | column -t -s ':'
