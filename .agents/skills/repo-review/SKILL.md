---
name: repo-review
description: Use when reviewing a pull request or a change in this repository — orients the reviewer to this repository's standards (AGENTS.md conventions, decision log, gates) and the review-specific checks that code alone cannot show
---

# Reviewing a change in this repository

Read the diff, the owning docs, the decision log, and enough surrounding code to understand the design before judging it. **Blocking requirements** are hard: a violation blocks the change. **Manual checks** rank remaining risk by this project's real failure modes — apply the ones the change touches, not every one, to every change.

## Sources of truth

- [AGENTS.md](../../../AGENTS.md): standing repository rules.
- [docs/AGENTS.md](../../../docs/AGENTS.md): documentation placement and prose discipline.
- [docs/decisions/](../../../docs/decisions/): durable design rationale. Disagreement with a decision record is a design discussion, not an automatic veto.
- [docs/specs/](../../../docs/specs/): risk-triggered change contracts.
- [docs/development/testing.md](../../../docs/development/testing.md): risk-to-layer selection, test topology, evidence rules, and maintenance budget.
- [docs/architecture/system-overview.md](../../../docs/architecture/system-overview.md): the module map and seams.

## Blocking requirements

### Universal (applies to any repository)

1. **New prose receives semantic review.** Critically review every added or changed Markdown passage, JSDoc, comment, prompt, description, diagnostic, and visible string. Verify required coverage, accuracy, placement, and editorial quality against the owning code or behavior; automated checks do not establish those properties.
2. **Docs match the code.** Config, defaults, errors, wire fields, events, and public behavior update the owning docs and JSDoc in the same diff.
3. **Contracts and decisions use the right artifact.** A risk-boundary change has an Approved spec; a durable choice with real alternatives has a decision record. Do not demand an ADR as a change log.
4. **Tests provide minimum sufficient behavior evidence.** Name the meaningful regression each changed test prevents and place its primary evidence at the lowest sufficiently real boundary. A fix links an existing deterministic reproduction or adds a regression test that fails before the fix and passes afterward; redundant coverage or implementation-only assertions do not satisfy this requirement.
5. **External-source provenance is retained.** If implementation is materially derived from a paper, article, community post, benchmark, research report, or copied/adapted code, cite it at the closest stable code location or link that location to a decision record whose `## Links` cites the source. A pull request, issue, prompt, or chat-only citation does not count; copied or adapted material also preserves applicable license and NOTICE requirements.

### Project-specific (this repository defers to its resident review skill)

The project-specific blocking requirements and manual checks are owned by
[`.agents/skills/yourtj-code-review/SKILL.md`](../yourtj-code-review/SKILL.md) (contract impact, PG
migration safety, security boundaries, performance, maintainability, test coverage — with the
`blocker / should / nit` severity model). Read it for every review in this repository and apply its
layers to the surfaces the change touches. The universal sections above apply on top; the resident
skill wins when the two disagree.

## Manual checks

### Project-specific

Covered by [`yourtj-code-review`](../yourtj-code-review/SKILL.md) (see above); the universal
fallbacks below apply where it is silent.

### Universal fallbacks (apply where the project has no specific rule)

- **Intent and interface contracts:** trace both sides of every changed interface. Confirm the implementation matches the change and any decision record, including errors, cancellation, ownership, and disposal.
- **Lifecycle and concurrency:** for async setup, callbacks, processes, or teardown, check races before publication, cancellation during awaits, independent error reporting, and complete cleanup.
- **Human readability:** read changed code from its entry point downward. The primary path stays understandable without opening every helper. Challenge abstractions that only forward or rename, mixed abstraction levels, misleading or generic names, new synonyms for existing concepts, and logic outside its owning boundary. Block when these make behavior unreliable to infer, hide effects or cost, or violate established vocabulary or boundaries; otherwise report subjective polish as advisory. Line counts, parameter counts, duplication, and complexity scores are signals, not verdicts.
- **Scope and necessity:** map each abstraction, option, defensive copy, and compatibility path to its current contract and consumer. Challenge unrelated features and speculative generality.
- **Bounds cover the final operation:** probe tiny and exact limits, oversized chunks, and multibyte text for byte limits.
- **Test value and boundary:** identify the contract each changed test owns, challenge duplicate evidence, and use the real shipped entry path whenever packaging, configuration, process setup, or wiring is part of the risk.
