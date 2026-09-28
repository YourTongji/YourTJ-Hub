# Optional moment titles stored as empty strings

## Status
Accepted
Class: feature

## Context and Problem Statement

Moments (`contentType=2`) are short-form posts. Both publishers derived a title from the body when the
author left the title field empty — cleaned Markdown, the first line truncated to 60 characters, or an
image-only placeholder — and every list and detail surface rendered the stored title. The product
requirement is that an untitled moment publishes without a derived title and displays without one.
Questions and articles still require titles, and the topic write endpoint enforced `title` with
`validate:"required"` plus a configurable minimum length for every content type.

## Decision Drivers

- An empty moment title stays empty; the server never invents one from the body.
- Questions and articles keep the required-title contract and minimum-length validation unchanged.
- No sentinel titles, no model or migration change, and no search-projection field change.
- List and detail surfaces stay navigable and accessible when the title is absent.

## Considered Options

- Store an empty title for moments, relax validation conditionally, and hide empty titles in every display surface.
- Keep the derived text in the title column and hide it only in the frontend.
- Store a sentinel title (the placeholder copy) and let clients treat it as "no title".
- Migrate existing moments to clear derived titles.

## Decision Outcome

`WriteTopicRequest.title` may be omitted or empty only for `contentType=2`; the write core
normalizes absent and whitespace-only moment titles to `""` and stores that.
Questions and articles with an empty title keep the previous `common.request.invalidParams` response,
and non-empty titles keep the configurable minimum/maximum length checks. Agent writes keep their own
non-empty title requirement. A moment with an explicitly typed title keeps it.

Clients send `title: ""` for untitled moments and no longer derive one from the body or image
placeholders. Every title display surface renders nothing for an empty title: web list rows (the row
link falls back to the body excerpt so navigation and accessible names survive), feed cards, profile
lists, the topic detail `h1` and scroll header, the SSR crawler templates, and the Flutter list cards
and detail header (whose app bar falls back to the generic localized "topic" label). Share text,
delete confirmation and notification copy fall back to the body excerpt or a generic string instead of
an empty title.

Search stores the empty title and keeps the body in the searchable `searchContent` field, so untitled
moments remain findable without a reindex. Moments that already carry a derived title keep it; no
backfill is performed.

## Pros and Cons of the Options

- Empty stored title: the data reflects what the author wrote and derived text cannot resurface in new
  surfaces. Display code must treat an empty title as valid, and label contexts need a fallback.
- Frontend-only hiding: no contract change, but derived titles stay in the database and leak back
  through search results, notifications, feeds and future consumers; rejected.
- Sentinel titles: detectable only by matching localized copy, and they pollute search and any
  downstream consumer; rejected.
- Migrating existing moments: derived titles cannot be told apart from titles the author typed, so a
  backfill would silently erase author-visible titles; rejected.

## Links

- [Forum product spec](../product/forum.md)
- [OpenAPI contract](../../packages/api-contract/openapi.yaml) (`WriteTopicRequest`)
- [Issue #895](https://github.com/YourTongji/YourTJ-Hub/issues/895)
