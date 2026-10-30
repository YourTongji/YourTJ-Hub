# Show stickers as images while editing, on Web and app alike

## Status

Accepted
Class: feature

## Context and Problem Statement

Sticker content is stored and published as plain `[:sticker:name:]` tokens (MADR 0030); readers
see the image. Editors, however, showed the raw token. The app also deleted it one character at a
time ([0106](0106-editor-sticker-token-atomic-delete.md) covered only the Web), and compensated with
a separate bounded "live preview" under the input. Writers therefore saw a different layout from what
they published, and Web and app writers saw different layouts from each other. Hand-typed tokens had
no completion, and the Web picker's pack tabs and topic chips overflowed with no horizontal affordance
for mouse users.

## Decision Drivers

- What a writer sees in every reply, short-form and long-form editor matches the published layout,
  identically on Web and app: one square image 2.5 × the surrounding font size, centred on the line.
- The stored value stays the plain token; no server, contract or storage change.
- One delete removes one sticker everywhere; the caret never rests inside one.
- Typing `[:sticker:` offers matching stickers; a complete known token becomes the image.
- No new dependency; each editor keeps its native input, IME and undo behaviour.

## Considered Options

1. Decorate tokens in place inside each existing editor (Vditor DOM decoration, a Flutter text
   controller that draws images, Quill sticker embeds) and serialize back to the token.
2. Keep raw tokens in the editors and show a separate live preview (the previous app behaviour).
3. Replace the editors with one shared custom rich-text model.

## Decision Outcome

Choose option 1.

- Web (Vditor wysiwyg; publish, reply, quick publish, course review): `runtime/sticker-inline-editor.ts`
  wraps each resolved token outside code, links and raw HTML in a non-editable
  `span.gf-sticker-inline` that keeps the token as its text, so Lute serializes the same Markdown.
  A MutationObserver re-decorates after Vditor re-renders a block, before paint. The atomic delete
  from 0106 expands to whole decorated spans. Lute's emoji shortcode conversion is disabled
  (`SetEmoji(false)`), because it rewrote `:name:` inside tokens; the server never expanded shortcodes.
- App plain-text composers (reply, post edit, simple publish, chat): `StickerTextEditingController`
  draws each resolved token as a `WidgetSpan` image followed by the remaining token characters at zero
  width, so text offsets still map onto the layout. Deletions that touch a token or a half-typed
  `[:sticker:` fragment expand to the whole range. Carets snap out of tokens and selections widen to
  whole tokens. Chat uses plain-text rules and posts use Markdown rules, matching each destination.
  The separate draft preview is removed.
- App rich editor (long-form publish, course review): stickers are Quill inline embeds; Markdown
  loading converts tokens outside code, links and math into embeds, saving writes the token back, and
  a watcher converts a token as soon as its closing `]` is typed. Unresolved names render as the token
  text.
- Suggestions: a `[:sticker:` fragment at the caret lists the viewer's stickers, then official ones,
  then subscribed packs, ranked exact > prefix > substring by name and label (shared rules in
  `sticker-token.ts` and `sticker_token.dart`). The Web shows a keyboard listbox next to the caret
  (arrows, Enter/Tab, Escape); the app shows a strip above the input. Picking replaces the fragment.
- Sizing: published Web stickers become a 2.5em square (previously 2.5em tall with a free width), the
  same box the app uses.
- Web picker strips: pack tabs and topic chips scroll horizontally with the mouse wheel, fade at an
  edge that has more content, show paging arrows on hover and keep the active tab in view, following
  Telegram Web and Bilibili.

The Web private-message input stays a plain `<textarea>` with atomic deletion only; a textarea cannot
draw images, and replacing it is a separate change.

## Pros and Cons of the Options

### decorate in place (selected)

- Good: the stored token, renderers and API are unchanged; editing shows the published layout on both
  platforms; atomic deletion and caret rules come from the same model; native IME and undo stay.
- Bad: three host-specific adapters to maintain; the Vditor decoration depends on Lute keeping span text
  and on re-decorating after each block re-render; undoing inside a just-typed fragment can remove the
  whole fragment.

### raw tokens plus preview

- Good: smallest code.
- Bad: the editor still shows markup and a second copy of the content; layout differs from the result
  and between platforms; it is what the issue asks to replace.

### one custom rich-text model

- Good: a single behaviour definition for both platforms.
- Bad: rewrites every editor, loses upstream Vditor/Quill features and needs new serialization for
  paste, drafts and previews; far larger than the problem.

## Links

- [MADR 0030: global sticker library](0030-global-sticker-library.md)
- [MADR 0106: atomic sticker deletion in the Web editor](0106-editor-sticker-token-atomic-delete.md)
- [MADR 0089: unified Twemoji rendering](0089-unified-twemoji-rendering.md)
- [Forum product: links, Markdown and stickers](../product/forum.md#links-markdown-and-stickers)
- [Mobile sticker library](../product/mobile-experience.md#sticker-library)
