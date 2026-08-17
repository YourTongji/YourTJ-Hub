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

## Wiki local search (Current)

Wiki has its own paragraph-level search, separate from the site's forum search. The desktop sidebar
and mobile Wiki drawer expose the search entry; `/` and `Ctrl/Cmd+K` open the same search panel while
the user is in Wiki mode.

The public endpoint is `GET /api/wiki/search`:

- `q` is the trimmed query, limited to 100 characters. An empty query returns an empty result without
  contacting Meilisearch.
- `limit` requests up to 20 page results and defaults to 12 when omitted or out of range.
- Results are aggregated at the page level. Each item represents one public page and carries all
  matching paragraph anchors, so `total` counts distinct pages rather than paragraph hits.
- Paragraph anchors use the form `s-<n>`. Selecting a result opens the page at its first matching
  paragraph and briefly highlights the target; subsequent matches can be cycled from the page.
- `searchUnavailable: true` means the optional Meilisearch backend is unavailable. The API keeps HTTP
  200 and returns an empty item list so Wiki reading remains available.

Search snippets contain `<mark>` tags only for matched text. Search-rendering code escapes all other
HTML, so Markdown text that looks like an HTML tag remains visible as text rather than executable
markup. The index contains only currently public Wiki pages; page updates, empty pages, and soft
deletions remove the page's old paragraph documents.

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

### CDN selection

Site administrators choose how asset URLs are rendered during sync from the Wiki admin panel
(资源 CDN section): serve from this forum (`self`, the default) or through the jsDelivr GitHub
mirror (`jsDelivr`). With `jsDelivr` selected, rendered asset URLs point at
`https://cdn.jsdelivr.net/gh/<owner>/<repo>@<branch>/<repository-path>` instead of
`/wiki/_assets/...`, offloading bandwidth and disk I/O from the forum binary. The setting is stored
in `page_config` (`WikiSyncSettings.AssetCDN`) and takes effect on the next sync (webhook, manual, or
scheduled); the repo owner/name/branch must be parseable from `[wiki.git].repo`, otherwise assets
fall back to the self route. The `_assets` route, its allowlist, and CSP headers remain available
regardless of the selection.

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

## Contributors (Current)

Wiki 详情页右侧「贡献者」区块显示页面的 Git 作者（`Current` 行为）：每次同步
（webhook/手动/定时）从仓库 `git log` 重新聚合**每个页面**的编辑者与提交数，
并缓存于 `wiki_pages.contributors_json`。展示按 GitHub noreply 隐私邮箱解析出的
username 聚合（合并新旧邮箱格式同人），同时提供 GitHub 头像直链与主页外链；
自定义邮箱贡献者无头像/外链，降级为首字母占位。贡献者无需论坛账号（无数字
用户 ID）。GitHub 作者（非 PR 审核者）即该页面的贡献者名单。
