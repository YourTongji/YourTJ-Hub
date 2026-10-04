# Dispatch status collection from a separate Cloudflare Cron Worker

## Status

Proposed
Class: architecture

## Context and Problem Statement

GitHub schedule is a best-effort event source and can delay or drop runs. Production public snapshots
can expire despite working providers, storage and manually dispatched collectors. A fifteen-minute
schedule cannot keep a one-hour retention window populated when trigger gaps exceed that window.
The heavy collector exceeds Workers Free CPU limits, and the hosting constraint remains no monthly
subscription. Extending snapshot freshness would hide collection failures instead of fixing them.

## Decision Drivers

- Keep the status site and collection independent from the forum host.
- Preserve real timestamps, failure states and retention rather than displaying old data as live.
- Retain the existing main-only collector, credentials and free standard GitHub runner.
- Keep public serving separate from credentials that can trigger repository automation.

## Considered Options

- Continue relying on GitHub schedule or change its minute offsets.
- Use a separate Cloudflare Cron Worker to dispatch the existing main workflow.
- Move collection to Workers Paid.
- Use the forum host or a third-party cron service.

## Decision Outcome

Use a separate, small Cron Worker to POST the fixed workflow_dispatch endpoint with a fixed main ref
and one of the two fixed collection kinds. It holds only a repository-scoped Actions write token;
there is no HTTP collection endpoint, workers.dev URL, public route, R2 binding or provider credential.
Unknown schedules and dispatch failures fail visibly without logging response bodies or secrets.
An ambiguous POST is not retried automatically. The public Worker remains read-only.

The token grants Actions write to the repository, not only to the named workflow: that residual
authority requires explicit owner approval and finite expiry. Do not reuse a user's broad CLI token.
Production collection still requires STATUS_COLLECTION_ENABLED and the main-only environment.
STATUS_SCHEDULER_ENABLED selects Cloudflare deployment and disables the default-branch fallback
dispatcher; disabling it alone does not remove already deployed Cloudflare Cron triggers.

Timed acceptance requires repeated real Cron executions, successful main collection and advancing
API timestamps; a successful deployment or manual run is insufficient. Runner queues and provider
failures can still delay updates. No scheduling or freshness guarantee is claimed. New Cron CPU must
be measured against the Free limit before declaring acceptance.

This supersedes only the GitHub scheduling choice in [0058](0058-status-cloudflare.md); its reader,
storage, external collection, privacy, retention and main-source boundaries remain in force.

## Pros and Cons of the Options

- GitHub timer: no new credential, but changing offsets does not remove documented event loss/delay.
- Cloudflare dispatcher: removes that event source with little CPU work, but adds a narrowly scoped
  persistent credential and still depends on GitHub dispatch/runner availability.
- Workers Paid: supports heavier native collection but violates the no-subscription constraint.
- Forum host: shares the monitored system's failure domain. An additional cron service adds another
  operator/account and requires dispatch credentials anyway.

## Links

- [Cloudflare runbook](../operations/status-cloudflare.md)
- [GitHub schedule behavior](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#schedule)
- [GitHub workflow dispatch permissions](https://docs.github.com/en/rest/actions/workflows#create-a-workflow-dispatch-event)
- [Cloudflare Cron](https://developers.cloudflare.com/workers/configuration/cron-triggers/)
- [Workers CPU and trigger limits](https://developers.cloudflare.com/workers/platform/limits/)
