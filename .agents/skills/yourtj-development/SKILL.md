---
name: yourtj-development
description: Develop, fix, refactor, review, test, document, or publish changes in the yourtj-hub repository. Use for any task involving the GooseForum forum (apps/gooseforum Go backend + Vue resource), Flutter mobile (apps/mobile), OpenAPI contract (packages/api-contract), services/deployment, repository documentation, commits, pushes, or pull requests so layer boundaries, verification, documentation impact, and publication authority are handled consistently.
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

Run `git status --short --branch`, inspect worktrees, and preserve existing changes. Never work directly on
`main`. Create a branch from current `origin/main`; use `feat/<topic>` / `fix/<topic>` / `docs/<topic>`.
Prefer a worktree when the current checkout is dirty. Never discard or include unrelated work.

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
- database migration/backfill/concurrency (app/migration, SQLite/MySQL, PostgreSQL pending);
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

Hard constraints that must never be violated:

- **Deployment stays a single binary** (resource/static/dist + templates go:embed). No nginx/CDN split.
- **User IDs must be numeric (uint64)** once Casdoor auth lands — credit's `GetID()` only parses
  numeric sub; UUID collapses all users to 0. Enforce and test in the auth layer.
- **Casdoor is the only identity source** once integrated; the forum JWT is a session credential.
- **Contract changes ship with generated output and fixtures in the same PR** (once the pipeline exists).
- config.toml contains signingKey — never commit it (gitignored).

## 5. Verify proportionally

Read and follow [`docs/development/testing.md`](../../../docs/development/testing.md). Always run:

```bash
git diff --check
```

Then run the exact gates for changed paths:

- Backend: `cd apps/gooseforum && go vet ./... && go test ./...` (use `GOPROXY=https://goproxy.cn,direct`
  if module fetch times out).
- Web: `cd apps/gooseforum/resource && pnpm typecheck && pnpm build` (output lands in static/dist).
- Full build: `make build` then smoke-test the binary (`./bin/yourtj-hub serve`, then curl the
  configured port, default 5234).
- Contract (once the pipeline exists): regenerate openapi.yaml + TS/Dart, inspect diff, run fixture
  contract tests.
- Auth/PII/governance/credit/search: include documented negative, replay, privacy, failure, and
  reconciliation cases.

Report commands that failed, skipped, or were not run. Never infer CI success from a local subset.

## 6. Synchronize documentation

Follow [`docs/development/documentation.md`](../../../docs/development/documentation.md). Update the owning
product/architecture/development/operations documents and status words (`Current`/`Partial`/`Planned`/
`Decision needed`) in the same PR. Documents describe the current supported model only — no timeline or
milestones. Big decisions are recorded in the project ADR note (not in git), append-only numbering.

## 7. Deliver

Commit only when the user explicitly asks. Use conventional types
(`feat:` / `fix:` / `docs:` / `refactor:` / `chore:` / `test:`).

Push / open PR / deploy only with explicit authorization. Preserve unrelated dirty files. Summarize what
changed, what was verified (commands + results), what remains uncertain, and which status words were updated.
