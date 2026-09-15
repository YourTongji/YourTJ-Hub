# Independent public status site on Netlify

## Status

Accepted
Class: architecture

## Context and Problem Statement

A status page must remain reachable when the forum process or host fails. Embedding both its frontend
and upstream adapter in the monitored forum makes that impossible. Uptime Kuma runs independently.

## Decision Drivers

- Isolate page delivery and data collection from the forum failure domain.
- Preserve the existing status UI and public-only data projection.
- Bound upstream traffic independently of visitor traffic and cancellations.
- Retain source timestamps and distinguish stale, missing and zero data.

## Considered Options

- Keep the embedded forum status page.
- Host only the frontend on Netlify and proxy the forum's aggregation API.
- Netlify frontend with request-triggered upstream collection and a shared cache.
- Netlify frontend, scheduled collectors and persisted public snapshots.

## Decision Outcome

Use `apps/status`, an isolated Vue/Vite + TypeScript application in the same repository, with Netlify
Scheduled Functions and Blobs. The forum exposes only a normal external link. Current resource and
uptime collectors run every minute; traffic and resource history every five minutes. A read-only API
serves the selected snapshots behind a 15-second durable CDN cache. Blobs reads use strong consistency;
conditional writes prevent old tasks overwriting new snapshots. Published production data persists
across deploys; other deploys use separate stores. Existing history comes from providers, not a new
historical database. Missing source configuration disables its reads. All provider requests have an
eight-second budget, response-size limits and fixed configured origins.

Supersedes [0026](0026-public-status-projection.md). Forum single-binary deployment is unaffected.

## Pros and Cons of the Options

- Embedded page: easiest reuse, but unavailable during the failure it must report.
- External frontend plus forum API: independent HTML, but data still depends on the failed forum.
- Request collection: lower idle work, but cold starts, concurrent cache misses and upstream failures
  affect readers; a durable CDN cache is not a global single-invocation guarantee.
- Scheduled snapshots: visitors do not trigger provider work; retained snapshots survive collection
  failure. Costs include persistent storage, scheduled compute, minute-level freshness and a separate
  deployment. Preview schedules must be invoked manually, and production billing limits require monitoring.

## Links

- [Product specification](../product/server-status.md)
- [Deployment runbook](../operations/status-netlify.md)
- [Netlify Scheduled Functions](https://docs.netlify.com/build/functions/scheduled-functions/)
- [Netlify Blobs consistency and conditional writes](https://docs.netlify.com/build/data-and-storage/netlify-blobs/)
- [Netlify caching](https://docs.netlify.com/build/caching/caching-overview/)
- [Netlify usage limits](https://docs.netlify.com/manage/accounts-and-billing/billing/resume-paused-projects/)
