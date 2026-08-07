# Development Entry

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-06

Any code, contract, migration, CI, or documentation change starts here. `AGENTS.md` holds repository hard
constraints; this directory holds the executable process. Do not copy development steps from historical
PRs or chat messages.

## Before you start

1. Read the root `AGENTS.md`, the [docs index](../README.md), and the product/architecture/operations
   specs relevant to the request.
2. Determine whether the request is read-only analysis, a change, or explicitly authorizes
   commit/push/open PR.
3. Check branch, worktree, and uncommitted content; never overwrite or commit others' changes.
4. Create a feature/fix/docs branch from `origin/main`.
5. Write the change impact: backend, web, contract, migration, auth/PII, search, deploy, docs.

The repository-level `$yourtj-development` skill lives in `.agents/skills/yourtj-development` and unifies
this process, verification, and delivery.

## Standard workflow

```text
requirements and product semantics
  -> impact and risk boundary
  -> contract/migration (if needed)
  -> service/models/http implementation
  -> focused tests
  -> scope-wide CI-parity checks
  -> documentation impact and diff review
  -> commit/push/PR (only when explicitly authorized)
  -> CI + preview verification
```

## Detailed guides

- [Local environment](local-development.md)
- [Testing strategy & commands](testing.md)
- [Branches, commits & pull requests](pull-requests.md)
- [Documentation governance](documentation.md)
- [Contracts, data & derived projections](../architecture/contracts-and-data.md)

## Definition of done

- No unexplained gaps in product semantics, permissions, failure/recovery, privacy, or retention.
- Code lives in the right layer (service/models/http); for OpenAPI-covered operations, the OpenAPI
  definition and generated types match the implementation; migrations match the deployed schema.
- The numeric-ID constraint (uint64 sub) is not bypassed; auth still has Casdoor as the only identity
  source once integrated.
- Docs status words are updated; contract changes ship generated output and fixtures.
- The commands actually run and their results are reported; a local subset is not CI passing.
