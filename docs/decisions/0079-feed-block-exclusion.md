# Feed hides blocked peers in either direction

## Status

Accepted
Class: feature

## Context and Problem Statement

Reader feedback (issue #1064): after blocking a user, that user's public topics still appeared in
the reader's home, category and following feeds. [0040](0040-user-blocks-and-private-message-reports.md)
scoped blocks to stopping direct contact and kept public content visible there. The For You ranking
already hard-excludes authors with a block relation in either direction
([0065](0065-bounded-feed-ranking-and-measurement.md), [0078](0078-for-you-viewport-seen-and-browsing-sessions.md)),
but the shared public lists — home latest/hot/popular/new, category pages, the following feed and the
topic-detail hot sidebar — served viewer-independent queries with no block predicate, so a reader
could block someone and still see them on every explicit list.

## Decision Drivers

- Blocking must cover content consumption, not only direct contact.
- One semantic across the platform: private messages, interaction notifications and For You all
  treat a block in either direction as authoritative; explicit lists must not disagree.
- The exclusion must not disclose which side created the block.
- Pagination correctness: filtering after fetching would break page size and HasNext; the predicate
  belongs in the list query.
- The 5-second shared list cache has no viewer dimension; viewer-filtered output must never enter it.
- Feeds are public content; a transient storage failure is not an authorization failure.

## Considered Options

- One-way exclusion: only the viewer's own block list hides authors from feeds.
- Two-way exclusion consistent with For You and interaction blocking.
- Extend the exclusion to search results and reply floors inside other topics in the same change.

## Decision Outcome

Chosen: two-way exclusion. Public feed lists exclude authors related to the viewer by a block in
either direction — the viewer blocked the author, or the author blocked the viewer. Covered surfaces:
home latest/hot/popular/new, category pages, the following feed and the topic-detail hot sidebar
(wiki detail pages reuse it). The scalar set matches For You's eligibility filter
(`users.BlockedPeerIDsQuery`, the same relation as `users.FeedAuthorsQuery`); it never discloses
which direction caused an exclusion, and a shortfall from filtering stays a short page.

A viewer with any block relation bypasses the shared list cache and loads a viewer-scoped page; every
other request keeps the shared cached list, so mainstream traffic is unaffected. The existence check
that picks the path is itself a per-reader decision, so it is answered from a per-reader 5-second
in-process cache instead of a database query on every logged-in request (issue #1129); the block
write path invalidates both sides' entries after commit, so block changes apply to lists immediately
instead of waiting out the TTL. The relation check fails open on a storage error — a moment of
unfiltered public list beats a blank page — and a failure is never cached as filtered output; a
failed existence check is not cached either, so fail-open is re-evaluated on the next request.
Bulk relation removals outside the write path (account close) need no invalidation: they can only
leave a cached true behind, which keeps a viewer-scoped query with correct output and is bounded by
the TTL.

Boundaries: direct links to a blocked author's topic, aggregate search results and the author's
replies inside other topics remain available; the block removes authors from feeds, not content from
the platform. This record supersedes the public-feed visibility clause of
[0040](0040-user-blocks-and-private-message-reports.md); its block storage, interaction blocking,
notification filtering and private-message report-evidence decisions remain in force.

## Pros and Cons of the Options

- One-way exclusion follows the narrowest reading of "I do not want to see them", but it contradicts
  For You and interaction blocking and leaves the same pair visible or hidden depending on which tab
  the reader opens, with no way to explain the difference.
- Two-way exclusion keeps one platform semantic and matches the For You hard filter that shipped
  first. A reader can infer an author may have blocked them when topics disappear; that inference
  already existed in For You, and the API never confirms the direction or the counterparty.
  Anonymous topics filter with their real author like For You already does; a reader who already
  holds candidate identities could still use block toggling as a weak membership test. No
  anonymous-content exemption applies.
- Extending to search and reply floors would widen the change into search-index filtering and post
  rendering, where it would remove content the reader deliberately navigated to; it stays out of
  scope and is documented as a boundary instead.

## Links

- [User blocks and private-message reports](0040-user-blocks-and-private-message-reports.md)
- [Bounded feed ranking and measurement](0065-bounded-feed-ranking-and-measurement.md)
- [For You viewport seen state and stable browsing sessions](0078-for-you-viewport-seen-and-browsing-sessions.md)
- [Feed ranking product specification](../product/feed-ranking.md)
- [Forum product behavior](../product/forum.md)
- [Mobile product behavior](../product/mobile-experience.md)
