# User-visible change declarations and independent reconciliation

## Status

Proposed
Class: process

## Context and Problem Statement

Behavior tests can preserve action values while a component migration silently removes an icon or
other visible feature. In PR #1170, Home's display-options migration replaced icon-bearing ListTiles
with text-only menu items. A large diff makes that loss hard to distinguish from an intentional choice.
The visual scope must be reviewable without treating each implementation file as a separate design.

## Decision Drivers

- Make every user-perceptible delta traceable to declared scope, including removed or weakened features.
- Keep review independent of the author's inventory to avoid confirming the same omission twice.
- Keep mechanical reskins proportional without exempting them from feature-preservation checks.
- Preserve behavior tests and platform visual acceptance as separate evidence.

## Considered Options

- Rely only on existing behavior tests, visual acceptance and ordinary code review.
- Require per-surface declarations and independent review reconciliation.
- Also require a deterministic PR-body/UI classifier gate.

## Decision Outcome

Use per-surface declarations and independent reconciliation, with the normative rule in the
[Issue/PR standard](../development/pull-requests.md#visual-change-declaration). Authors identify old → new
features, removals and scope provenance. Reviewers independently inventory actual deltas before matching
them to declarations; unmatched changes are process blockers resolved by declaration correction or
restoration, not by imposing a reviewer's design preference. Mechanical reskins may use a complete
call-site inventory and independent samples; one differing feature invalidates the entire batch claim.

Enforcement is an author/reviewer process, not a new CI gate. Existing governance gates continue to
check repository artifacts, not PR-body semantics. The repository skills and PR template route to the
same rule. Oryn Mini's runtime prompt is owned in its separate repository; changing these repository
skills alone does not establish that the external runtime executes reconciliation.

## Pros and Cons of the Options

- Existing tests and visual acceptance remain necessary, but do not expose whether a missing feature
  was in the approved scope; they cannot replace the declaration audit trail.
- Declarations plus independent review expose unexpected removals and permit a bounded mechanical
  exception. They require reviewer judgment and cannot guarantee complete visual coverage by themselves.
- A deterministic PR-body gate could detect an absent heading, but cannot establish semantic completeness
  or reliably infer presentation impact from filenames. It adds machinery without proving reconciliation.

## Links

- [Issue #1238](https://github.com/YourTongji/YourTJ-Hub/issues/1238) — scope and declaration proposal.
- [PR #1170](https://github.com/YourTongji/YourTJ-Hub/pull/1170/files) — Home migration evidence.
- [Issue #1241](https://github.com/YourTongji/YourTJ-Hub/issues/1241) — separately tracked menu leading capability.
- [0095](0095-adaptive-ios-interaction-surfaces.md) — adaptive-surface migration scope; later surface changes remain owned by its linked decisions.
- [0047](0047-mobile-visual-acceptance.md) — independent mobile visual acceptance.
- [Oryn integration](../operations/oryn.md) — separately pinned external runtime.
