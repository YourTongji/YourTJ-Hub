# Documentation Governance

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-03

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

## Requirement and delivery records

The [Issue/PR standard](pull-requests.md) owns the reusable process for problem evidence, external
research, target users, user stories and acceptance criteria. Task-specific requirements and verification
results belong in the relevant Issue/PR; they do not replace the maintained product, architecture or
operations specification. A PR retains its engineering delivery sections alongside product context.
Development guides may describe reusable readiness and review criteria; the current-model rule excludes
one-off delivery plans and execution diaries, not the process itself.

Keep raw research artifacts in untracked `research/`. Preserve concise conclusions and accessible source
links in Issue/PR descriptions; when external evidence materially informs a durable choice, cite it in
the owning MADR or stable implementation location. Mark assumptions and access limitations explicitly.

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
3. Durable choices with real alternatives get a MADR record in
   [docs/decisions/](../decisions/) (append-only numbering, supersede instead of rewrite;
   `node scripts/verify-decisions.mjs` enforces format).
4. Delete stale content instead of keeping "deprecated but useful" copies; git history owns archival.
5. Any new feature PR must include documentation changes: user-visible features update the docs center
   and status words; purely internal changes at least update the relevant README or code comments.

## Link verification

`node scripts/run-gates.mjs` includes relative Markdown link and heading checks for governance docs,
all root/nested Markdown READMEs, and `apps/gooseforum/docs`. Discovery uses Git's tracked and new
non-ignored files, so local dependency/SDK caches are excluded. Deleted files and symlinked documents
are not scanned. This gate does not fetch external URLs or verify prose against implementation.

Run `node --test scripts/test-doc-links.mjs` when changing the link checker. The documentation CI
workflow runs these regression tests and the governance gates for every PR and every push to `dev`
or `main`, including changes that only rename or delete a linked non-Markdown target.
