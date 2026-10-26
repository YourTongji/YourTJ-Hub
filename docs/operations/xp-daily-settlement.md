# Daily XP settlement operations

> Doc type: operations tutorial
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-26

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

Accounts created after the target Shanghai date are outside the scan. The event time is on the
target date and is moved forward to account creation, email activation when required, or the
effective time of the active `daily_base` rule when one of those occurs later in that day. A
creation or activation time outside the target date cannot create an award for that date.

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
skipped/unverifiable-history/failed counts, and any warning. It contains no account IDs or ledger
rows. `dry_run` means this service did not write XP ledger, balance, or quota data. The Cobra root
command still runs normal database migrations before this subcommand and starts the process event
bus, so dry-run is not a whole-process read-only invocation. A migration failure may stop execution
before the settlement summary is emitted.

## Rule publication and scheduling boundary

The active seeded `daily_base` rule is +5 XP once per Shanghai day. The 65 XP daily positive-reward
ceiling is approved as a product rule but is not present in the currently seeded immutable rule set.
Do not treat it as active until a separate rule-publication change appends a version containing that
limit. This operation does not edit published rules.

There is no Cron registration for XP. A later scheduled integration belongs in
`apps/gooseforum/app/console/job/cron.go` and should call the same `xpservice.Service.SettleDailyBase`
entry point with an explicit date, configured start date, and cancellation context. It should retain the same
source-key, start-boundary, historical-eligibility, and aggregate-reporting semantics.
