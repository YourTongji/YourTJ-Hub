# Bounded feed ranking and account-level measurement

## Status

Proposed
Class: architecture

## Context and Problem Statement

Cumulative replies/views give old topics an indefinite advantage. The forum shares a five-connection
PostgreSQL pool with HTTP and existing workers and must retain its single Go/Vue binary. Personalized
ranking also needs a measurable outcome, correct delayed-publication attribution and finite personal
data retention rather than an unbounded event stream.

## Decision Drivers

- Preserve explicit Latest, guest and old-client behavior.
- Rank only currently public content and eligible contributors, including internal anonymous deduplication.
- Bound SQL, memory, queue sizes, retries, rebuilds and personal retention.
- Evaluate default entry by account assignment, including zeros, fallback and tab switching.
- Keep admin observations anonymous and distinguish measurements from causal claims.

## Considered Options

- Sort cumulative counters directly on every request.
- Add an external recommendation/event-processing service and continuously refresh every historical topic.
- Materialize rule scores in the primary DB with one bounded worker and finite per-account snapshots.

## Decision Outcome

Choose the third option. Hot and rolling Daily scores use immutable, separately hashed parameters;
transactional dirtiness, generation/version/due fencing and public-time watermarks protect updates.
Actor eligibility changes queue bounded owner batches; quiet old topics have no periodic timer. Historical
backfill is resumable outside startup migrations and remains unavailable until all public topics match.

For You builds a finite candidate set and 30-minute snapshot, with sequential reads and current hard
filter checks on every continuation. Explicit Latest is retained; strict page capability v2 protects
old mobile clients. Cold-build saturation or timeout returns a bounded Latest page rather than mixing
snapshot cursors with offset pagination.

Raw identifiable observations/assignments/samples expire after at most30 days, with24-hour ranking
facts and account-close ingestion fencing. Backup copies exclude raw records and derived work. A process
epoch invalidates transient cursors/traces and aborts an experiment after incomplete shutdown/restore.
Only anonymous aggregates and parameter history persist long term. Defaults disable the feature;
initial account-level default-entry rollout is20%.

Default-entry and ranking comparisons have independent assignments. Fourteen-day enrollment followed
by an eight-natural-day account observation window fits30-day retention. Anonymous n/sum/sumSquares
support difference-of-means intervals only when each observed group has at least30 accounts. Missing
and closed trajectories retain their assigned denominator and are explicitly reported; intervals
cannot correct missingness. Shared interleaving items/exploration do not supply team wins. Independent
candidate samples support finite-pool deterministic replay, not historical recall or causal estimation.

## Pros and Cons of the Options

- Direct counters minimize write cost but preserve age/popularity feedback and require expensive request
  aggregates for time windows; there is no controlled measurement or read-time budget.
- External processing scales independently but adds deployment/state/recovery boundaries incompatible
  with the single-binary objective. Refreshing every historical topic costs work even without change.
- Materialization/snapshots provide indexed reads, bounded resources and inspectable rules. They give
  up instantaneous freshness, durable session snapshots, full candidate replay and automatic model
  fitting. Queue loss and finite candidate recall remain visible limitations; production capacity and
  statistical value require separate evidence.

## Links

- [Product contract](../product/feed-ranking.md)
- [Operations and recovery](../operations/feed-ranking.md)
- [Issue1056 design and review](https://github.com/YourTongji/YourTJ-Hub/issues/1056)
- [Viper concurrent reads/writes FAQ](https://github.com/spf13/viper#is-it-safe-to-concurrently-read-and-write-to-a-viper-instance)
- [Upstream popularity ranking](https://github.com/leancodebox/GooseForum/commit/50caa91d)
- [Team-draft interleaving research](https://www.microsoft.com/en-us/research/publication/interleaving-ranking-functions/)
