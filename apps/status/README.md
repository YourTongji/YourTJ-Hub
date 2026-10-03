# YourTJ status

`Current`: independent Vue status app on Cloudflare Workers Static Assets, a read-only Worker API,
and private R2 snapshots. GitHub Actions collects public metrics every fifteen minutes and device
reports hourly; collection never runs in the Worker or in response to a visitor. The forum retains
its separate single-binary deployment.

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
preview and production configuration. Browser tests cover four languages, themes and mobile/desktop
sizes. A real Workers-runtime test checks upstream fetch and credential-safe redirect rejection.

## Deployment

See [the Cloudflare runbook](../../docs/operations/status-cloudflare.md). Production and preview use
separate private buckets. Only reviewed main is deployed. `Deploy / status` verifies the build tree
and API; `Collect / status` executes main even though GitHub schedules start on default branch dev.

The collector uses bucket-scoped S3 credentials in the `status-collector` GitHub environment. Only
that environment holds Umami credentials. The Worker has no upstream credentials or public collect
endpoint. Public `UMAMI_DEVICE_REVISION` isolates device snapshots when an account or permission
changes; clear it to disable the device source, or increment it to invalidate previous snapshots.

GitHub collection is best effort, so timestamps and stale/unavailable states remain visible. Current,
history and traffic sources are stale after twenty minutes and hidden after one hour; devices are
stale after seventy minutes and hidden after three hours. Polling and caching never freshen source
timestamps. The site is not a replacement for the independent uptime monitor.
