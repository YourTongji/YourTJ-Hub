# Distinguish Oryn source changes from repository administration

## Status

Accepted
Class: process

## Context and Problem Statement

Assigning issue #594 during implementation changed its GitHub timestamp. Oryn rejected the pending
decision report despite unchanged source and discussion, then showed a generic publication failure.
Normal issue administration must not discard otherwise current maintenance evidence.

## Decision Drivers

- Preserve exact source, discussion and maintainer authority checks before publication.
- Allow unrelated assignment, project and bot-receipt updates during work.
- Make pending decisions and actual validation status visible.

## Considered Options

1. Upgrade the pinned Oryn runtime to compare source fields and context snapshots separately.
2. Require maintainers to avoid all issue administration during model execution.
3. Disable publication freshness checks.

## Decision Outcome

Choose option 1. The runtime compares item identity, title/body, source SHAs, labels and eligibility,
then verifies the captured discussion/review/check snapshot and live command authority. A result without
a context snapshot cannot publish across timestamp drift. Changes to requirements, code or evidence
remain blocking. Assignee/project updates no longer cause false source-change failures.

Decision-needed repairs publish their question and actual validation status without exporting a patch.
Publication failures identify source-field or context categories without copying source content or raw
errors into receipts. Existing repository policy, model credentials and CI/approval requirements remain
in force. The Actions-only runtime and ephemeral model-session architecture follow decision 0015.

## Pros and Cons of the Options

- Option 1 separates relevant evidence from metadata, but requires maintaining the runtime pin.
- Option 2 needs no code change but blocks ordinary collaboration and does not fix bot timestamp writes.
- Option 3 avoids false rejections but permits stale results or withdrawn authority to publish.

## Links

- [Operations guide](../operations/oryn.md)
- [Repository-local runtime decision](0015-oryn-repository-local-actions.md)
- [Runtime fix](https://github.com/yzxoi/oryn-mini/pull/9)
- [Observed publication failure](https://github.com/YourTongji/YourTJ-Hub/actions/runs/34426777441)
