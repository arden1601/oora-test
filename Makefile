.PHONY: up down test fmt lint frontend

up: ## start PostgreSQL + backend
	docker compose up --build

down: ## stop and remove containers (keeps the database volume)
	docker compose down

test: ## run all backend tests
	cd backend && go test ./...

fmt: ## format Go code
	cd backend && gofmt -l -w .

lint: ## static checks
	cd backend && go vet ./...

frontend: ## run the admin screen in dev mode
	cd frontend && pnpm install && pnpm dev
