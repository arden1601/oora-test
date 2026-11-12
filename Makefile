.PHONY: up down test fmt lint frontend

# Local overrides from .env, if there is one. The defaults match docker-compose.yml.
-include .env
POSTGRES_USER ?= portal
POSTGRES_PASSWORD ?= local-dev-only
POSTGRES_DB ?= portal
DB_HOST_PORT ?= 5433

up: ## start PostgreSQL + backend
	docker compose up --build

down: ## stop and remove containers (keeps the database volume)
	docker compose down

test: ## run all backend tests (starts PostgreSQL first; the tests use their own schema)
	docker compose up -d --wait db
	cd backend && TEST_DATABASE_URL="postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(DB_HOST_PORT)/$(POSTGRES_DB)?sslmode=disable" go test ./...

fmt: ## format Go code
	cd backend && gofmt -l -w .

lint: ## static checks
	cd backend && go vet ./...

frontend: ## run the admin screen in dev mode
	cd frontend && pnpm install && pnpm dev
