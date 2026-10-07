# For You viewport seen state and stable browsing sessions

## Status

Accepted
Class: feature

## Context and Problem Statement

A reader returning from a detail page loses their For You position when native actions or background
ranking change scores. The prior first-page reuse window cannot distinguish return from refresh.
The maintainer chooses card viewport exposure as seen, 30-day retention, and eligibility again when
another account publishes a new reply. This extends [0065](0065-bounded-feed-ranking-and-measurement.md)
with explicit browsing and discovery semantics; its ranking, deployment and experiment boundaries remain.

## Decision Drivers

- Preserve loaded order, pagination and an anchor on browser/PWA return.
- Make explicit refresh discover unseen cards, including when analytics is disabled.
- Use displayed content time so delayed exposure reporting cannot swallow a new reply.
- Bound SQL, client memory, writes and identifiable data with the existing pool and worker.
- Preserve Latest/Following/Hot/Daily and global detail-visit unread semantics.

## Considered Options

- Extend the 30-second server snapshot reuse window.
- Filter only on the client or use detail visits/analytics queue as seen state.
- Add an asynchronous per-reader latest-reply projection or an exposure event log.
- Keep independent browsing sessions, durable aggregate seen ACKs and bounded public-reply probes.

## Decision Outcome

Choose independent sessions and aggregate seen state. A foreground card at least 50% visible for one
continuous second qualifies. Signed public-card proofs bind owner, positions, content cutoff and process
epoch for 30 minutes. They constrain claims to the owner's recommendations; they do not prove gaze,
create opened events or credit rank/points. Capture commits one `(user,topic)` aggregate before ACK,
uses the account-close fence and expires after 30 days. Replays and older cutoffs cannot extend retention.

Seen without a currently eligible other-account public reply after the displayed cutoff is removed
before all candidate selectors, including latest supplementation and comparisons. Author replies can
qualify; own replies, edits, restored old replies and ineligible content cannot. Short/empty pages are
valid. Candidate reads remain at most 300, snapshots 120, pages 20; cold builds stay at two/200 ms.
One correlated posts-owner query probes the public-time index and current identity/block eligibility;
its deadline bounds long ineligible reply chains. No new projection, pool or full-history scan is added.

Web/PWA caches at most two history-entry sessions, 120 cards each, with a combined 512 KiB JSON payload
budget including a recovery shell and a 30-minute inactive TTL. Return/foreground/SSE reconcile only
loaded IDs, update state in place and remove authoritative hard-ineligible rows. Original order, cursor
and attribution stay intact; deleted anchors fall forward then backward. Explicit refresh confirms
pending seen state then builds a new batch. Failure and expired cursors retain the existing list;
evicted sessions show a recovery prompt. Account changes clear memory; nothing is persisted to disk.

The events route preserves boolean success for old metric-only requests. Shared OpenAPI, TS and Dart
mirrors describe proofs/ACK, refresh and reconciliation. Flutter functional viewport capture and stable
browsing restoration remain Partial. Signed-but-unconfirmed claims cannot survive proof expiry or
process/browser loss; the guarantee applies after ACK and to in-document refreshes carrying valid pending.
Analytics remains disposable and separate. Seen state joins existing raw cleanup and backup exclusion.

## Pros and Cons of the Options

- Longer reuse is cheap but conflates return with discovery and still expires into reorder.
- Client filtering lacks cross-device consistency; visits violate the selected viewport definition,
  and disposable/disabled analytics cannot guarantee immediate-refresh exclusion.
- Reply projections/event logs add rebuilding, freshness, storage and privacy costs without being
  necessary for the current finite pool.
- Independent sessions/aggregate ACKs make semantics inspectable and preserve position, at the cost
  of viewport-volume batch writes and indexed reply probes. Finite recall may yield empty pages;
  synthetic plans do not establish production capacity, statistical benefit or durable offline history.

## Links

- [Product behavior](../product/feed-ranking.md)
- [Operations and capacity boundaries](../operations/feed-ranking.md)
- [Seen aggregate owner](../../apps/gooseforum/app/models/forum/feed/seen.go)
- [Bounded reply probes and PG regression](../../apps/gooseforum/app/models/forum/posts/seen_reply_pg_test.go)
- [Web/PWA browser acceptance](../../apps/gooseforum/resource/test/for-you-session.browser.mjs)
- [Discourse official reading-statistics distinction](https://meta.discourse.org/t/understanding-stats-for-topic-views-posts-read-and-reading-time/254542)
  (reviewed 2026-10-07; only the distinction between opens, read events and time is borrowed, not its algorithm).
