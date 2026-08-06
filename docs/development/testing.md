# Testing Strategy & Commands

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-06

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
cd apps/gooseforum/resource && pnpm typecheck && pnpm build

# Full
make test

# Contract (once the contract pipeline exists)
cd packages/api-contract && dart test test/contracts   # or a script wrapper

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
| contract | fixture deserialization | dart test / jest (once pipeline exists) |
| mobile | widget/unit | flutter test (when mobile lands) |

## CI mapping

- server.yml: go vet + go test + go build (apps/gooseforum/app/**, main.go, go.mod, go.sum); the
  PostgreSQL integration tests in `app/bundles/connect/sqlconnect` are gated by `TEST_PG_DSN` and
  skip when unset (CI stays green without a PG service)
- web.yml: pnpm typecheck + build (apps/gooseforum/resource/**)
- contract.yml: openapi validation + no-diff generation + fixture (apps/gooseforum/app/**, packages/**)

## Smoke checklist

```bash
curl http://localhost:5234/            # homepage HTML (three-mode rendering, GoHTML)
curl http://localhost:5234/api/...     # JSON API (per upstream routes)
# Frontend dev: http://localhost:3010
```
