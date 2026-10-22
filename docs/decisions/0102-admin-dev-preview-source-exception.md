# Administrator exception for a fixed dev preview source

## Status

Accepted
Class: process

## Context and Problem Statement

A maintainer can explicitly authorize an emergency dev preview after a bypass merge, while the
normal signing gate requires a different eligible human to approve the exact source PR head.
Branch bypass is not evidence of that source review. The release controller needs a deliberate
exception with an accountable human and an exact scope, without treating it as formal release approval.

## Decision Drivers

- Permit explicitly authorized internal dev testing when a second reviewer is unavailable.
- Preserve the selected commit across queue delays, source advances and partial reruns.
- Keep sealed audit attribution consistent with the person who can execute the exception.
- Retain formal production review, QA, ownership, CI and internal-only distribution boundaries.

## Considered Options

- Require a second human source approval for every preview, without an emergency exception.
- Treat any admin actor or bypass merge as unrestricted standing preview authority.
- Require an explicit admin reason and source SHA, sealed for one cycle and workflow run.

## Decision Outcome

Use an explicit, audited exception for dev previews only. Both the workflow initiator and triggering
actor must be repository admins. The request names its full source SHA even when using an existing
cycle. If the cycle advances while the request is queued, the request fails rather than adopting
its new source. Sealed authority records source, cycle, run, both actors, time and reason.

Identity allocation and signing validate the exact binding and current admin permissions. A partial
rerun must use the recorded triggering actor. A full rerun preserves the original sealed record;
it cannot rebind the exception after source advancement. Another actor or source needs a new
explicit dispatch, subject to the ordinary cycle ownership rules. Package rebuilds remain supported
for the fixed source with the normal automatically allocated preview identities.

The exception supplies neither QA nor permission for production-configuration rehearsals, formal
Release, hotfix backflow or branch merging. Ordinary previews retain exact human source review.
TestFlight previews still use the selected internal group and do not submit external beta review.
This supplements the preview admission boundary; the [reviewed production pipeline](0055-reviewed-release-pipeline.md)
remains unchanged.

## Pros and Cons of the Options

- Always requiring another reviewer has the simplest authority model and independent review, but
  cannot fulfill an explicit emergency preview request while that reviewer is unavailable.
- Unrestricted admin bypass minimizes steps, but leaves no per-source intent and makes stale queued
  requests, actor changes and retries difficult to distinguish from new authorization.
- A sealed one-run exception records concrete scope and rejects drift. It gives up independent
  source review for that dev preview and adds a narrow recovery protocol: a changed actor/source
  needs a new dispatch rather than an implicit retry. Admin trust is explicit, not inferred from merge status.

## Links

- [Release-cycle runbook](../operations/release-cycles.md)
- [Initial implementation and maintainer-requested scope](https://github.com/YourTongji/YourTJ-Hub/pull/1254)
- [Fixed-source release strategy](https://github.com/YourTongji/YourTJ-Hub/issues/1240)
- [GitHub workflow reruns and retained event identity](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/re-run-workflows-and-jobs)
- [GitHub initiating and triggering actor variables](https://docs.github.com/en/actions/reference/workflows-and-actions/variables)
