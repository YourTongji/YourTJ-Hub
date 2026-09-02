# yourtj-hub Docs Center

> Doc type: documentation index
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-02

This is the single entry point for yourtj-hub product, architecture, development, and operations specs.
Docs describe only the currently supported model; stale phase plans, PR delivery checklists, and
duplicate API/DDL snapshots are not kept long-term in the current doc tree (git history owns archival).

## How to decide the fact source

Different questions are owned by different sources; no single "goal vs. current state" ordering:

| Question | Authoritative source |
|---|---|
| How the product should work | `docs/product/` |
| Security, privacy, compliance hard constraints | `AGENTS.md` and `docs/security/` (until then, AGENTS.md) |
| HTTP request/response structure | `apps/gooseforum/app/http/controllers`; for OpenAPI-covered operations, `packages/api-contract/openapi.yaml` |
| Deployed database structure | Migrations under `apps/gooseforum/app/migration` |
| Current code behavior | Source, automated tests, and the deployed version |
| Development, test, and PR process | `docs/development/` |
| Deployment and incident handling | `docs/operations/` |

When these sources disagree, do not pick a convenient version and keep going. Treat the difference as a
defect and fix the contract, implementation, tests, and related docs in the same PR, or record it
explicitly as `Partial`.

## Implementation status words

Product docs use only these four implementation states:

- `Current`: the user-reachable required chain, backend constraints, and corresponding verification all exist.
- `Partial`: only some layers are done; must state which of Web, API, schema, worker, ops process, or tests is missing.
- `Planned`: the target business rules are formed but not delivered; must not be claimed usable in UI or marketing.
- `Decision needed`: data model, permissions, or product direction still need an owner decision; must not be
  implemented by default before the decision.

Status words apply to concrete, verifiable behavior, not to whole domains. A domain can have a `Current`
backend base and a `Partial` end-to-end product chain; the former must not be used to claim the whole
feature is complete.

Docs themselves use `Active`, `Draft`, `Deprecated` for lifecycle. That is separate from implementation
status. Do not use PR-relative "shipped this / later" labels as long-term status.

## Index

### Product

- [Vision & principles](product/vision-and-principles.md)
- [Current state & gaps](product/current-state.md)
- [Wiki authoring](product/wiki-authoring.md)
- [Identity, login & account lifecycle](product/identity-and-access.md)
- [Points & cross-platform settlement](product/credit-and-escrow.md)

### Architecture

- [System overview & domain boundaries](architecture/system-overview.md)
- [Contracts, data & derived projections](architecture/contracts-and-data.md)

### Development

- [Development entry](development/README.md)
- [Local environment](development/local-development.md)
- [Testing strategy & commands](development/testing.md)
- [Branches, commits & pull requests](development/pull-requests.md)
- [Documentation governance](development/documentation.md)

### Operations

- [Deployment & release](operations/deployment.md)
- [Object storage](operations/object-storage.md)

### Decision records

- Architecture decision records (ADR) live in the project note (yourtj-hub ADR note), **not in git**;
  new decisions append a number, append-only, history never rewritten.
