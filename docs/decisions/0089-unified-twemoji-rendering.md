# Unified Twemoji rendering across Web and Flutter

## Status

Accepted
Class: architecture

## Context and Problem Statement

Web browsers, iOS, and Android ship different Emoji fonts and can split the same
Unicode sequence into different glyphs. The forum renders user text through Vue
templates, Markdown/HTML, and Flutter's shared UI and Markdown components. A
consistent result must preserve raw Unicode, Markdown structure, selection and
offline display without relying on a runtime CDN.

## Decision Drivers

- Show the same artwork for the same Unicode sequence on Web, iOS, and Android.
- Match complete Unicode Emoji sequences rather than individual code points.
- Keep user content, search terms, and API/database values as original Unicode.
- Package graphics locally and keep Flutter rendering available offline.
- Preserve code, links, Markdown, text scaling, line wrapping, accessibility, and
  text selection/copy.
- Keep the supported source version and licenses reviewable.

## Considered Options

1. Bundle a pinned Twemoji PNG release with one generated sequence map and
   render it through shared Web and Flutter text components.
2. Keep native platform Emoji fonts and patch individual rendering problems.
3. Use a custom Emoji font or runtime-hosted graphics.

## Decision Outcome

Choose option 1 and pin [jdecked/twemoji 17.0.3](https://github.com/jdecked/twemoji/releases/tag/v17.0.3).
Use the release's official 72×72 PNG files for both platforms and generate the
Web manifest and Dart map from those same asset filenames. Exclude the upstream
legacy private-use glyph because it is not a Unicode Emoji sequence. Keep the
source sequence in the text model; replace only its visual glyph with an inline
image. A missing map entry or image falls back to the source text.

Vue's compiler transform handles template text and interpolations; a Vue
directive handles text nodes under rendered rich HTML while skipping code and
editable content. Flutter's shared `GfText`/`GfSelectableText` transforms rich
spans; Markdown supplies those same components through its rich-text builder.
Selectable copy maps an image placeholder back to the complete source sequence.
Text inputs remain native during editing.

The Web bundle serves Twemoji files from the app's same-origin static assets.
Flutter packages the files in `ui_kit`. Code is MIT licensed and graphics are
CC BY 4.0; both license texts and attribution are kept with the assets.

## Pros and Cons of the Options

- Option 1 gives the three app platforms one versioned image set and mapping,
  works offline, and makes upgrades reviewable. It adds roughly 4.25 MB of PNG
  source assets before platform packaging and requires inline-span/image
  selection handling.
- Option 2 avoids bundled assets but cannot make operating-system Emoji
  renderers match and leaves complex sequences dependent on each OS release.
- Option 3 either adds font-format and cross-platform fallback complexity or
  introduces a runtime network dependency; neither fits the existing image
  asset pipeline as directly as the pinned PNG release.

## Links

- [Product rendering behavior](../product/emoji-rendering.md)
- [Twemoji 17.0.3 release](https://github.com/jdecked/twemoji/releases/tag/v17.0.3)
- [Twemoji license and attribution](https://github.com/jdecked/twemoji/blob/v17.0.3/LICENSE)
- [Twemoji package metadata](https://github.com/jdecked/twemoji/blob/v17.0.3/package.json)
