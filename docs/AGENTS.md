# AGENTS.md — Documentation standard

The authoritative documentation governance for this repository is
[docs/development/documentation.md](development/documentation.md) (docs are code's neighbors, current-model-only
prose, the four implementation status words, fact sources, and the doc change process). The docs-center
entry point with the full index is [docs/README.md](README.md). Do not restate their rules here.

This file only carries the repo-seed structure rules that complement that governance:

- **One home per fact.** A document's subject and tree position fix its scope; describe your own subject
  at appropriate detail, direct children only by purpose and responsibility, and link to the owning
  descendant for lower-level detail.
- **Tutorial or reference.** Classify every document as a tutorial (ordered path to an outcome) or a
  reference (lookup scope, current behavior, no teaching sequence).
- **Machine-checkable links.** Cross-reference with relative Markdown links that resolve; `node
  scripts/verify-doc-links.mjs` enforces this for `AGENTS.md`, `CLAUDE.md`, `docs/**`, and
  `CONTRIBUTING.md`, and `node scripts/verify-placeholders.mjs` fails on any leftover fill-in token.
- **Governed surfaces.** `docs/specs/` remains a registered external pointer (see
  `.repo-seed/manifest.json`); `docs/decisions/` is the in-git MADR decision log enforced by
  `node scripts/verify-decisions.mjs`; `docs/postmortems/` follows the incident format in its README.
  Semantic review and decision procedures use the `.agents/skills/repo-*` skills.
