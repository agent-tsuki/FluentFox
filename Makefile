# FluentFox API — Makefile
# This is the single interface for all developer commands.
# Run `make help` to see available targets.

.PHONY: help dev dev-down build test test-coverage migrate-up migrate-down \
        migrate-new seed sync-content lint tidy docker-build

MIGRATE := migrate -path db/migrations -database "$(DATABASE_URL)"

help: ## Show available make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Development ────────────────────────────────────────────────────────────────

dev: ## Start all services (API + Postgres) with hot reload
	docker compose up

dev-down: ## Stop containers without deleting named volumes
	docker compose down

# ── Build ──────────────────────────────────────────────────────────────────────

build: ## Build production binary locally using Dockerfile.prod
	docker build -f docker/Dockerfile.prod -t fluentfox-api:latest .
	@echo "✓ Production image built: fluentfox-api:latest"

docker-build: build ## Alias for build

# ── Testing ────────────────────────────────────────────────────────────────────

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

# ── Database ───────────────────────────────────────────────────────────────────

migrate-up: ## Apply all pending migrations
	$(MIGRATE) up

migrate-down: ## Roll back exactly one migration
	$(MIGRATE) down 1

migrate-new: ## Create a new migration pair (usage: make migrate-new name=add_something)
ifndef name
	$(error name is required. Usage: make migrate-new name=add_something)
endif
	$(MIGRATE) create -ext sql -dir db/migrations -seq $(name)

seed: ## Run all seed SQL files against the dev database
	@for f in db/seeds/*.sql; do \
	  echo "Seeding $$f..."; \
	  psql "$(DATABASE_URL)" -f $$f; \
	done
	@echo "✓ Seeding complete"

# ── Content ────────────────────────────────────────────────────────────────────

sync-content: ## Parse MDX files and upsert into the dev database
	go run ./cmd/sync-content

# ── Code quality ───────────────────────────────────────────────────────────────

lint: ## Run golangci-lint
	golangci-lint run --config .golangci.yml ./...

tidy: ## Run go mod tidy and verify
	go mod tidy
	go mod verify
	@echo "✓ go.mod and go.sum are tidy"
