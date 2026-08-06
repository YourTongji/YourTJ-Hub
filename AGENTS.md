# AGENTS.md — yourtj-hub

Operating guide for anyone (human or AI agent) changing this repository.

Before changing anything, read this file, [`docs/README.md`](docs/README.md),
[`docs/development/README.md`](docs/development/README.md), and the product/architecture/operations
documents directly affected by the request. Use the repository `$yourtj-development` skill for
implementation, testing, review, CI, or PR work.

---

## 1. What this is

yourtj-hub 是同济大学校级论坛平台 monorepo（品牌：yourtj，与已归档的 YourTJ-Platform 区分）。
论坛是核心产品；统一认证（Casdoor）、搜索（Meilisearch）和积分（credit，二期）是共享基础设施
的子域。项目**不 fork GooseForum**，以其为参考蓝本自研，允许在数据库、搜索和结构上大改。

- Backend: **Go 1.26** — Gin，`apps/server` 单模块，`internal/` 按层分包（config/domain/repository/service/http/auth/search）。
- Database: 建议 **PostgreSQL**（决策待定，见 docs/product/current-state.md），repository 层抽象可换库。
- Search: **Meilisearch**（独立服务，server 代理统一鉴权）。
- Web: **Vue 3 + Vite + TypeScript**（pnpm workspace，`apps/web`），产物 go:embed 进 server 单二进制。
- Mobile: **Flutter**（`apps/mobile`，melos workspace，Riverpod，规划中）。
- Auth: **Casdoor OIDC** 统一认证，`sub` = 数字用户 ID（已实测：应用 ID 规则 Incremental / 显式数字 id）。
- Contract: `packages/api-contract/openapi.yaml` 是单一事实源（swag 接入后生成）。
- Points: credit（linux-do）二期，积分走商户模型，本期不实现。

## 2. Repository layout & boundary rules

```
apps/
  server/    Go 后端 —— cmd/server 仅组装；internal/{config,domain,repository,service,http,auth,search}
             webdist/ 是 web 构建产物（go:embed），gitignore，只保留 .gitkeep
  web/       Vue 3 前端源码，构建产物输出到 ../server/webdist
  mobile/    Flutter melos workspace（core/auth/ui_kit/forum_app）
packages/
  api-contract/  openapi.yaml + gen 脚本 + fixtures + 契约测试
services/
  casdoor/   统一认证部署配置（数字 ID 初始化清单）
  search/    Meilisearch 部署配置
  credit/    积分（二期占位）
third_party/
  gooseforum/  上游参考源码快照（不参与构建，见 README「参考上游」）
deploy/      分环境 compose + env.example
docs/        文档中心（product/architecture/development/operations）
```

**Boundary rules**
- `apps/server/internal/repository` 是唯一允许触碰 DB 的层；业务逻辑在 `service`，HTTP 在 `http`。
- 跨域访问走 owner 的公开 API，禁止 foreign SQL 直读其他域的表。
- Web/Mobile 消费 `packages/api-contract` 生成的类型，禁止手写与后端重复的 DTO。
- `services/` 只放部署配置，不放第三方源码（Casdoor/Meilisearch/credit 是现成组件）。
- `third_party/` 只读参考，不参与构建、不改动（更新走 README 记录的 rsync 流程）。

## 3. Hard constraints

- 认证唯一来源是 **Casdoor**；论坛 JWT 只是会话凭证，不是身份事实源。
- 用户 ID 必须是**数字**（uint64），因为 credit 的 `GetID()` 只接受数字 sub——UUID 会让所有用户解析为 0 互相覆盖。Casdoor 必须配 Incremental ID 规则或显式数字 id。
- 契约变更必须同 PR 更新：Go struct → openapi.yaml → TS/Dart 生成物 → fixture 契约测试。
- 部署形态是**单二进制**（go:embed webdist），禁止引入 nginx/CDN 分离部署。
- 文档使用四种实现状态词（`Current`/`Partial`/`Planned`/`Decision needed`），见 docs/README.md。
- 文档只描述当前支持的行为模型，不写时间线/里程碑（见 docs/development/documentation.md）。

## 4. Verification

- Backend: `cd apps/server && go vet ./... && go test ./...`
- Web: `cd apps/web && pnpm typecheck && pnpm build`（产物进 server/webdist）
- Full build: `make build`（web → go build 单二进制）
- Contract: `packages/api-contract` 的 fixture 契约测试（契约管线接入后）
- 报告实际运行的命令与结果；本地子集不等于 CI 通过。

## 5. Git & PR discipline

- 不在 `main` 上直接开发；从 `origin/main` 建 `feat/<topic>` 或 `fix/<topic>` 分支。
- 只提交本任务明确拥有的文件；保留无关的 dirty/untracked 文件。
- 只在用户明确要求时 commit/push/开 PR。Agent 提交必须带
  `Co-authored-by: synergy-agent <299070056+synergy-agent@users.noreply.github.com>` 尾注。
- 不 push 到受保护分支；发布走 PR + CI。
- 提交信息用 concise conventional type（`feat:`/`fix:`/`docs:`/`refactor:`/`chore:`）。

## 6. Reference

- [文档中心](docs/README.md)（事实来源表 + 状态词）
- [开发入口](docs/development/README.md)
- 架构决策记录保存在项目 note（yourtj-hub 架构决策记录），不落盘 git
- 上游参考：GooseForum（`third_party/gooseforum/` 快照）、YourTJ-Platform（本地，同品牌已归档仓库）
