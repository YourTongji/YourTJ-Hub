# Render extra blank lines between blocks as empty lines

## Status

Accepted
Class: feature

## Context and Problem Statement

Writers leave space by pressing Enter several times (#1584, DQ-1 of #1630). The Web editor (Vditor
wysiwyg) already stores each empty paragraph as one more blank Markdown line: k empty paragraphs
become k+1 blank lines. CommonMark renderers (Goldmark on the server, markdown-it in Web previews,
markdown_widget in the app) collapse any run of blank lines into one paragraph break, so the space
disappeared after publishing. Lute also collapsed the run when a draft or post was reopened, and the
app long-form editor (markdown_quill) wrote each empty line as two newlines and collapsed them on
load. A second edit therefore lost the space the writer had made, and Web and app editors disagreed.

## Decision Drivers

- What a writer sees in the editor survives publishing, re-editing and a different device.
- Markdown stays the only stored source; no stored HTML, `<br>`, NBSP or zero-width placeholders,
  no data migration and no API or contract change.
- Block structure never changes: code, math, lists and indented continuations keep CommonMark meaning.
- One rule, implemented identically by the three renderers and checked by a shared fixture.
- A run of blank lines cannot stretch a feed card or thread without bound.

## Considered Options

1. Render each blank line after the first in a run between top-level blocks as one empty-line
   spacer, capped per run, and keep empty paragraphs when the editors reload Markdown.
2. Keep CommonMark rendering and make every Enter a paragraph with larger automatic spacing
   (option A of #1630).
3. Store explicit empty-block syntax or a sidecar AST (option C of #1630).
4. Convert soft line breaks to hard breaks globally.

## Decision Outcome

Choose option 1, at most three spacers per run (`MAX_RENDERED_BLANK_LINES`, `MaxRenderedBlankLines`,
`maxRenderedBlankLines`).

- **Rule.** A run of r blank lines (lines of only spaces or tabs) between two top-level blocks
  renders `min(r - 1, 3)` spacers. It applies only before an unindented line, never inside fenced
  code or an unclosed fence, never inside a math segment and never before an indented continuation.
  Math segments are the ones the shared math scanner finds (`$$…$$`, `\[…\]`, math environments,
  including display math opened on a text line), so blank lines inside math stay part of the formula.
  Blank lines between items of the same list are not marked; a different bullet character (`-`,
  `+`, `*`) or ordered delimiter (`.`, `)`), or a switch between bullet and ordered items, starts a
  new list as in CommonMark and is marked. Leading and trailing blank lines are ignored.
- **Mechanism.** Each renderer inserts a marker paragraph (a string absent from the source) at every
  spacer, renders with its usual parser, then swaps the marker paragraph for
  `<p class="gf-blank-line" aria-hidden="true"></p>`. Because the marker sits between blocks that
  are already separated by blank lines, the parse tree is otherwise unchanged.
  - Server: `apps/gooseforum/app/bundles/markdown2html/blank_lines.go`, applied by
    `PostMarkdownToHTML` and `PostMarkdownToHTMLWithMentions`; `GetPostVersion` is 10 so cached HTML
    re-renders lazily. `MarkdownToHTML` (site pages) and course reviews, which collapse blank runs
    before rendering, are unchanged.
  - Web preview: `resource/src/runtime/blank-lines.ts`, used by `renderMarkdownPreview`.
  - App reader and publishing preview: `core/lib/src/markdown/blank_lines.dart`; `GfMarkdownView`
    renders the marker paragraph as an empty line of body text.
  - Styling: `.gf-blank-line { min-height: 1lh }` in `resource/src/styles/prose.css`, so a spacer
    is one line tall plus the profile's paragraph spacing, the same as an empty editor paragraph.
- **Editors.** `VditorOfficial.vue` wraps the instance's Lute `Md2VditorDOM`, `Md2VditorIRDOM` and
  `SpinVditorSVDOM` to restore every empty paragraph (no cap) and `Md2HTML` to render capped
  spacers in the built-in preview. `MarkdownConverter` restores empty Quill lines on load and saves
  k empty lines as k+1 blank lines. The app short-form `TextField` already stores text as typed.
- **Shared fixture.** `packages/api-contract/fixtures/markdown-blank-lines.json` is read by Go,
  Vitest and Flutter tests.

Consequences:

- Existing posts, replies and wiki pages whose Markdown contains a run of two or more blank lines
  between blocks gain spacers when next rendered. Stored Markdown is not modified; rolling back is
  reverting the code and bumping `GetPostVersion` again.
- Lists still collapse blank lines between items, matching CommonMark and Lute's list handling.
- Spacers are hidden from assistive technology and contribute no text to selection or excerpts.

## Pros and Cons of the Options

### Option 1: capped spacers for extra blank lines

- Good, because the stored Markdown that editors already produce now renders as written.
- Good, because the change is render-only, versioned and reversible.
- Bad, because rendering departs from CommonMark for this one case, and Markdown authored elsewhere
  with incidental double blank lines shows extra space.

### Option 2: automatic paragraph spacing only

- Good, because it keeps CommonMark rendering.
- Bad, because the space a writer deliberately made still disappears after publishing and re-editing.

### Option 3: explicit empty-block storage

- Good, because the intent is unambiguous.
- Bad, because it needs a new syntax or data contract for every renderer, editor, API consumer and
  old client.

### Option 4: global hard breaks

- Bad, because it rewrites the meaning of every existing soft line break and does not address blank
  lines at all.

## Links

- Issues #1584 and #1630 (DQ-1).
- [0119](0119-wysiwyg-sticker-editing.md) for the editor round-trip approach.
- `apps/gooseforum/docs/architecture/markdown-rendering.md`.
