# Keep Oryn work across normal collaboration updates

## Status

Accepted
Class: process

## Context and Problem Statement

PR #609 completed model review but CI completion and a review bot's status update invalidated its
context digest before publication. This discarded useful review without any source change. Maintainers
require interruption only for explicit stop commands, changes to the original command or its authority,
and changes to the reviewed title/body or code.

## Decision Drivers

- Allow ordinary comments, CI progress, review updates and label/project administration during work.
- Preserve source and command authority checks.
- Keep live merge readiness separate from source review validity.

## Considered Options

1. Upgrade Oryn to separate task interruption from live operation gates.
2. Rerun the model after every context-digest change.
3. Trust captured CI and approval state for merges.

## Decision Outcome

Choose option 1. This supersedes decision 0016's context-digest and active-label locking. Active polling
and publication check title/body, exact source SHAs, the original command, current author permission and
authorized stop commands. Context is a review snapshot, not a publication lock. New comments, CI/review
updates and label/project changes do not discard results. Protected labels continue to filter new work.

Repository eligibility and branch permissions remain publication prerequisites. Merge separately checks
current CI, mergeability and independent approval, and publishes a waiting report when they are unmet.
Closure retains source evidence and linked-PR checks. There is no change to model/provider settings,
credentials, capability opt-ins or the Actions-only ephemeral runtime.

## Pros and Cons of the Options

- Option 1 preserves completed source work while checking the actual write's current requirements.
- Option 2 is simpler but makes normal parallel CI and review activity waste model runs.
- Option 3 avoids waiting but permits merging after failed CI or withdrawn approval.

## Links

- [Operations guide](../operations/oryn.md)
- [Previous freshness decision](0016-oryn-semantic-freshness.md)
- [Observed failure](https://github.com/YourTongji/YourTJ-Hub/actions/runs/34431404905)
