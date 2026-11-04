# Bounded public avatar reuse

## Status

Proposed
Class: bug-fix

## Context and Problem Statement

Uploaded avatars carry a no-store response, so compliant clients download and decode them again on
navigation. Even fresh reusable images pass through an asynchronous byte loader before Flutter can
consult its decoded cache, which exposes a placeholder on a new page.

## Decision Drivers

- Reuse avatars across pages without bypassing HTTP restrictions.
- Bound visibility after withdrawal and preserve pending/private upload access.
- Retain existing account/site/language and cache-clear boundaries.

## Considered Options

1. Keep no-store for uploaded avatars and only fix fresh decoded-cache hits.
2. Give active public avatars five minutes of HTTP freshness and reuse fresh decoded frames synchronously.
3. Retain no-store responses in an App session cache.

## Decision Outcome

Choose option 2, using the maintainer-approved five-minute withdrawal window for public avatars.
Only successful image reads backed by an active, unexpired user-avatar reference receive the public
header; derived sizes resolve their parent reference. New uploads use new URLs. Other upload policies
remain unchanged. The App's host confirms freshness before looking up a decoded frame synchronously;
a miss or expiry returns to the byte loader. No-store/private/no-cache frames remain outside ImageCache.

## Pros and Cons of the Options

- Option 2 benefits both Web and App and keeps disk policy aligned with HTTP. Clients may display
  an already cached withdrawn avatar for up to five minutes; memory eviction can still require decoding.
- Option 1 preserves immediate reauthorization but leaves repeated downloads for uploaded avatars.
- Option 3 avoids a server change but conflicts with no-store and creates a separate retention policy.

## Links

- [Avatar caching report and source analysis](https://github.com/YourTongji/YourTJ-Hub/issues/1578)
- [Public avatar behavior](../product/forum.md#public-avatar-caching)
- [Mobile cache boundaries](../architecture/mobile-state-and-cache.md#cache-policy)
