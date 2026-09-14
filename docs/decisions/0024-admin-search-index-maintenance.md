# Admin search index maintenance

## Status

Accepted
Class: architecture

## Context and Problem Statement

Deploying a new binary does not update existing Meilisearch documents. Operators
need to inspect projection versions and reconcile every managed index without
turning a long scan into an HTTP request. The deployment shares a small database
and search engine between main and dev; dev receives main database snapshots.

## Decision Drivers

- Preserve live search while maintaining indexes within existing resources.
- Detect missing, extra and outdated documents even when counts match.
- Reuse durable background execution, bounded retries and lease ownership.
- Prevent a copied dev task from modifying production search.

## Considered Options

- Run the existing clearing CLI rebuilds inside admin HTTP requests.
- Build shadow indexes and atomically swap them with live indexes.
- Reconcile live indexes in batches through the existing task queue.

## Decision Outcome

Use a SiteManager-only page and two API operations for all five owned indexes:
topics, users, categories, courses and Wiki paragraphs. Store maintenance in
`task_queue`, with a partial unique index allowing one active maintenance task
per database. Each task records its requesting user and originating instance;
lease checks fence progress and batch boundaries. Maintenance requires an explicit
configuration opt-in. The deployment template enables main and disables dev;
workers skip tasks copied from another configured origin without contacting Meili.

Rebuild by replacing documents in bounded batches, waiting for settings and each
write task to succeed. Browse before removing ghosts, revalidate candidates in
the source database, and replay documents restored during deletion. Recheck after
the rebuild. Search continues using the live index; a failed rebuild leaves
partial progress and can be retried.

Each index has an explicit projection revision. Verification compares canonical
document content, version markers, IDs and managed settings against public source
projections. It reports observed versions rather than inferring a version from a
successful application deploy. Checks are online observations, not transactional
snapshots. Changes detected during the index scan prevent a complete result.

## Pros and Cons of the Options

- Clearing CLI rebuilds are simple but interrupt search and exceed HTTP deadlines.
- Shadow indexes avoid partially updated documents, but duplicate index storage
  and require coordinating concurrent projection writes across the swap.
- In-place reconciliation uses the existing queue and storage. It preserves
  search availability and supports repair after interruption, but concurrent
  changes can require another check. Operators see the check time and any drift;
  successful job execution alone does not mean the index is complete.

## Links

- [Aggregate search](0003-aggregate-search-multi-index-pinyin.md)
- [Course candidates](0023-course-catalog-search-candidates.md)
- [Operations](../operations/deployment.md#admin-search-index-maintenance)
- [Meilisearch document browsing](https://www.meilisearch.com/docs/reference/api/documents/list-documents-with-get)
