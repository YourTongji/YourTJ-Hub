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
| mobile | widget/unit | flutter test (when mobile lands) |

## CI mapping

- ci-backend.yml: go vet + go test + go build (apps/gooseforum/app/**, main.go, go.mod, go.sum); the
  PostgreSQL integration tests in `app/bundles/connect/sqlconnect` are gated by `TEST_PG_DSN` and
  skip when unset (CI stays green without a PG service)
- ci-backend.yml also runs `ci-backend-pg`: a real `postgres:16-alpine` service + the migration
  schema tests in `app/migration/migration_pg_test.go` (`TestSchemaMigratesOnPostgreSQL`,
  `TestSchemaUpgradeCreatesNewTablesOnPostgreSQL`), gated by `YOURTJ_TEST_PG_URL` (set in CI, skipped
  locally when unset). **Any model/migration change must pass these PG tests** — models must not
  hardcode MySQL-only types (`bigint unsigned` / `datetime` / `tinyint`), which GORM renders verbatim
  and PostgreSQL rejects, silently leaving tables uncreated (issue #8 production regression).
- ci-frontend.yml: pnpm typecheck + site unit tests + build (apps/gooseforum/resource/**)
- ci-contract.yml: on every PR, installs the locked `packages/api-contract` pnpm tooling, runs OpenAPI
  lint + bundle + TypeScript generation, then rejects an uncommitted diff below
  `apps/gooseforum/resource/packages/client/src/gen`. It also runs on push when contract-relevant
  backend, generated types, contract tooling, CI/Make configuration, or contract/testing documentation
  changes. The route-level HTTP contract fixture tests run inside the backend `go test ./...` gate.

## Smoke checklist

```bash
curl http://localhost:5234/            # homepage HTML (three-mode rendering, GoHTML)
curl http://localhost:5234/api/...     # JSON API (per upstream routes)
# Frontend dev: http://localhost:3010
```
