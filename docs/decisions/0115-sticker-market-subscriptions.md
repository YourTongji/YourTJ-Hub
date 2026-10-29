# Curated sticker market with stored assets and pack subscriptions

## Status

Accepted
Class: feature

## Context and Problem Statement

Issue #1429 asks for a sticker market: members browse themed packs from an external catalog
(`s.pwsh.us.kg`, a static JSON index of topics, pages and packs with image URLs), add whole packs to
their picker, and preview stickers by holding and sliding across the grid. Rendering must not depend
on the upstream host being reachable, readers' devices must not request third-party hosts, and tokens
stored in content must keep resolving if upstream changes or disappears.

## Decision Drivers

- Rendering never depends on the upstream host; content tokens stay stable.
- Administrators decide what members can see; third-party content is never published automatically.
- Upstream fetches go through the existing SSRF-safe client with size, type and dimension limits.
- Bounded storage and per-account state; identical images are stored once.
- One contract for Web, admin and Flutter.

## Considered Options

1. Administrators sync the catalog and import chosen packs; the server downloads and stores every
   image (content-addressed) before the pack can be published; members subscribe to published packs.
2. Lazy materialization: publish catalog entries directly and download an image the first time a
   member adds or sends it.
3. Proxy or hot-link upstream images without storing them.

## Decision Outcome

Choose option 1.

- **Catalog.** The admin sync action (`sticker-market-sync`) reads the upstream topic and page index
  into `sticker_market_topics` and `sticker_market_packs`. Synced packs start hidden. The upstream base URL
  and image hosts come from the `sticker_market.base_url` and `sticker_market.image_hosts`
  preferences; `sticker_market.enabled` turns the market off. Requests use `safefetch` with a 4 MiB
  body limit; index redirects must stay on the HTTPS base host and image redirects on allowed image
  hosts. The market client dials validated IPv4 addresses first and falls back to the next validated
  address, because image CDNs whose IPv6 path connects but stalls otherwise time out every download.
  Sync also keeps each pack's upstream cover (an allow-listed HTTPS image or a short text cover);
  only the admin catalog shows it, from the administrator's browser, until a local cover exists.
- **Import.** Importing a pack queues one `sticker-market-import` task. The worker downloads at most
  500 images per pack, checks the actual bytes against the upload image policy and a 4096 px side
  limit, and stores each image once as `stickers/market/<sha256>.<ext>`. HEIC and TIFF stay
  rejected as `unsupported`, as for member uploads ([0113](0113-upload-header-authority.md)).
- **AVIF.** Upstream packs often ship AVIF, which neither the upload policy nor Flutter accepts, so
  the import transcodes it once: a still becomes PNG (full alpha) and an animation becomes GIF, the
  format the market already serves for animated packs. Before decoding, the container's sample
  count bounds the work (at most 240 frames and 512 × 512 × 120 decoded pixels for animations);
  one transcode runs at a time. Animations use one shared median-cut palette, merge identical
  frames, keep the source timing and treat alpha below one half as transparent; a result above
  2 MiB is redone at a 384 px, then 256 px long side, and anything still above the 4 MiB storage
  limit fails as `tooLarge`. The decoder is `github.com/gen2brain/avif` (libavif with dav1d,
  compiled to WASM and run by wazero); release builds use the `nodynamic` tag so it never loads a
  system libavif. Each image becomes a shared,
  approved sticker in the `market` group with the stable token name `m_` + 40 hex characters; the
  official directory and admin library do not list it. Failed items keep a reason and can be retried;
  re-importing also picks up stickers that upstream added later. Each run records its item total and
  updates the ready and failed counts after every item, so the admin list shows live progress; a
  retry clears the previous failures first. When every item fails for one reason, that reason becomes
  the pack's error. A pack can be published once it holds at least one imported sticker.
- **Publication.** Administrators publish, hide and order packs, and disable single stickers. Hidden
  packs disappear from browsing and pickers, while their stickers keep resolving in existing content;
  a disabled sticker stops resolving, like a disabled official sticker.
- **Subscriptions.** Anyone can browse published packs; adding one requires sign-in. An account holds
  at most 50 packs in `user_sticker_packs`, separate from the 200-item personal library. Account
  closure removes subscriptions.
- **Caching.** `/file/img/*` serves `stickers/market/` objects with `public, max-age=86400`; they are
  public catalog assets with content-addressed names. Other files keep the policy in
  [object storage](../operations/object-storage.md#image-read-authorization-and-cache-policy).
- **Preview.** Holding a sticker in a picker or pack grid shows a larger preview; sliding switches the
  preview and releasing closes it. Web also previews on pointer hover and keyboard focus.

Issue #1429 suggests materializing images when a member first adds a pack. This record materializes
at import time instead, so a member's request never triggers an upstream fetch and a pack's
availability is known before it is published.

## Pros and Cons of the Options

### admin import with stored assets (selected)

- Good: members never wait on or reach the upstream host; tokens and bytes survive upstream changes;
  administrators review packs before publication; duplicate images share one object.
- Bad: storage grows with every imported pack even if nobody adds it; upstream updates need a manual
  re-import.

### lazy materialization on first use

- Good: stores only packs members actually add.
- Bad: the first member's request blocks on, and fails with, the upstream host; publication happens
  before any image is checked; concurrent first uses need extra coordination.

### AVIF output format

- Selected — **GIF/PNG with the standard library:** no new encoder, no change to how the rest of
  the server decodes images, and no deployment change. Costs size and binary alpha (see Open
  questions).
- Rejected — **animated WebP via `gen2brain/webp` or `vpx`:** both register a WebP decoder in Go's
  global `image` registry under the same signature as `golang.org/x/image/webp`. Package
  initialization order would then silently replace the decoder used by the upload policy, feed
  variants and decoration artwork, and `gen2brain/webp` loads a system libwebp when one is present.
- Rejected — **ffmpeg or libavif/libwebp command-line tools in the image:** a deployment dependency
  for every self-hosted instance, larger images, and uncertain alpha handling for AVIF sequences.
- Rejected — **storing AVIF as is:** Flutter cannot render it.

### proxy or hot-link upstream images

- Good: no storage.
- Bad: content breaks when upstream changes; hot-linking exposes readers to a third-party host;
  proxying still makes every render depend on upstream availability.

## Links

- Issue #1429
- [0030: global sticker library](0030-global-sticker-library.md)
- [0038: personal sticker library](0038-personal-sticker-library.md)
- [0114: sticker rename aliases](0114-sticker-rename-aliases.md)
- [gen2brain/avif v0.6.0 API reference](https://pkg.go.dev/github.com/gen2brain/avif@v0.6.0)
- [Go specification: package initialization order](https://go.dev/ref/spec#Package_initialization)
- [Mobile sticker library](../product/mobile-experience.md#sticker-library)

## Open questions

- Copyright and licensing for third-party packs is undecided; administrators publish packs at their
  own discretion.
- Reachability of the upstream host from production networks, including mainland China, is
  unverified.
- Transcoded GIFs are much larger than their AVIF sources (about 1.7 MB per 512 px Telegram
  sticker versus about 100 KB) and lose soft alpha edges; animated WebP would be smaller but is
  blocked by the decoder-registration issue described above.
- The 50-pack, 500-item and 1-day cache limits are conservative defaults; members cannot reorder
  their packs yet.
