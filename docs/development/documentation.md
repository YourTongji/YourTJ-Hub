# Documentation Governance

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-06

## Docs are code's neighbors

- Any PR that changes product behavior, contracts, schema, security boundaries, or deployment must
  update the affected docs in the same PR.
- Docs are not "write later"; if they are not updated in the same PR, it is a defect.
- Code is authoritative. When code changes a contract, update the owning document in the same change;
  do not keep parallel narratives for historical versions.

## Docs describe only the current model

- Docs describe the **currently supported behavior model** (how the product works, system invariants,
  commands and verification) — no timelines, phase plans, milestones, or PR delivery checklists.
- Planned capabilities are marked with implementation status words (below), not "phase N".
- Historical narrative (retired schemas, old processes, drafts) does not belong in the current doc
  tree; git history owns archival.

## Status words (mandatory)

- Implementation status: `Current` / `Partial` / `Planned` / `Decision needed`, applied to concrete
  verifiable behavior.
- Doc lifecycle: `Active` / `Draft` / `Deprecated` — separate from implementation status.
- No PR-relative "shipped this / later" labels as long-term status.

## Fact sources (see docs/README.md)

| Question | Authoritative source |
|---|---|
| How the product should work | docs/product/ |
| Security/privacy/compliance | AGENTS.md, docs/security/ (until then AGENTS.md) |
| HTTP structure | apps/gooseforum/app/http/controllers, packages/api-contract/openapi.yaml |
| DB structure | apps/gooseforum/app/migration/ |
| Current behavior | source, tests, deployed version |

When sources disagree, treat it as a defect and fix it in the same PR, or record it explicitly as `Partial`.

## Doc change process

1. Confirm facts before writing (read source/tests/contracts; never from memory).
2. Update affected product/architecture/development/operations docs and status words.
3. Big decisions go into the project note's ADR record (yourtj-hub ADR note), append-only numbering,
   history never rewritten.
4. Delete stale content instead of keeping "deprecated but useful" copies; git history owns archival.
