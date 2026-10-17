# Platform-specific link preview adapters

## Status

Accepted
Class: feature

## Context and Problem Statement

Issue #1201 asks readers to see what an external media link is — a Bilibili video, an Apple Music
song or album — instead of a bare URL. The forum already has a bounded link-preview pipeline
(issues #729, #731–#735): `/api/link-previews/resolve` classifies first-party links, fetches
third-party pages through the SSRF-safe `safefetch`, exposes a fixed metadata whitelist, caches
and singleflights results, and lets Web and Flutter render at most five cards per body. It does not
execute third-party HTML or JavaScript, and it has no per-platform resource semantics: no resource
type, no platform identifier, no uploader or artist.

The question is how to add platform-aware cards without turning the forum into a scraping proxy, an
embed framework, or a place where page-supplied URLs and remote scripts get executed.

## Decision Drivers

- Reuse the existing resolve contract, cache, singleflight, quota and outbound-link guard; add no
  parallel fetch path and no new database table.
- Keep reading raw Markdown the only editing source of truth; enhancement stays a rendering concern.
- Recognize a platform only by strict host, path shape and resource id, so lookalike hosts and
  unsupported resource types never produce a platform label.
- Never trust a URL supplied by a user or by a remote page as a metadata endpoint.
- Keep the four status words honest: unavailable metadata degrades to the generic card and then to
  the original link, and missing optional fields are omitted rather than synthesized.
- Let clients localize user-visible labels; the server sends identifiers, not localized copy.
- Do not expand the current "no third-party HTML/JS" boundary in this change.

## Considered Options

- Strict per-platform adapters on the existing pipeline, with code-owned metadata endpoints.
- Generic oEmbed discovery: follow a page's `<link rel="alternate" type="application/json+oembed">`
  or provider list, and optionally render the returned `html`.
- Undocumented per-platform JSON APIs (for example Bilibili's web-interface endpoint) as the only
  metadata source.
- A universal embed framework that loads official players in the reader's browser.

## Decision Outcome

Chosen option: "Strict per-platform adapters on the existing pipeline, with code-owned metadata
endpoints."

Recognition is a pure function of the URL: canonical Bilibili hosts with a `/video/BV…` or
`/video/av…` path, and `music.apple.com` with a `/{storefront}/{song|album}/{slug}/{id}` path. An
album link carrying `?i=` is the song, not the album. Short links are not recognized directly: a
`safefetch` redirect lands on the real address, and only that validated final URL is recognized, so
the card keeps showing the URL the author pasted. Recognition only adds fields; it never changes
which fetch or safety policy applies.

Metadata sources differ per platform and are both bounded by the same fetcher:

- Bilibili reads the same bounded page fetch the generic card uses and takes only whitelisted meta
  values: `og:title` (minus the platform's `_哔哩哔哩_bilibili` suffix), `<meta name="author">` for
  the uploader, and `og:image` restricted to the platform's own CDN. The platform's SEO description
  keeps only the descriptive part, dropping the trailing view-count and related-video block.
- Apple Music uses one code-owned lookup endpoint, `https://itunes.apple.com/lookup`, with the id and
  storefront taken from the recognized URL. The response is parsed into a whitelisted struct
  (`trackName`/`collectionName`, `artistName`, `artworkUrl100`), the record must match the link's
  resource type, the final host must still be the endpoint host, and artwork is upgraded to 600×600
  only when it stays on the platform's own CDN. This is a fixed, documented endpoint, not a
  discovery URL, and it is fetched through the same URL/DNS/redirect/size/timeout policy as any page,
  with HTML responses rejected by that policy.

A platform card is produced only when the page or endpoint actually supplied a name; otherwise the
generic card stands. Empty optional fields (author, cover) are omitted. Cache keys for recognized
platform URLs lowercase the host, drop a whitelist of share-tracking parameters, and keep
resource-defining parameters (`p`, `t`, `i`) and the Apple storefront; every other URL keeps its
existing cache key unchanged.

The response adds `platform`, `resourceType` and `author` to the existing preview shape. Clients
render the platform name, resource-type badge and author label from their own locale; older clients
ignore the new fields and still show a correct generic card.

Loading official players stays out of scope and remains `Decision needed` with the maintainer: the
current boundary is that preview cards never request a player, execute remote HTML/JS, or start
media playback.

## Pros and Cons of the Options

### Strict per-platform adapters on the existing pipeline

- Good, because one pipeline keeps cache, singleflight, quota, SSRF policy and the outbound-link
  guard authoritative; the platform layer cannot bypass them.
- Good, because a fixed endpoint and a whitelisted struct are testable with fixtures, with no live
  network in tests.
- Good, because failure has a defined ladder — platform card, generic card, original link — so a
  platform outage never blocks posting or reading.
- Bad, because each new platform needs its own recognition, metadata mapping and fixtures instead of
  one generic mechanism.
- Bad, because a page-based platform (Bilibili) inherits the platform-side anti-bot behavior: a
  datacenter source may receive a challenge page with no metadata, and the card then falls back to
  the generic card or the original link.

### Generic oEmbed discovery and rendered embeds

- Good, because one mechanism could cover many providers that publish oEmbed.
- Bad, because the discovery URL comes from the fetched page, which makes remote data choose our
  outbound request target; the issue explicitly rejects that.
- Bad, because rendering the returned `html` (or an `iframe`) requires executing third-party markup
  in the reader's session and would extend the current no-third-party-HTML/JS boundary.
- Bad, because protocol availability does not imply per-site coverage or stable fields.

### Undocumented platform JSON APIs

- Good, because a JSON endpoint can return exact structured fields without HTML parsing.
- Bad, because an undocumented interface can change or disappear without notice and is not an
  approved authorization basis; it must not be the only path to a supported card.
- Bad, because it invites requests that bypass a platform's own public-page terms.

### Universal embed framework with browser-loaded players

- Good, because a reader could watch or listen without leaving the forum.
- Bad, because it changes the script-execution boundary, needs per-platform consent, CSP and
  referrer work, and cannot be verified in this change; it needs its own decision with the
  maintainer before implementation.

## Links

- [Issue #1201](https://github.com/YourTongji/YourTJ-Hub/issues/1201)
- [Bilibili out-of-site player documentation](https://player.bilibili.com/)
- [iTunes Lookup API](https://developer.apple.com/library/archive/documentation/AudioVideo/Conceptual/iTuneSearchAPI/LookupExamples.html)
- [oEmbed specification](https://oembed.com/), [Open Graph protocol](https://ogp.me/)
- Product rule: [Links, Markdown and stickers](../product/forum.md#links-markdown-and-stickers)
