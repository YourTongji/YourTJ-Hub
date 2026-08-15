---
name: yourtj-development
description: Develop, fix, refactor, review, test, document, or publish changes in the yourtj-hub repository. Use for any task involving the GooseForum forum (apps/gooseforum Go backend + Vue resource), Flutter mobile (apps/mobile), OpenAPI contract (packages/api-contract), services/deployment, repository documentation, commits, pushes, or pull requests so layer boundaries, verification, documentation impact, and publication authority are handled consistently. Read-only analysis, review, and diagnosis also route here; use $yourtj-pre-push-checks before pushing or claiming checks pass.
---

# yourtj Development

Use this workflow for repository work from initial scope through verified handoff. Keep product behavior,
wire contracts, database shape, implementation, tests, and documentation synchronized.

## 1. Establish authority and workspace

Classify the request before changing state:

- **Read-only:** analysis, review, diagnosis, or status. Inspect and report; do not implement or publish.
- **Change:** fix, build, update, or create. Implement and verify only the requested scope.
- **Publish:** commit, push, open/update a PR, deploy, or mutate an external system. Require explicit
  authorization for the requested publication action; change authorization alone is insufficient.

Follow the branch, worktree, and commit discipline in `AGENTS.md` §5 and
[`docs/development/pull-requests.md`](../../../docs/development/pull-requests.md); do not duplicate it here.
Run `git status --short --branch` first, inspect worktrees, and preserve existing changes. Prefer a worktree
when the current checkout is dirty. Never discard or include unrelated work.

## 2. Read the governing sources

Read completely before implementation:

1. repository `AGENTS.md`;
2. [`docs/README.md`](../../../docs/README.md) — fact-source table and status words;
3. [`docs/development/README.md`](../../../docs/development/README.md);
4. the directly affected product, architecture, and operations documents;
5. `apps/gooseforum` source (app/ + resource/), `packages/api-contract/openapi.yaml`,
   relevant migrations, and tests as needed.

Do not use deleted historical plans, old PR descriptions, or chat messages as a second source of truth.

## 3. Build an impact matrix

Before editing, state whether the change affects:

- forum backend layer (bundles / models / service / http controllers) and cross-layer access;
- forum frontend (`resource/`, generated types, GoHTML templates);
- HTTP/OpenAPI compatibility (packages/api-contract);
- database migration/backfill/concurrency (app/migration, SQLite dev / PostgreSQL deployment default);
- auth (GitHub OAuth current; Casdoor OIDC planned), JWT sessions, PII, privacy, retention, or audit;
- credit compliance / signatures / replay (phase 2, only when credit work is in scope);
- search (Meilisearch), cache, counters, notifications, or background jobs;
- deployment/config/provider secrets (config.toml);
- product, architecture, development, operations, and decision documents.

Stop and escalate if the request crosses the credit compliance line, needs new PII without lifecycle
answers, or requires a product decision that changes access or data semantics.

## 4. Implement in dependency order

Use this order where applicable:

1. product semantics and acceptance criteria;
2. OpenAPI contract (once the contract pipeline exists; otherwise keep `@gooseforum/client` in sync);
3. append-only migration and compatibility plan;
4. owner-layer implementation: service → models → http controllers;
5. generated Web types and user/admin surfaces;
6. focused tests, then scope-wide verification;
7. documentation and operational runbooks.

Repository hard constraints live in `AGENTS.md` §3 — single binary, numeric user IDs, contract changes ship
generated output and fixtures in the same PR, docs status words, and new-feature documentation. Read them
before implementing and never violate them.

## 5. Verify proportionally

Read and follow [`docs/development/testing.md`](../../../docs/development/testing.md). Run only the checks
that cover the changed surface; CI owns the full repository-wide gate matrix. The lefthook pre-push hook
already runs `go vet` + `golangci-lint` + `pnpm typecheck` (install with `make hooks`) — do not repeat them
manually for the same push.

Always run:

```bash
git diff --check
```

Select evidence by changed surface:

| Surface | Evidence |
|---|---|
| Backend (bundles/models/service/controllers) | `cd apps/gooseforum && go vet ./... && go test ./...` — focus on affected packages with `-run` when practical |
| Model/migration change (mandatory PG gate) | run the PostgreSQL migration tests (docker `postgres:16-alpine` + `YOURTJ_TEST_PG_URL=... go test ./app/migration/ -run TestSchemaMigratesOnPostgreSQL -v`; command in testing.md) |
| Frontend (`resource/**`) | `cd apps/gooseforum/resource && pnpm typecheck` + affected component tests |
| Contract (`packages/api-contract/**`) | `make contract-check` (regenerates and requires committed TS output) |
| Docs | `git diff --check` + verify links and status words |
| Auth/PII/governance/credit/search | documented negative, replay, privacy, failure, and reconciliation cases |
| Cross-cutting, CI diagnosis, or explicit user request | full `make test` / `make build` |

Report commands that failed, skipped, or were not run. Never infer CI success from a local subset.

## 6. Synchronize documentation

Follow [`docs/development/documentation.md`](../../../docs/development/documentation.md). Any new feature PR
must include documentation changes: user-visible features update the docs center and status words
(`Current`/`Partial`/`Planned`/`Decision needed`); purely internal changes at least update the relevant
README or code comments. Documents describe the current supported model only — no timeline or milestones.
Big decisions are recorded in the project ADR note (not in git), append-only numbering.

## 7. Deliver

Commit only when the user explicitly asks. Use conventional types
(`feat:` / `fix:` / `docs:` / `refactor:` / `chore:` / `test:`).

Push / open PR / deploy only with explicit authorization. Preserve unrelated dirty files. Summarize what
changed, what was verified (commands + results), what remains uncertain, and which status words were updated.
