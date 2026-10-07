# Development Entry

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-03

Any code, contract, migration, CI, or documentation change starts here. `AGENTS.md` holds repository hard
constraints; this directory holds the executable process. Do not copy development steps from historical
PRs or chat messages.

## Before you start

1. Read the root `AGENTS.md`, the [docs index](../README.md), and the product/architecture/operations
   specs relevant to the request.
2. Determine whether the request is read-only analysis, a change, or explicitly authorizes
   commit/push/open PR.
3. Check branch, worktree, and uncommitted content; never overwrite or commit others' changes.
4. Create a feature/fix/docs branch from `origin/dev`.
5. Use the [Issue/PR standard](pull-requests.md) to establish current-state evidence, relevant external
   research, affected users/stories, scope and observable acceptance criteria. Scale detail to the change;
   resolve material unknowns before dependent implementation.
6. Write the change impact: backend, web, mobile, contract, migration, auth/PII, search, deploy, docs.

The repository-level `$yourtj-development` skill lives in `.agents/skills/yourtj-development` and unifies
this process, verification, and delivery.

## Standard workflow

```text
problem evidence, research and affected users
  -> user stories, scope and acceptance criteria
  -> impact and risk boundary
  -> failing regression first (bug fixes)
  -> contract/migration (if needed)
  -> service/models/http implementation
  -> acceptance evidence and focused checks
  -> documentation impact and diff review
  -> commit/push/PR (only when explicitly authorized)
  -> CI + preview verification
```

## Detailed guides

- [Local environment](local-development.md)
- [Standalone status app](../../apps/status/README.md)
- [Testing strategy & commands](testing.md)
- [Mobile performance](mobile-performance.md)
- [Issues, requirements, review & pull requests](pull-requests.md)
- [Licensing](licensing.md)
- [Release workflow and agent entrypoints](../operations/releases.md)
- [Project board workflow](project-board.md)
- [Documentation governance](documentation.md)
- [Contracts, data & derived projections](../architecture/contracts-and-data.md)
- [Coding conventions](coding-conventions.md)
- [Go dependency vulnerability scanning](dependency-scanning.md)

## Definition of done

- The problem, affected users and scope are clear; applicable research conclusions are sourced, and
  each acceptance criterion maps to actual evidence. Failed or unverified criteria remain visible.
- No unexplained gaps in product semantics, permissions, failure/recovery, privacy, or retention.
- Code lives in the right layer (service/models/http); for OpenAPI-covered operations, the OpenAPI
  definition and generated types match the implementation; migrations match the deployed schema.
- The numeric-ID constraint (uint64 sub) is not bypassed; the built-in OIDC Provider always issues
  numeric `sub` = users.id.
- Docs status words are updated; contract changes ship generated output and fixtures.
- The commands actually run and their results are reported; a local subset is not CI passing.
- The PR retains summary, behavior change, verification, docs/contract impact and known gaps, with
  product context, user stories, acceptance results and supporting material added per the Issue/PR standard.
