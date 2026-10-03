# Budgeted status collection on Netlify

## Status

Superseded by [0057](0057-status-cloudflare.md)
Class: architecture

## Context and Problem Statement

The independent status site consumes credits for successful production deploys and for scheduled
function execution, including time waiting for upstream data. Short-term service health and long-term
visitor reports do not need the same refresh interval. A build cache can contain an unpublished
preview, so it cannot prove that production already contains a change.

## Decision Drivers

- Keep the existing status UI, providers and independent forum failure boundary.
- Preserve minute-level availability collection while reducing noncritical work.
- Keep stale data visibly stale and retain it only for a bounded source-specific period.
- Avoid production deployments only when the same status content is demonstrably published.

## Considered Options

- Keep uniform five-minute analytics collection and always build production.
- Migrate the status application to another hosting provider or a managed host.
- Keep Netlify with separate schedules and a published content fingerprint.

## Decision Outcome

Keep Netlify Functions and Blobs. Current metrics and Uptime run every minute, resource history and
traffic every fifteen minutes, and joint device reports hourly at minute seven. Device authentication
does not run with history. The public API remains read-only; browser polls run every minute and the
CDN caches for thirty seconds. Existing public projection and credential boundaries remain as defined
in [0046](0046-status-device-aggregates.md).

Current snapshots stay fresh for 150 seconds and retain data for fifteen minutes. History and traffic
stay fresh for twenty minutes and retain data for one hour. Devices stay fresh for seventy minutes
and retain data for three hours. Failures preserve the original success timestamp and immediately mark
retained data stale. Browser checks use the same policies; Uptime's latest detection still has its
independent health deadline. Older collectors cannot overwrite newer snapshots.

Production builds compare the current `apps/status` Git tree with a small manifest served by the
published production site. Only a matching production manifest permits skipping. Previews, malformed
or missing manifests, network failures and unavailable Git objects do not. Empty-cache and same-commit
rebuilds continue; `STATUS_FORCE_BUILD=true` explicitly applies environment-only changes. The manifest
contains no secrets and is generated with the deployment artifact, not saved on build success alone.

This refines the scheduling and retention policy of [0027](0027-independent-status-netlify.md), while
preserving its independent deployment, provider projection and conditional-write model.

## Pros and Cons of the Options

- Uniform collection: simple, but repeats expensive long-period queries unnecessarily.
- Provider migration: may reduce the bill, but adds migration, compatibility and operational work.
- Separate schedules: keeps health freshness and existing infrastructure, at the cost of slower
  analytics updates and one additional scheduled handler. Published manifests avoid unrelated deploys,
  while uncertainty intentionally favors rebuilding rather than accidentally missing a release.

## Links

- [Status product semantics](../product/server-status.md)
- [Deployment and forced rebuilds](../operations/status-cloudflare.md)
- [Netlify credit meters](https://docs.netlify.com/manage/accounts-and-billing/billing/billing-for-credit-based-plans/how-credits-work/)
- [Netlify build environment and cached commit semantics](https://docs.netlify.com/build/configure-builds/environment-variables/)
