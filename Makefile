# FluentFox API — Makefile
# Run `make help` to see available targets.

.PHONY: help dev dev-down build test test-coverage \
        migrate migrate-status migrate-down seed lint tidy update-disposable-domains

DEV_DB_URL  := postgres://fluentfox:fluentfox_dev@localhost:5435/fluentfox_dev?sslmode=disable
TEST_DB_URL := postgres://fluentfox:fluentfox_test@localhost:5431/fluentfox_test?sslmode=disable

help: ## Show available make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

# Development

dev: ## Start all services (API + Postgres) with hot reload
	docker compose up --build -d

dev-down: ## Stop containers without deleting named volumes
	docker compose down

# Build

build: ## Build production binary locally using Dockerfile.prod
	docker build -f docker/Dockerfile.prod -t fluentfox-api:latest .
	@echo "✓ Production image built: fluentfox-api:latest"

# Testing

test: ## Run all tests against an isolated test database
	docker compose -f docker-compose.test.yml up -d --wait
	APP_ENV=test go test -race ./... ; \
	  EXIT_CODE=$$? ; \
	  docker compose -f docker-compose.test.yml down ; \
	  exit $$EXIT_CODE

test-coverage: ## Run tests with coverage report
	docker compose -f docker-compose.test.yml up -d --wait
	APP_ENV=test go test -race -coverprofile=coverage.out ./... ; \
	  EXIT_CODE=$$? ; \
	  docker compose -f docker-compose.test.yml down ; \
	  go tool cover -html=coverage.out -o coverage.html ; \
	  xdg-open coverage.html 2>/dev/null || open coverage.html 2>/dev/null ; \
	  exit $$EXIT_CODE

# Database migrations (goose, SQL files in migrations/)

migrate: ## Apply all pending goose migrations
	DATABASE_URL="$(DEV_DB_URL)" go run ./cmd/migrate up

migrate-status: ## Show which goose migrations are applied
	DATABASE_URL="$(DEV_DB_URL)" go run ./cmd/migrate status

migrate-down: ## Roll back the most recent goose migration
	DATABASE_URL="$(DEV_DB_URL)" go run ./cmd/migrate down

# Content

seed: ## Run all seed SQL files against the dev database
	@for f in db/seeds/*.sql; do \
	  echo "Seeding $$f..."; \
	  psql "$(DEV_DB_URL)" -f $$f; \
	done
	@echo "✓ Seeding complete"

# Email domain checks

DISPOSABLE_DOMAINS_URL := https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/main/disposable_email_blocklist.conf

update-disposable-domains: ## Refresh the embedded disposable-email blocklist (CC0); review the diff, then commit
	curl -fsSL $(DISPOSABLE_DOMAINS_URL) -o pkg/emailcheck/data/disposable_email_blocklist.conf
	@wc -l pkg/emailcheck/data/disposable_email_blocklist.conf

# Code quality

lint: ## Run golangci-lint
	golangci-lint run --config .golangci.yml ./...

tidy: ## Run go mod tidy and verify
	go mod tidy
	go mod verify
	@echo "✓ go.mod and go.sum are tidy"
