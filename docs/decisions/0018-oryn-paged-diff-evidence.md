# Read large Oryn PR diffs on demand

## Status

Accepted
Class: process

## Context and Problem Statement

PR #611 failed before inference when its roughly 183 KB diff exceeded Oryn's 150 KB inline-evidence
limit. Increasing the configured model context window cannot remove a host-side rejection. Maintainers
need large PRs to enter review with explicit coverage rather than fail solely because of diff size.

## Decision Drivers

- Preserve complete change inventory and access to relevant source evidence.
- Keep model context bounded per read without silently truncating review coverage.
- Retain the Actions-only runtime and current execution/publication permissions.

## Considered Options

1. Upgrade Oryn to paged inventories and on-demand per-file diff reads.
2. Increase the hardcoded inline diff limit.
3. Truncate large diffs before inference.

## Decision Outcome

Choose option 1. The setup action pins Oryn's tested paged-evidence implementation. The host provides
statistics and a complete file inventory, then Core reads bounded diff pages through its existing read
tool. Renames, deletions, binary notices and long lines remain explicit. Independent repair review uses
the same mechanism for the cumulative patch. Evidence lives outside the candidate checkout in temporary
task storage; Core allows reading it but denies external writes, and task cleanup removes it.

The 150 KB PR input rejection is removed. Time and model-step budgets still apply; incomplete review
coverage must be disclosed as needs_human. The separate repair-publication patch limit, validation,
source/command freshness and live close/merge gates remain. No provider, key, App grant or workflow
trigger change is required.

## Pros and Cons of the Options

- Option 1 spends context on relevant evidence and reuses Core's public read capability, at the cost of
  more file reads on large reviews.
- Option 2 is smaller but shifts the failure threshold and continues to inline unrelated evidence.
- Option 3 avoids the exception while concealing missing review evidence.

## Links

- [Operations guide](../operations/oryn.md)
- [Oryn implementation](https://github.com/yzxoi/oryn-mini/pull/11)
- [Observed failure](https://github.com/YourTongji/YourTJ-Hub/actions/runs/34435400271)
