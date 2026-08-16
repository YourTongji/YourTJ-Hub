# Wiki Authoring

> Doc type: product guide
>
> Status: Active
>
> Owner: Wiki maintainers
>
> Last verified: 2026-08-16

Wiki content is authored in the public `YourTongji/YourTJ-Wiki` Git repository. The forum is a
read-only projection: merge changes through GitHub, then wait for the scheduled sync or trigger the
admin Wiki sync. Pages are Markdown files beneath a top-level namespace directory; page URLs omit the
`.md` suffix and use the namespace slug where one is declared.

## Links And Assets

Repository-relative links are supported as `Current` behavior. Author page links with their repository
path and `.md` suffix, and author images or attachments with their repository-relative file path:

```markdown
[下一页](other.md?tab=2#section)
![示意图](../assets/a%20b.png?raw=1#preview)
[下载讲义](../assets/guide.pdf)
```

During sync, page links are resolved from the source Markdown file and rendered as normalized
`/wiki/<namespace-slug>/...` URLs without `.md`. Images and non-Markdown attachments are served by the
single forum binary through `/wiki/_assets/<repository-path>`. Query strings, fragments, URL encoding,
and nested paths are preserved. This lets an author keep links valid when a namespace directory has a
display name different from its URL slug.

Absolute URLs, protocol-relative URLs, site-root URLs such as `/static/logo.svg`, and anchor-only or
query-only links are left unchanged. Do not use relative links without a file extension for pages: page
targets must name their `.md` source file.

### Asset serving policy

`/wiki/_assets/` serves only a small allowlist of inert content types: images (PNG/JPEG/GIF/WebP/AVIF/
BMP/ICO), PDF, Office documents, archives, and plain text. Everything else — including HTML, SVG, XML,
JavaScript, and extension-less files — is forced to `application/octet-stream` with
`Content-Disposition: attachment`, so a repository file can never be rendered as same-origin executable
content. All asset responses carry `Content-Security-Policy: sandbox` and `X-Content-Type-Options:
nosniff` as defense in depth, and the endpoint is rate-limited per IP.

### Failure semantics

Every relative target must remain inside the repository. A linked Markdown page must be projected by the
same sync; an asset must exist as a regular, non-Markdown file. Hidden paths and path traversal are
rejected. Failures are split into two classes:

- **Security-class errors** (target escapes the repository root, or an asset symlink resolves outside
  the clone directory) abort the whole sync — they must never be bypassed by a per-page skip.
- **Content-class errors** (linked page or asset missing, image pointing at a `.md` page, unlinkable
  hidden page) skip only the offending page — it keeps its previous rendered version — while the rest
  of the repository syncs normally. The sync run records the skipped page and target so it can be
  corrected before the next run.

Repository hygiene requirements:

- Pages are regular files: symlinked `.md` files are rejected at scan time (a merged symlink would
  otherwise be followed to arbitrary server paths).
- Individual page sources are capped at 4 MiB.
- Page and asset filenames must not contain `%`, `#`, `?`, or the filesystem-reserved characters
  (`/ \ : * ? " < > |`): `%` is the URL-escape prefix and `#` opens a fragment, so Markdown link
  syntax cannot represent them reliably.

The top-level namespace `_assets` is reserved for the controlled asset route.
