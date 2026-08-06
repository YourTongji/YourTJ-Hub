.PHONY: dev down server web build test gen

dev: ## Start local dependencies (postgres + meilisearch + mariadb + casdoor)
	docker compose up -d

down: ## Stop local dependencies
	docker compose down

server: ## Run forum backend (default port 5234; place config.toml in apps/gooseforum first)
	cd apps/gooseforum && go run . serve

web: ## Run frontend dev server (:3010; run pnpm install first)
	cd apps/gooseforum/resource && pnpm dev

build: ## Build frontend output + forum single binary
	cd apps/gooseforum/resource && pnpm build
	cd apps/gooseforum && go build -o ../../bin/yourtj-hub .

test: ## Run all tests
	cd apps/gooseforum && go vet ./... && go test ./...
	cd apps/gooseforum/resource && pnpm typecheck

gen: ## Contract generation (openapi → ts/dart), enabled once the contract pipeline exists
	@echo "TODO: swag generates openapi.yaml + gen-ts/gen-dart"
