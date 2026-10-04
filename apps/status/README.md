# YourTJ status

`Partial`: independent Vue status app on Cloudflare Workers Static Assets, a read-only Worker API,
and private R2 snapshots. GitHub Actions collects public metrics every fifteen minutes and device
reports hourly, dispatched by a separate Cloudflare Cron Worker; collection never runs in the public
Worker or in response to a visitor. The forum retains its separate single-binary deployment.
The Worker also serves `/mobile/releases.json`, a read-only
proxy of the published mobile release-notes asset. Production delivery and manual public/device
collection are verified; the separate scheduler still requires credential setup and timed acceptance.

## Development

Node 24 and pnpm 11:

```sh
pnpm install --frozen-lockfile
pnpm dev:local
```

`dev:local` is a local collector/API with an in-memory store. Configure ignored `.env.local` from
`.env.example`; without providers the UI shows unconfigured sources. `pnpm dev` builds assets and
starts Wrangler with local R2. Its default configuration has disabled sources and no scheduled jobs.

```sh
pnpm contract:check
pnpm test
pnpm build
pnpm test:browser
pnpm worker:build
```

The HTTP contract is in `api/openapi.yaml`. Query validation, provider sanitization, freshness and
conditional writes are shared by the API and external collector. `worker:build` checks both default
preview and production reader configuration, plus the production scheduler. Browser tests cover four
languages, themes and mobile/desktop sizes. A real Workers-runtime test checks upstream fetch and
credential-safe redirect rejection.

## Deployment

See [the Cloudflare runbook](../../docs/operations/status-cloudflare.md). Production and preview use
separate private buckets. Only reviewed main is deployed. `Deploy / status` verifies the build tree
and API; the separate Cron Worker dispatches `Collect / status` on main. The GitHub dev timer is a
best-effort fallback, disabled when `STATUS_SCHEDULER_ENABLED=true`.
The collector's workflow definition and checked-out source come from the same main commit, and its
environment allows only main. Freeze Netlify automatic builds before removing its handlers from main;
keep the existing deployment until Cloudflare cutover passes acceptance.

The collector uses bucket-scoped S3 credentials in the `status-collector` GitHub environment. Only
that environment holds Umami credentials. The Worker has no upstream credentials or public collect
endpoint. Public `UMAMI_DEVICE_REVISION` isolates device snapshots when an account or permission
changes; clear it to disable the device source, or increment it to invalidate previous snapshots.

Public collection also publishes twelve range views. An API request reads at most one public view and
one device object; views retain each source's original timestamp, failure flag and scope fingerprint.

Cloudflare dispatch avoids GitHub schedule delays, but runner availability and provider failures can
still delay collection, so timestamps and stale/unavailable states remain visible. Current,
history and traffic sources are stale after twenty minutes and hidden after one hour; devices are
stale after seventy minutes and hidden after three hours. Polling and caching never freshen source
timestamps. The site is not a replacement for the independent uptime monitor.
