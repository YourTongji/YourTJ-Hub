# Keep Oryn scheduled catch-up without repeating completed reviews

## Status

Accepted
Class: process

## Context and Problem Statement

The six-hour Oryn scan can review an already reported issue after cooldown expiry,
new discussion or an unrelated default-branch commit. Queued PR events can also
review the same latest source repeatedly. Maintainers want catch-up for missed
events and unfinished work while reserving another completed review for a changed
source version or an explicit request.

## Decision Drivers

- Preserve missed-event, failure and waiting-task recovery.
- Avoid model work caused only by age, discussion or repository administration.
- Review changed PR source without inheriting an obsolete lease or exhausted lane.
- Preserve exact-source cancellation, publication checks and manual review authority.

## Considered Options

1. Disable scheduled scans and rely exclusively on events.
2. Extend the current cooldown.
3. Keep six-hour catch-up and select completed reviews once per source version.

## Decision Outcome

Choose option 3. The operator policy uses `reviewPolicy=once_per_version` in the
pinned Oryn runtime. Both scans and exact automatic events skip a published review
of the current version. PR identity includes head/base SHAs and title/body; issue
review identity includes title/body independently of default-branch movement.
Exact source SHAs still control active-task interruption and publication.

Unreviewed items, authorized commands, due retries, automatic implementation and
generated-PR continuation remain eligible. Explicit selected manual reviews and
new authorized review commands can review completed work again. Same-version
leases, retry deadlines, exhaustion and protected labels remain enforced.

Legacy issue reports have no text-only fingerprint. Catch-up preserves them;
an edited-issue event or explicit review replaces them with a versioned report.
If a legacy text-edit event is lost, an explicit review is needed. Legacy PR
reports still identify their exact source. Receipts and assistance answers are
not review completion. No credential or receipt-schema change is required.

## Pros and Cons of the Options

- Option 1 reduces scanning but loses automatic recovery of missed and unfinished work.
- Option 2 delays duplicate work without fixing event deduplication or source identity.
- Option 3 preserves recovery and source checks, with an explicit legacy issue compatibility limit.

## Links

- [Operations guide](../operations/oryn.md)
- [Repository policy](../../.github/oryn/repositories.json)
- [Existing interruption rules](0017-oryn-task-interruption.md)
- [Oryn selection decision](https://github.com/yzxoi/oryn-mini/blob/main/docs/decisions/0017-review-once-per-version.md)
