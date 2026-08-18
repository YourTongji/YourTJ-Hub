# Markdown Rendering Direction

GooseForum will keep Markdown as the canonical authoring format and use a
dual-implementation rendering strategy:

- Server rendering is authoritative for saved and public content.
- Client rendering is used for editor preview and progressive enhancement.
- The two implementations are kept aligned by a shared test specification, not
  by embedding a JavaScript runtime in the Go server.

This keeps the runtime simple while supporting client-side code highlighting
and leaving room for richer rendering such as diagrams and math.

## Goals

- Store user-authored Markdown as the source of truth.
- Render final topic and post HTML on the server with `goldmark`.
- Render editor preview on the client with `markdown-it`.
- Define a Markdown compatibility test suite that both renderers must satisfy.
- Add advanced visual features as client-side enhancements when they are needed.
- Do not render raw HTML from user Markdown as a supported contract.

## Non-Goals

- Do not store rich-text HTML as the canonical content format.
- Do not add an editor runtime to the global site bundle or to server-side
  rendering. Topic publishing loads Vditor only with the publish-page chunk.
- Do not embed Node, `goja`, or another JavaScript runtime in the Go server for
  normal Markdown rendering.
- Do not promise full compatibility with Discourse Markdown extensions.

## Rendering Model

```text
Vditor WYSIWYG
        |
        | emits Markdown
        v
Editor raw Markdown

Other composer raw Markdown
        |
        | optional client preview
        v
markdown-it preview HTML

Editor raw Markdown
        |
        | submit / save
        v
goldmark server HTML
        |
        | persisted rendered_html / page payload
        v
public topic and post display
```

The server result is the trusted result. Client preview should be close enough
for writing, but it is not the security or storage boundary.

## Markdown Compatibility Spec

The project should maintain a small corpus of Markdown fixtures under
`testdata/markdown-compat/`. Each fixture is a themed Markdown file with a
matching assertion file:

```text
testdata/markdown-compat/
  blocks.md
  blocks.json
  inline.md
  inline.json
```

Each fixture contains:

- A Markdown input.
- Expected semantic behavior.
- Allowed HTML differences when exact HTML is not important.

The suite should cover common forum content first:

- headings and generated heading IDs
- paragraphs and hard/soft line breaks
- emphasis, strong, strikethrough, inline code
- fenced code blocks
- ordered and unordered lists
- nested lists
- task lists
- blockquotes
- links and autolinks
- images
- tables
- raw HTML handling, currently treated as unsupported user markup
- escaped Markdown characters
- mixed Chinese and English punctuation

Exact HTML should only be asserted where the public contract depends on it. For
example, task list checkbox attributes and heading anchors matter; incidental
attribute order does not.

Current checks:

```bash
go test ./app/http/controllers/markdown2html
cd resource && pnpm exec vitest run test/markdown-compat.test.ts
```

## Server Responsibilities

Server rendering owns:

- final topic and post HTML
- security-sensitive normalization
- link attributes such as `target` and `rel`
- image attributes such as lazy loading and async decoding
- rendered version tracking and rebuild migrations
- SEO and no-JavaScript output

The current server renderer is `goldmark` in
`app/http/controllers/markdown2html`.

## Client Responsibilities

Client rendering owns:

- editor preview
- topic authoring through Vditor WYSIWYG, with Markdown emitted as the stored value
- lightweight authoring helpers for other composer surfaces
- optional post-render enhancements that do not change stored Markdown

The topic publish page loads Vditor only when its route is opened. It uses a
reduced toolbar, disables editor cache and optional preview renderers, and
packages only the required parser, icon, and locale assets into the embedded
frontend output. The toolbar includes image selection, while project-owned
validation, compression, upload, paste, and drag-and-drop handling remain in
the publish page. A separate Markdown preview is not exposed because Vditor
already provides WYSIWYG authoring. It does not use an external CDN. Vditor
HTML is editor state only; the submitted value remains Markdown.
Dark/light chrome uses Vditor `setTheme`; body text color follows via a local
`content-theme` stylesheet swap plus a CSS binding of `.vditor-reset` to
`--textarea-text-color` (Vditor's base CSS hardcodes light text color, and an
empty `preview.theme.path` would otherwise skip content-theme loading).

The current client preview renderer is centralized in
`resource/src/runtime/markdown.ts`. Pages should call this helper instead of
creating local `MarkdownIt` instances.

Current client enhancement:

- Fenced code blocks with an explicit, recognized language are highlighted
  with the Highlight.js common-language build.
- Saved topic/post HTML and Markdown editor previews use the same Vue directive
  and highlighter adapter. The adapter is loaded dynamically only after a
  `language-*` code block is detected.
- Language auto-detection is disabled. Unknown and unlabelled languages remain
  escaped plain-text code blocks, and a load failure leaves the server or
  Markdown-it output unchanged.
- Inline `$...$`/`\(...\)` and block `$$...$$`/`\[...\]`/`\begin{...}` math is rendered with KaTeX by the
  `v-math-render` directive. The KaTeX chunk (JS, CSS and fonts) is loaded
  lazily only after a math marker is detected outside code blocks, and ships
  inside the single binary via the go:embed asset pipeline. Detection uses a
  brace-balanced scan with MathJax-style inline guards (delimiters must not sit
  next to whitespace, inline math cannot span lines) so prices and shell
  variables stay literal; render failures leave the original text unchanged.
- KaTeX renders from text input only — `trust`/raw-HTML output is never
  enabled — so math content stays escaped and cannot execute script.

Potential client enhancements:

- Mermaid for diagrams

These should be loaded only on pages that need them, preferably by detecting
matching code fences or inline markers. They should not become part of the base
forum bundle until real usage justifies it.

Client enhancement libraries should decorate already-rendered content. They
should not change the canonical Markdown storage format.

## Feature Policy

Add Markdown features in this order:

1. Add or update compatibility fixtures.
2. Make `goldmark` pass the server expectations.
3. Make `markdown-it` preview match the semantic behavior.
4. Add client enhancement only if static HTML is not enough.
5. Document any accepted preview/final differences.

This avoids turning editor behavior into the content contract by accident.

## Current Decision

GooseForum should continue with dual implementation:

- `goldmark` for authoritative server rendering.
- `markdown-it` for client preview.
- fixture-based compatibility tests to keep them aligned.
- Highlight.js as a client-only, explicit-language code enhancement.
- Vditor WYSIWYG as a publish-page-only authoring UI that emits Markdown.
- client-only optional renderers for diagrams and math.

The `goja` experiment is useful as a reference, but it should not replace the
current approach unless the Markdown dialect becomes too complex to keep aligned
with tests.

Math protection: the server and preview renderers now replace math segments
with unique placeholders before Markdown parsing and restore them in the
rendered HTML. This keeps typographer, emphasis parsing, and paragraph splitting
from corrupting `$...$`/`$$...$$`. Restored math segments are HTML-escaped
(`html.EscapeString` in Go, the equivalent on the markdown-it preview side) so
raw HTML inside `$...$` cannot bypass the renderer's raw-HTML filtering. The
client enhancer still scans rendered text nodes, so historical HTML created
before the rendered-version bump may keep a split inline expression literal
until posts are re-rendered.
