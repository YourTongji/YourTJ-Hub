.PHONY: dev down server web build test gen contract-lint contract-generate-ts contract-check hooks

dev: ## Start local dependencies (postgres + meilisearch)
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

contract-lint: ## Validate and bundle the OpenAPI contract
	cd packages/api-contract && pnpm install --frozen-lockfile && pnpm run lint && pnpm run bundle

contract-generate-ts: ## Generate OpenAPI TypeScript types for @gooseforum/client
	cd packages/api-contract && pnpm install --frozen-lockfile && pnpm run generate:ts

contract-check: ## Validate, bundle, generate, and require committed OpenAPI TypeScript output
	cd packages/api-contract && pnpm install --frozen-lockfile && pnpm run check
	git diff --exit-code -- apps/gooseforum/resource/packages/client/src/gen

test: ## Run backend, contract, and frontend checks
	cd apps/gooseforum && go vet ./... && go test ./...
	$(MAKE) contract-check
	cd apps/gooseforum/resource && pnpm typecheck && pnpm test

gen: contract-generate-ts ## Generate currently supported API client artifacts

hooks: ## Install/verify local git hooks (lefthook)
	@if command -v lefthook >/dev/null 2>&1; then \
		lefthook install; \
	else \
		echo "lefthook not found — run: brew install lefthook"; \
		exit 1; \
	fi
