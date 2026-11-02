# Delete sticker tokens as atomic blocks in the editor

## Status

Accepted
Class: feature

## Context and Problem Statement

Stickers reach post, reply and topic content as plain-text tokens `[:sticker:name:]`; the token is
expanded only when rendering (server for posts/replies, client for chat bubbles and mobile, MADR
0030). In the Web editor the token is therefore ordinary text, and the default Backspace/Delete
removes it one character at a time, leaving fragments like `[:sticker:na` that no expansion path
recognizes. Issue #1332 reports the resulting visible garbage.

## Decision Drivers

- One keypress removes one sticker; the caret must not stop inside a token.
- Keep the plain-text token storage/rendering contract and require no new dependency.
- Preserve native input behaviour: IME composition, modifier combinations, undo stack.
- One implementation for every Web editor host, without touching the call sites.
- Decide what a corrupt fragment (paste/undo/partial typing) does when deleted.

## Considered Options

1. Intercept Backspace/Delete on the editor root, select the token span and delete it through
   `document.execCommand('delete')`.
2. Render the token as an atomic inline node inside Vditor and serialize it back to text.
3. Prevent fragments at insertion time only (pad the inserted token with whitespace).

## Decision Outcome

Choose option 1. `runtime/sticker-atomic.ts` reads a caret text window scoped to the caret's block,
`site/utils/sticker-token.ts` returns the span to delete, and the capture-phase `keydown` listener in
`VditorOfficial.vue` selects that span and issues the native delete, then prevents the default
keypress. All four hosts that mount the component (publish page, reply composer, quick publish, course
review) share it, so they need no change.

Fragment policy: a `[:sticker:` prefix followed by up to 64 name characters and not yet closed by
`:]` counts as one atom and is deleted whole in both directions, so paste/undo/partial typing cannot
strand half a token (a `:` typed after the name without its `]` is part of the atom); a prefix
immediately followed by a terminator (whitespace, `:`, `[`, `]`) is ordinary text. A caret strictly inside a token deletes the whole token. IME composition, non-collapsed
selections and Ctrl/Meta/Alt combinations are left to the browser; Shift combinations are not
excluded, so Shift+Backspace/Shift+Delete also delete the whole token. The caret window is limited to
the caret's nearest block-level element (paragraph, list item, table cell, heading, code block,
blockquote, figure caption) so two adjacent blocks cannot form a fake token; a caret outside any of
them reads only the editor root's direct child.

The Flutter editor is out of scope: its soft keyboard delivers deletion as `TextEditingValue`
changes, which needs a separate controller/formatter implementation. Issue #1332 tracks it; the
app controller and decorated Web stickers are decided in
[0119](0119-wysiwyg-sticker-editing.md).

## Pros and Cons of the Options

### keydown interception + native delete (selected)

- Good: no change to the storage/rendering contract; no new dependency; insertion sites untouched;
  the selection + native delete path keeps Vditor's input events and modelValue sync (asserted by the
  browser tests) and goes through Vditor's own input path instead of direct DOM removal; the browser
  tests do not assert undo.
- Bad: relies on `document.execCommand`, deprecated but still the shared contenteditable path in this
  repository (mention insertion uses it); cannot make arrow keys skip over a token.

### atomic inline node in Vditor

- Good: the caret can never enter the token, so no key interception is needed.
- Bad: needs a custom wysiwyg node plus text serialization for value, paste, undo and preview paths —
  a large surface for one interaction, and the hosts would diverge from upstream Vditor.

### insertion-time isolation only

- Good: minimal.
- Bad: reduces the chance but fixes nothing; paste, undo and typing reproduce the fragment, and the
  issue asks for deletion behaviour.

## Links

- [MADR 0030: global sticker library](0030-global-sticker-library.md)
- Related open issues: #1242 (token in summaries), #1304 (token text in previews)
- [Forum product: links, Markdown and stickers](../product/forum.md)
