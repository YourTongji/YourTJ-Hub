.PHONY: dev down server web build test gen

dev: ## 起本地依赖（postgres + meilisearch + mariadb + casdoor）
	docker compose up -d

down: ## 停本地依赖
	docker compose down

server: ## 跑后端（:8080）
	cd apps/server && go run ./cmd/server

web: ## 跑前端 dev（:5173，代理 /api 到 :8080）
	cd apps/web && pnpm dev

build: ## 构建 web 产物 + server 单二进制（go:embed）
	cd apps/web && pnpm build
	cd apps/server && go build -o ../../bin/yourtj-hub ./cmd/server

test: ## 全量测试
	cd apps/server && go vet ./... && go test ./...
	cd apps/web && pnpm typecheck

gen: ## 契约生成（openapi → ts/dart），M3 接入 swag 后启用
	@echo "TODO(M3): swag 生成 openapi.yaml + gen-ts/gen-dart"
