# YourTJ Status

Standalone Vue/Vite app and Netlify Functions for `status.yourtj.de`. No forum process, database,
login or forum runtime asset is required. Optional visitor-device reports use a server-only Umami
account with read access to the configured website; basic traffic statistics remain public-share based.

## Local development

Use Node 24 and pnpm 11. From this directory:

```sh
pnpm install --frozen-lockfile
cp .env.example .env
pnpm dev
```

Fill `.env` with the public sources documented in the [Netlify runbook](../../docs/operations/status-netlify.md).
Add `UMAMI_USERNAME` and `UMAMI_PASSWORD` for visitor-device reports. Use an ignored `.env.local`
for local secrets; never use `VITE_*` names. `pnpm dev:local` serves both UI and the real snapshot API
at `http://localhost:5247`, including automatic minute/15-minute/hourly collection. This worktree-friendly
preview keeps snapshots in memory and never reads or writes production Blobs. Restart it after
changing server code or environment settings; initial collection can take several seconds.

For Netlify runtime/emulator verification, use `pnpm dev` instead:
Netlify Dev serves the page and API at `http://localhost:8888`. In another terminal, invoke the
collectors once (local schedules do not run automatically):

```sh
pnpm exec netlify functions:invoke collect-current --port 8888
pnpm exec netlify functions:invoke collect-history --port 8888
pnpm exec netlify functions:invoke collect-devices --port 8888
```

Netlify CLI 27.6 currently misidentifies the repository root in Git worktrees (`.git` is a file).
Use a regular checkout for Netlify CLI commands (Dev and function bundling). A clean checkout containing only this app was verified with
`netlify build --offline` and the local collectors; Git-based Netlify builds use a regular checkout.

## Verification

```sh
pnpm test
pnpm build
pnpm exec playwright install chromium
pnpm test:browser
pnpm contract:generate
pnpm exec netlify functions:build --src netlify/functions --functions .netlify/functions
```

Unit and browser tests use explicit fixtures and do not contact real providers. `api/openapi.yaml`
owns the read API; `src/generated/openapi.ts` is generated. `server/` owns public-field adapters,
persistence and read semantics; `netlify/functions/` only wires platform handlers.

Product behavior is documented in the [status specification](../../docs/product/server-status.md).
Deployment settings, environment variables, domain setup and failure checks live in the
[Netlify runbook](../../docs/operations/status-netlify.md).
