# Emoji rendering

> Doc type: product spec
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-13

## Supported behavior

`Current`: Web and Flutter iOS/Android render in-app Unicode Emoji with the same
Twemoji 17.0.3 artwork and sequence map. Forum posts and replies, Markdown, chat,
notifications, profiles, course reviews, Wiki content, and other app text share
the renderer. Unicode remains the stored and transmitted content; rendering does
not rewrite existing or new content.

The pinned release contributes 4,008 official 72×72 PNGs. A generated sequence
manifest is shared by Web and Flutter; both match complete sequences, including
ZWJ families and professions, variation selectors, skin tones, regional flags,
keycaps, and tag sequences. The longest complete matching sequence wins. A
sequence without a bundled graphic remains the original text. The legacy
private-use glyph in the upstream archive is excluded because it is not Unicode
Emoji.

Web serves the files from the same-origin Vite asset tree. Flutter bundles them
in `ui_kit`, so Emoji graphics work offline. Images use the surrounding text
size and remain inline with line wrapping. Flutter image decoding uses the
shared asset cache; Web does not walk the document globally and only processes
Vue text nodes or the text nodes in Vue-managed rich HTML. Code formatting and
syntax highlighting remain intact while Emoji in code uses Twemoji. Editable
controls retain their native input behavior.

Flutter's `GfText` and `GfSelectableText` are the shared display components.
Markdown and the HTML renderers for Wiki/course reviews use the same span
transformer and preserve inline/fenced code formatting while rendering Emoji.
Selectable copy maps each image placeholder back to its complete original
Unicode sequence, including code. Missing Flutter assets fall back to native text. Web images
retain the original Unicode in their `alt` text and fall back to text on load
failure. Text fields remain native Flutter controls during IME composition;
their post-submit display uses the shared renderer.

## Boundaries

`Partial`: The operating system draws notification center content, native
keyboard UI, status/navigation bars, and some browser or app-shell title chrome.
Those surfaces do not use the in-app text renderer and may use the system Emoji
font. In-page headings, buttons, labels, and in-app notification rows are
rendered by the shared components. A Web page `<title>` and the server's
JavaScript-disabled HTML fallback also remain text-only/system-rendered.

Sticker-library images, uploaded images, and other non-Unicode Emoji resources
keep their existing renderers.

## Assets, attribution, and size

The source is [jdecked/twemoji 17.0.3](https://github.com/jdecked/twemoji/releases/tag/v17.0.3).
The source release's JavaScript is MIT licensed and the graphics are CC BY 4.0.
Both license texts and required attribution are retained with the assets:
[MIT](../../apps/mobile/packages/ui_kit/assets/twemoji-licenses/LICENSE-MIT.txt),
[CC BY 4.0](../../apps/mobile/packages/ui_kit/assets/twemoji-licenses/LICENSE-CC-BY-4.0.txt),
and [attribution](../../apps/mobile/packages/ui_kit/assets/twemoji-licenses/ATTRIBUTION.md).
The asset inventory records the generator command.

The pinned 4,008 PNGs total **4,247,435 bytes (4.051 MiB)** as source files. The generated
Web asset tree and Flutter bundle include the same PNG set. The Android release
workflow records the arm64 APK byte count and size-analysis artifact for each
build; the compressed APK/IPA increment depends on the platform build and is
not equal to the source-file total.

## Implementation and verification references

- [Decision rationale: MADR 0089](../decisions/0089-unified-twemoji-rendering.md)
- [Web renderer tests](../../apps/gooseforum/resource/test/twemoji.test.ts)
- [Flutter UI Kit Twemoji tests](../../apps/mobile/packages/ui_kit/test/gf_twemoji_test.dart)
- [Asset generator](../../apps/mobile/packages/ui_kit/tool/generate_twemoji_manifest.mjs)
- [Mobile CI size artifact](../../.github/workflows/ci-mobile-native.yml)
