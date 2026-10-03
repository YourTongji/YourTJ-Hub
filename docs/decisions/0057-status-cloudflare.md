# Independent status hosting on Cloudflare with external collection

## Status

Proposed
Class: architecture

## Context and Problem Statement

Netlify credits can suspend the independent status site even after collection and deployment frequency
are reduced. The workload is a static page, bounded HTTP collection and replaceable public snapshots.
A real cloud probe shows the history collector exceeds Workers Free's 10ms CPU allowance. A paid
Workers subscription is outside the zero-monthly-fee requirement. The repository is public, so standard
GitHub-hosted runners are available without minute charges.

## Decision Drivers

- Keep delivery and collection independent from the monitored forum.
- Stay on free plans within their quotas, without relying on temporary CPU burst tolerance.
- Reuse provider adapters, privacy boundaries, timestamps and conditional-write ordering.
- Use the existing Cloudflare account and DNS zone without moving other applications.

## Considered Options

- Keep Netlify with a larger credit allowance.
- Workers Static Assets, Worker Cron and R2 on Workers Paid.
- Workers Static Assets and a read-only API, with GitHub Actions collection into private R2.
- Cloudflare Pages with a separate collector and D1 or KV storage.
- Serve the status app on the forum host.

## Decision Outcome

Use Workers Static Assets, a read-only Worker API and a private R2 Standard bucket. Collect public
metrics/history/traffic every fifteen minutes and optional device aggregates hourly on standard Linux
GitHub Actions runners. No HTTP route starts collection, and no Worker Cron or upstream credentials
are deployed. Worker API requests remain subject to Free CPU/request limits and require cloud testing.

GitHub's default branch is dev; schedules execute there but checkout reviewed main. The collector
uses a separate environment with bucket-scoped S3 credentials and private Umami credentials. A repository
variable enables collection only after production code and credentials are ready. Deployment credentials
use a main-only environment; status checks precede deployment and a source fingerprint verifies it.

Public collection preassembles twelve range views so each API miss needs at most two R2 reads. Views
retain source scope fingerprints, failure flags and timestamps; projection time never freshens data.
R2 strong reads and conditional puts preserve attempted-at ordering. Production snapshots survive
deployments; preview uses a separate bucket. Dynamic responses remain no-store and do not use Cache
API, whose response-copy CPU spikes are unsuitable for Free. Storage failure returns noncacheable 503.
Source scope fingerprints invalidate revoked data immediately. A public device revision identifies the credential
scope without putting secrets in Worker bindings; revoke or increment it when access changes.

GitHub schedules can be delayed or dropped. The page displays actual sample times, treats current,
history and traffic as stale after twenty minutes and hides them after one hour. Devices retain their
seventy-minute freshness and three-hour retention. The independent Uptime Kuma service remains the
monitoring and alerting system. Static delivery and normal collection fit free allowances; R2 overages
can still be billed and changing repository visibility requires reassessing runner charges.

Supersedes [0027](0027-independent-status-netlify.md) and
[0053](0053-status-collection-cost.md) for hosting, deployment and collection cadence. The forum's
single-binary deployment and the existing device aggregate privacy boundary remain unchanged.

## Pros and Cons of the Options

- Netlify: least code change, but preserves the credit-exhaustion failure mode.
- Paid Worker Cron: more predictable collection and one platform, but imposes a monthly subscription.
- Worker reader + Actions: keeps the heavy work off the CPU-limited edge and uses public-repository
  free runners; gives up minute-level freshness and guarantees about scheduling latency.
- Pages + separate collector: another deployment without eliminating the collector's CPU constraints.
  D1 adds unnecessary SQL semantics; KV's free write allowance leaves little headroom for this workload.
- Forum host: avoids another provider but shares the monitored service's failure domain.

## Links

- [Status behavior](../product/server-status.md)
- [Cloudflare runbook](../operations/status-cloudflare.md)
- [Workers pricing](https://developers.cloudflare.com/workers/platform/pricing/)
- [Workers limits](https://developers.cloudflare.com/workers/platform/limits/)
- [Static assets billing](https://developers.cloudflare.com/workers/static-assets/billing-and-limitations/)
- [R2 conditional writes](https://developers.cloudflare.com/r2/api/s3/api/)
- [R2 consistency](https://developers.cloudflare.com/r2/reference/consistency/)
- [R2 pricing](https://developers.cloudflare.com/r2/pricing/)
- [GitHub Actions billing](https://docs.github.com/en/billing/concepts/product-billing/github-actions)
- [GitHub schedule behavior](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#schedule)
