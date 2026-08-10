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

## CI mapping

All CI `push` triggers are limited to `dev` and `main`, so a push to an in-repository
PR branch is validated once by its `pull_request` run rather than again by a duplicate
`push` run. CI runs for the same PR or branch supersede older in-progress runs.

The required backend, frontend, and contract workflows start for every PR so their
required status checks cannot remain pending. Each first detects the changed paths and
skips its heavy job when its owned inputs are unchanged. `ci-mobile` is not a required
check and uses path filters directly, so an unrelated PR does not start a Flutter runner.

- ci-backend.yml: changed backend or contract-fixture paths run go vet + go test + go build
  (apps/gooseforum/app/**, main.go, go.mod, go.sum, packages/api-contract/**); the
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
  (apps/gooseforum/resource/**).
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
