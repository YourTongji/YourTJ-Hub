# Shared course search candidates with database validation

## Status

Accepted
Class: architecture

## Context and Problem Statement

Course catalog keyword queries combine text matching across courses, aliases,
offerings and instructors. Repeating these predicates for both counts and pages
consumes shared database capacity. The aggregate search already owns a rebuildable
Meilisearch courses index. Catalog ordering and visibility also depend on live
database state and review statistics.

## Decision Drivers

- Keep the single-binary application and existing deployment dependencies.
- Share keyword retrieval without duplicating course search infrastructure.
- Preserve exact filtered totals, stable review ordering and database visibility.
- Bound resource use and reject incomplete results instead of silently truncating.

## Considered Options

- Optimize all text matching inside PostgreSQL.
- Move catalog filtering, rating ordering and pagination entirely into Meilisearch.
- Retrieve complete bounded keyword candidates from Meilisearch and validate/page them in PostgreSQL.

## Decision Outcome

Use the shared `courses` index for catalog keyword candidates when Meilisearch is
configured. Include visible teaching-class codes and instructor search forms in
the projection. PostgreSQL applies visibility, exact filters, review ordering,
counts and page hydration. Empty keywords use the database directly.

Candidate retrieval uses exhaustive pagination with at most 20,000 IDs. The index
must permit one extra hit to detect overflow. Missing settings, engine failures or
incomplete/oversized results return a retriable 503; they never trigger a costly
SQL fallback. Installations without Meilisearch retain bounded SQL search.

Catalog requests have a four-second deadline and two concurrent execution slots
per process. Excess requests receive 503 with Retry-After. Public filter
dictionaries have a five-minute process-local cache. PostgreSQL connections
default to JIT disabled unless explicitly overridden in the DSN.

## Pros and Cons of the Options

- PostgreSQL-only search preserves substring matching, but keeps text-search work
  coupled to the identity, forum and worker connection pool.
- Full Meilisearch pagination minimizes database work, but requires synchronizing
  review ordering and filter semantics into a second projection and reconciling
  stale/deleted hits with exact pagination.
- Bounded candidates reuse the existing index and preserve authoritative database
  ordering and visibility. They add an ID-set transfer and an explicit capacity
  ceiling. Keyword tokenization follows Meilisearch rather than SQL substring
  semantics. Index lag can delay new matches, while hidden/deleted courses are
  always rejected by database validation.

## Links

- [Aggregate search decision](0003-aggregate-search-multi-index-pinyin.md)
- [Contracts and data](../architecture/contracts-and-data.md)
- [PostgreSQL JIT cost decisions](https://www.postgresql.org/docs/16/jit-decision.html)
- [Meilisearch search pagination](https://www.meilisearch.com/docs/reference/api/search)
