# Store a relative image width as a Markdown destination fragment

## Status

Accepted
Class: feature

## Context and Problem Statement

Writers asked for a quick way to make an image smaller than the column, or to place two or three
images side by side, in the Web editor, and for the result to look the same after publishing, on a
phone and in the app (#1630). Markdown has no width syntax. Images were rendered at their natural
size, capped by the column and a height limit, so the only workaround was to edit image files.

## Decision Drivers

- Markdown stays the only stored source: no HTML, no new API field, no migration.
- Content written with a width must still render correctly everywhere that ignores it (old app
  builds, feeds, RSS, other Markdown tools); the image request must not change.
- One rule, implemented identically by the Web reader, the Web editor and the app, checked by a
  shared fixture.
- A width must stay readable on a narrow phone and must not be an arbitrary pixel value that only
  suits the author's screen.

## Considered Options

1. A URL fragment on the image destination, `![alt](/file/img/a.webp#w=50)`, limited to presets.
2. Inline HTML `<img width="50%">`.
3. A Markdown attribute extension such as `![alt](a.webp){width=50%}`.
4. A query parameter (`?w=50`) handled by the file server.

## Decision Outcome

Choose option 1.

- **Rule.** A destination ending in `#w=<preset>` with preset 25, 33, 50, 66, 75 or 100 renders at
  that percentage of the content width, never narrower than 160 CSS/logical pixels (or the content
  width when that is narrower). A sized image has no height limit, because the width is the writer's
  explicit choice. Other values and other fragments (for example an SVG sprite `#icon`) are ignored
  and left untouched by the editor. A paragraph that contains only images (and whitespace) with
  adjacent sized images becomes a centred, wrapping row with an 8px gutter, so two `50%` or three
  `33%` images sit side by side and wrap when the column is too narrow.
- **Shared contract.** `packages/api-contract/fixtures/markdown-image-width.json` lists the presets,
  the minimum and the parse, set and snap cases. Web `resource/src/runtime/image-width.ts` and app
  `core/lib/src/markdown/image_width.dart` implement them and read the fixture in tests.
- **Web reader.** `resource/src/styles/prose.css` matches the fragment with attribute selectors; the
  server keeps `src` verbatim (`PostMarkdownToHTML` only adds loading attributes), so no rendering
  version bump is needed and existing HTML caches stay valid.
- **Web editor.** Clicking an image in the wysiwyg or instant-rendering mode selects it and shows a
  floating image toolbar (original, 33%, 50%, 75%, 100%, alt text, preview, delete) and side handles;
  dragging snaps to the nearest preset (90% or more becomes 100%). The editor applies the same CSS,
  so what the writer sees is what readers see.
- **App reader.** `GfMarkdownView` renders image-only paragraphs with a sized image as the same
  centred wrapping row; mixed text-and-image paragraphs keep the inline layout.
- **Server.** File usage tracking reads the URL path, so the fragment never creates a second file
  reference; `ExtractImageURLs` keeps the destination as written.

Consequences:

- Readers that do not know the rule show the image at its natural size, as before.
- An image inside a mixed text paragraph is sized on the Web but not in the app.
- The app editors do not offer a width control yet; they keep the fragment they load.

## Pros and Cons of the Options

### Option 1: destination fragment with presets

- Good, because the fragment is never sent to the file server and every Markdown renderer keeps it.
- Good, because presets keep layouts consistent across screens and can be matched with plain CSS.
- Bad, because the width is not visible as such to tools that do not know the convention.

### Option 2: inline HTML

- Bad, because post HTML is sanitized and the app renderer does not lay out raw HTML images.

### Option 3: attribute extension

- Bad, because Goldmark, markdown-it, Lute and markdown_widget would each need a parser extension,
  and every other renderer would show the braces as text.

### Option 4: query parameter

- Bad, because it changes the requested URL and caches, and ties layout to the file server.

## Links

- Issue #1630.
- [0133](0133-visible-blank-lines.md) for the shared-fixture approach.
- `apps/gooseforum/docs/architecture/markdown-rendering.md`.
