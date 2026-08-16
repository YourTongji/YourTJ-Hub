# Testing Strategy & Commands

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-07

## Principles

- Verification strength scales with change risk (auth/PII/governance/points/search must include
  negative, replay, privacy, failure, and reconciliation cases).
- A local subset is not CI passing; report the commands actually run and their results.
- Upstream already has solid Go unit tests (controller layer, i18n rendering, SEO meta); keep them and
  add tests when modifying.
- Bug fixes start red: write the smallest failing test that reproduces the bug, run it to confirm the
  failure, then implement the fix and turn it green. Mechanical changes (rename, formatting, dependency
  bump, docs-only) are exempt; the failing test stays as a regression test.

## Commands

```bash
# Backend
cd apps/gooseforum && go vet ./... && go test ./...

# Frontend
cd apps/gooseforum/resource && pnpm typecheck && pnpm test && pnpm build

# Full
make test

# Contract: lint, bundle, and regenerate the committed OpenAPI TypeScript output
cd packages/api-contract && pnpm install --frozen-lockfile && pnpm run check
# Equivalent Make targets: make contract-lint, make contract-generate-ts, make contract-check

# Build smoke
make build && ./bin/yourtj-hub serve   # then curl http://localhost:5234
```

## Layers

| Layer | Test type | Tool |
|---|---|---|
| bundles | unit tests (utilities) | go test |
| models | model/migration tests | go test |
| service | business unit + transaction cases | go test + sqlmock or testcontainers (when decided) |
| http/controllers | handler + rendering tests (upstream has some) | go test + httptest |
| resource (frontend) | typecheck + component tests | vue-tsc + Vitest |
| contract | OpenAPI lint/bundle/type generation plus real Gin route-chain fixture assertions | pnpm + go test + httptest |
| mobile | widget/unit | flutter test (melos analyze + test; see local-development.md) |
| mobile OIDC | controller chain unit + E2E script | `auth/test/oidc_controller_test.dart` (authorize→exchange 调用链) + `scripts/oidc_e2e.sh` (本地内建 Provider → AppAuth 模拟器回跳 → exchange 验证) |

## Test layout

测试与被测代码放在一起，按语言惯例分层；**不要为迁移而迁移**，新测试遵守以下归属：

| 层 | 位置 | 约定 |
|---|---|---|
| Go 单元测试（bundles/models/service 内部逻辑） | 与被测文件同包同目录的 `*_test.go` | Go 工具链/覆盖率/重构联动依赖同包布局；白盒测试访问包内未导出符号 |
| Go 黑盒测试（外部契约/集成行为） | 同目录 `*_test.go`，`package xxx_test` | 只通过导出 API 验证行为；需要外部文件（样例输入、golden 输出）时放同包 `testdata/` |
| 前端组件/单元测试 | `apps/gooseforum/resource/test/*.test.ts` | Vitest 独立目录，避免混入 `src/`；fixtures 就近放 `test/fixtures/` |
| Flutter 测试 | 各包 `apps/mobile/packages/<pkg>/test/` | `widget_test.dart` / `*_test.dart`，fixtures 放同目录 |
| 契约测试（路由级 HTTP 断言） | `apps/gooseforum/app/http/routes/*_test.go` | 位于 Go module 内，随 `go test ./...` 运行 |
| 契约 fixtures 与生成类型 | `packages/api-contract/fixtures/` | 只放 fixture 数据与 OpenAPI 生成的 TS 类型；该目录不在 Go module 内，测试代码不放这里 |

模型/迁移测试必须同时满足 PG 门禁（见下方 CI mapping 的 `ci-backend-pg`）。

## CI mapping

All CI `push` triggers are limited to `dev` and `main`, so a push to an in-repository
PR branch is validated once by its `pull_request` run rather than again by a duplicate
`push` run. CI runs for the same PR or branch supersede older in-progress runs.

The required backend, frontend, and contract workflows start for every PR so their
required status checks cannot remain pending. Each required job performs its own path
detection, so a detection failure fails that required check; its heavy Go or pnpm steps
run only when its owned inputs changed. `ci-mobile` is not a required check and uses
path filters directly, so an unrelated PR does not start a Flutter runner.

- ci-backend.yml: changed backend or contract-fixture paths run go vet + go test + go build
  (apps/gooseforum/app/**, main.go, go.mod, go.sum, embedded resource Go/GoHTML files,
  markdown compatibility fixtures, packages/api-contract/**); the
  PostgreSQL integration tests in `app/bundles/connect/sqlconnect` are gated by `TEST_PG_DSN` and
  skip when unset (CI stays green without a PG service)
- ci-backend.yml also runs `ci-backend-pg` only for model, migration, SQL-connection, or Go module
  changes: a real `postgres:16-alpine` service + the migration
  schema tests in `app/migration/migration_pg_test.go` (`TestSchemaMigratesOnPostgreSQL`,
  `TestSchemaUpgradeCreatesNewTablesOnPostgreSQL`), gated by `YOURTJ_TEST_PG_URL` (set in CI, skipped
  locally when unset). **Any model/migration change must pass these PG tests** — models must not
  hardcode MySQL-only types (`bigint unsigned` / `datetime` / `tinyint`), which GORM renders verbatim
  and PostgreSQL rejects, silently leaving tables uncreated (issue #8 production regression).
- ci-frontend.yml: changed frontend paths run pnpm typecheck + site unit tests + build
  (apps/gooseforum/resource/** and shared markdown compatibility fixtures).
- ci-contract.yml: changed contract inputs install the locked `packages/api-contract` pnpm tooling, run OpenAPI
  lint + bundle + TypeScript generation, then rejects an uncommitted diff below
  `apps/gooseforum/resource/packages/client/src/gen`. Its inputs are the contract package, generated
  TypeScript, client package manifest, and its own workflow configuration. The route-level HTTP
  contract fixture tests run inside the backend `go test ./...` gate.
- ci-mobile.yml: changed mobile paths run melos bootstrap, analyze, and test (apps/mobile/**).

## Smoke checklist

```bash
curl http://localhost:5234/            # homepage HTML (three-mode rendering, GoHTML)
curl http://localhost:5234/api/...     # JSON API (per upstream routes)
# Frontend dev: http://localhost:3010
```
