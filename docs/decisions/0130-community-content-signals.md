# Community content signals

## Status

Accepted
Class: feature

## Context and Problem Statement

`robots.txt` already allows the public site and disallows `/api/`, `/admin`, `/login`, and
`/publish`. It does not say whether automated systems may use public forum text for search, as model
input, or for training. A community forum needs those preferences to be explicit without adding a
separate policy site.

## Decision Drivers

- The site owner confirmed the community-site preference: allow search, allow use as model input,
  and disallow training.
- The preference has to stay in the existing `User-agent: *` group so it applies with the current
  allow and disallow rules.
- Search and ordinary reading should keep working. Training is a different use and is not granted.
- The directive must use the Content Signals vocabulary rather than a new product policy format.

## Considered Options

1. `search=yes, ai-input=yes, ai-train=no` on the existing wildcard group.
2. Search only: `search=yes, ai-input=no, ai-train=no`.
3. Allow search, input, and training.
4. Disallow search, input, and training.

## Decision Outcome

Choose option 1. `robots.txt` adds
`Content-Signal: search=yes, ai-input=yes, ai-train=no` to the existing `User-agent: *` group.
Allow, Sitemap, and Disallow lines stay as they are. No separate public policy page is added.

This gives up a training grant and also gives up a search-only posture. Search engines may index
the public site, and the text may be supplied as model input such as grounding. It may not be used
to train or fine-tune models. Crawlers can ignore `robots.txt`; the line records the preference.

## Pros and Cons of the Options

### `search=yes, ai-input=yes, ai-train=no`

- Good, because a campus forum stays findable and usable as a cited source while refusing training.
- Good, because one wildcard group covers the bots that already match `User-agent: *`.
- Bad, because input permission is broader than a pure search index.

### Search only

- Good, because it is the narrower permission.
- Bad, because it refuses the input use the owner allowed for a community site, including grounded
  answers that quote the forum.

### Allow all three uses

- Good, because it is the simplest permissive signal.
- Bad, because it grants training, which the owner did not allow.

### Disallow all three uses

- Good, because it reserves every automated use.
- Bad, because it also asks crawlers not to index the public community, which would hide the forum
  from ordinary search.

## Links

- [Content Signals](https://contentsignals.org/) — directive vocabulary and the three categories,
  reviewed 2026-11-12.
- [Forum experience](../product/forum.md) — the shipped robots.txt behavior.
