# Daily XP settlement operations

> Doc type: operations tutorial
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-27

## Supported operation

`gooseforum xp-settle-daily` previews or applies the `daily_base` reward for exactly one
Asia/Shanghai calendar date. The command defaults to dry-run. It does not schedule itself,
loop over a date range, or backfill dates since account registration.

Use a dry-run to inspect aggregate counts:

```bash
gooseforum xp-settle-daily --date 2026-10-26 --dry-run
```

An actual settlement requires both `--apply` and the operator-selected enablement boundary:

```bash
gooseforum xp-settle-daily --date 2026-10-26 --apply --start-date 2026-10-26
```

`--date` and `--start-date` must be exact `YYYY-MM-DD` values. Future dates are rejected.
An apply request earlier than `--start-date` is rejected. The service enforces this boundary for
all callers, including future scheduled callers. The command processes only the one date passed in
`--date`; it never advances the date or catches up other days automatically.

## Eligibility and event time

The service scans `users` by increasing numeric ID in bounded batches. It uses the existing
`xpservice.EligibleForXP` criteria: the account must be human, active, not deleted, and verified
when the current email-verification setting requires it. No Tongji SSO binding or daily activity
is required. Anonymous personas are not scanned or given separate accounts; the real forum
`users.id` receives at most one base award for the date.

Accounts created after the target Shanghai date are outside the scan. The day is evaluated as the
Shanghai half-open interval `[00:00, next 00:00)`. Published rule sets each cover
`[effective_at, next effective_at)`; the service intersects these windows with the target day and
only treats intervals whose set and `daily_base` rule are enabled as award windows. A rule effective
exactly at its start is active; the next version's exact effective time is excluded from the prior
version.

For an eligible account, the event time is the later of its date-specific eligibility time (account
creation, or email activation when required) and the start of the first valid rule window that
covers a moment at or after that eligibility time. Historical settlement may use a later window
from the target date. On the current date, the account must already be eligible and the selected
rule window must have started by execution time. The event time is never later than the actual
execution time. A future rule version is not used early; if today's only valid window starts in the
future, the date is reported as temporarily unavailable. When the date has a valid window but a
particular account becomes eligible only after every valid window ended, that account is skipped and
the run continues.

The user table does not retain historical freeze-state or site-setting changes. A past-date run
therefore reports a warning and applies the current account-status rules; it does not claim to
reconstruct a historical freeze state. When email verification is required, a past-date account
without `activated_at` is skipped and counted as unverifiable rather than assigned a fabricated
activation time. Review the warning and these counters before applying a past date.

## Idempotency and failures

Each account/date uses the stable source key
`xp:daily_base:YYYY-MM-DD:user:<numeric users.id>`, for example
`xp:daily_base:2026-10-26:user:1760`. The service sends each new award through `ApplyReward`,
which commits the immutable ledger entry, balance projection, and daily quota in one transaction.
Repeating the same date returns already-settled counts without a second award. If the process is
interrupted after some accounts commit, rerun the same date; completed entries are recognized and
remaining eligible accounts continue. Database failures return a non-zero exit and aggregate
failure count; earlier account transactions may already have committed, so the same-date retry is
the recovery path. Expected qualification and quota rejections are counted as skipped.

The JSON output contains the date, mode, status, scanned/eligible/candidate/awarded/already-settled/
skipped/unverifiable-history/failed counts, and any warning. Counts have these meanings:

- `scanned` counts rows read by the bounded numeric-ID cursor.
- `eligible` counts not-yet-settled accounts that pass the current static `EligibleForXP` check. It
  can overlap `skipped` when an eligible account has no date-specific window or is later rejected by
  quota.
- `candidates` counts pre-quota accounts that pass account and occurrence-time checks and have no
  existing settlement key. Apply mode retains this count if `ApplyReward` later rejects the reward
  for quota or a concurrent eligibility change. If another run commits the same source key first,
  that account is reclassified as `already_settled`.
- `awarded` counts new committed entries. `already_settled` counts existing source-key entries,
  including a concurrent settlement found by `ApplyReward`.
- `skipped` counts expected static-eligibility, date-window, unverifiable-history, or quota
  rejections. `unverifiable_history` is included within `skipped`.
- `failed` counts fatal errors observed before the run returns; earlier transactions may already
  have committed.

These counters are operational measures, not mutually exclusive buckets, and must not be added to
derive a total account count. Dry-run does not query `ApplyReward` quotas, so `candidates` means
pre-quota candidates and is not a prediction of how many accounts Apply mode will award. The JSON
contains no account IDs or ledger rows. `dry_run` means this service did not write XP ledger,
balance, or quota data. The Cobra root command still runs normal database migrations before this
subcommand and starts the process event bus, so dry-run is not a whole-process read-only invocation.
A migration failure may stop execution before the settlement summary is emitted.

## Rule publication and scheduling boundary

The active seeded `daily_base` rule is +5 XP once per Shanghai day. The 65 XP daily positive-reward
ceiling is approved as a product rule but is not present in the currently seeded immutable rule set.
Do not treat it as active until a separate rule-publication change appends a version containing that
limit. This operation does not edit published rules.

There is no Cron registration for XP. A later scheduled integration belongs in
`apps/gooseforum/app/console/job/cron.go` and should call the same `xpservice.Service.SettleDailyBase`
entry point with an explicit date, configured start date, and cancellation context. It should retain the same
source-key, start-boundary, historical-eligibility, and aggregate-reporting semantics.
