---
name: yourtj-development
description: Prepare requirements and implement, fix, test, document, or deliver changes in yourtj-hub. Connect product evidence, affected users and acceptance criteria to implementation and PR evidence. Also establishes repository context for read-only review and diagnosis; use the review skills for review findings and yourtj-pre-push-checks before publication checks.
---

# yourtj Development

Use this workflow for repository work from initial scope through verified handoff. Keep product behavior,
wire contracts, database shape, implementation, tests, and documentation synchronized.

Release preparation/status uses [yourtj-release](../yourtj-release/SKILL.md); platform prose uses
[yourtj-release-notes](../yourtj-release-notes/SKILL.md), and distribution failures use
[yourtj-release-recovery](../yourtj-release-recovery/SKILL.md). These share the repository CLI and do
not grant an agent authority to replace the required human release review.

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

### Requirements before implementation

Read the [Issue and PR standard](../../../docs/development/pull-requests.md). It owns the required
information and proportionality rules; templates are collection aids, not a separate policy.

- Establish the current behavior and problem with code/docs, reproduction or observed feedback.
  Distinguish facts, assumptions and unanswered questions. If the goal is unclear, clarify it before
  dependent implementation; if a simpler complete approach exists, explain the trade-off.
- For feature/interaction work, inspect relevant external products or standards, cite sources and
  observation dates, and explain applicability to this repository. Reuse verified existing research.
  Never invent findings or treat another product's behavior as a requirement by itself.
- Identify primary and other affected users, including passive recipients and privileged roles where
  relevant. Express their goals as `US-*` stories; record scope, exclusions and dependencies.
- Define observable `AC-*` criteria tied to stories or bug reproductions before implementation.
  Include permission, boundary and recovery behavior where affected. Small fixes and maintenance
  use the abbreviated form in the standard; no artificial personas or research quota.
- Reuse sufficient user instructions and existing decisions. Ask only about unresolved choices that
  materially affect the outcome or risk boundary; continue independent work. Do not manufacture an
  extra approval step, create a remote Issue without authorization, or require an Issue for every edit.

Keep the requirement summary in the working handoff or an already authorized Issue/PR. A request to
implement does not authorize remote publication. Raw research belongs in untracked `research/`;
durable conclusions and source links belong in the appropriate Issue/PR, docs or MADR.

## 3. Build an impact matrix

Before editing, state whether the change affects:

- forum backend layer (bundles / models / service / http controllers) and cross-layer access;
- forum frontend (`resource/`, generated types, GoHTML templates);
- Flutter mobile (`apps/mobile`, shared packages, platform behavior and mirrored contracts);
- HTTP/OpenAPI compatibility (packages/api-contract);
- database migration/backfill/concurrency (app/migration, SQLite dev / PostgreSQL deployment default);
- auth (GitHub OAuth and built-in OIDC Provider), JWT sessions, PII, privacy, retention, or audit;
- credit compliance / signatures / replay (only when credit work is in scope);
- search (Meilisearch), cache, counters, notifications, or background jobs;
- deployment/config/provider secrets (config.toml);
- product, architecture, development, operations, and decision documents.

For unresolved credit compliance, PII lifecycle, access or data-semantics choices, follow
[Ready for implementation](../../../docs/development/pull-requests.md#ready-for-implementation).
Pause dependent implementation only; existing explicit decisions remain valid authorization.

## 4. Implement in dependency order

Use this order where applicable:

1. product semantics and acceptance criteria;
2. for a bug fix, write and run the smallest failing regression before changing behavior (mechanical
   and docs-only exemptions follow `AGENTS.md`);
3. HTTP/OpenAPI compatibility, append-only migrations and recovery design where affected;
4. owner-layer implementation: service → models → http controllers, then relevant Web/mobile surfaces;
5. synchronized OpenAPI definitions, generated TypeScript, fixtures, route coverage and required
   Dart/Web mirrors in the same change; the contract pipeline already exists;
6. focused behavior tests and `AC-*` evidence, then broader checks only where the impact warrants them;
7. owning documentation and operational runbooks.

Repository hard constraints live in `AGENTS.md` §3 — single binary, numeric user IDs, contract changes ship
generated output and fixtures in the same PR, docs status words, and new-feature documentation. Read them
before implementing and never violate them.

## 5. Verify proportionally

Read and follow [`docs/development/testing.md`](../../../docs/development/testing.md). Run only the checks
that cover the changed surface; CI owns the full repository-wide gate matrix. The lefthook pre-push hook
already runs `go vet` + `golangci-lint` + `pnpm typecheck` + the web i18n gate `pnpm check` (install with
`make hooks`) — do not repeat them manually for the same push.

Always run:

```bash
git diff --check
```

Select evidence by changed surface:

| Surface | Evidence |
|---|---|
| Backend (bundles/models/service/controllers) | `cd apps/gooseforum && go vet ./... && go test ./...` — focus on affected packages with `-run` when practical |
| Model/migration change (mandatory PG gate) | run the PostgreSQL migration tests (docker `postgres:16-alpine` + `YOURTJ_TEST_PG_URL=... go test ./app/migration/ -run 'PostgreSQL|Postgres' -v`; command in testing.md) |
| Frontend (`resource/**`) | `cd apps/gooseforum/resource && pnpm typecheck` + affected component tests |
| Contract (`packages/api-contract/**`) | `make contract-check` (regenerates and requires committed TS output) |
| Docs, skills, templates | `node scripts/run-gates.mjs`, semantic/link review, skill frontmatter or form syntax validation when changed |
| Auth/PII/governance/credit/search | documented negative, replay, privacy, failure, and reconciliation cases |
| Cross-cutting, CI diagnosis, or explicit user request | full `make test` / `make build` |

Report commands that failed, skipped, or were not run. Never infer CI success from a local subset.
Map each applicable `AC-*` to observed evidence and an honest result. A test command alone does not
prove the user outcome. For UI changes, inspect the affected user flow and visible states on the relevant
platforms; follow testing.md for visual evidence rather than adding screenshot baselines by default.

## 6. Synchronize documentation

Follow [`docs/development/documentation.md`](../../../docs/development/documentation.md). Any new feature PR
must include documentation changes: user-visible features update the docs center and status words
(`Current`/`Partial`/`Planned`/`Decision needed`); purely internal changes at least update the relevant
README or code comments. Documents describe the current supported model only — no timeline or milestones.
- Durable decisions with real alternatives are recorded as MADR records in
  [`docs/decisions/`](../../../docs/decisions/): append-only numbering, supersede instead of
  rewrite, enforced by `node scripts/verify-decisions.mjs`.

## 7. Deliver

Commit only when the user explicitly asks. Use conventional types
(`feat:` / `fix:` / `docs:` / `refactor:` / `chore:` / `test:`).

Before handoff, use [repo-review](../repo-review/SKILL.md) and
[yourtj-code-review](../yourtj-code-review/SKILL.md) to check requirement completeness and the affected
engineering surfaces. Before push or readiness claims, use [yourtj-pre-push-checks](../yourtj-pre-push-checks/SKILL.md).

Push / open PR / deploy only with explicit authorization. Preserve unrelated dirty files. Prepare PR
content with the [repository template](../../../.github/pull_request_template.md): retain Summary,
Behavior change, Verification, Docs & contract impact, and Known gaps; add the product context,
affected users/stories, acceptance results and supplementary links. Rewrite the description around the
final implementation when scope changes. Report actual evidence, uncertainty and documentation impact.
