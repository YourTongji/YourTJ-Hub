/**
 * 本地 Markdown 导入（issue #1630）：把一份 .md 文档整理成可直接发布的正文。
 *
 * - 去掉 BOM、统一换行、去掉 YAML front matter（其 title 作为标题候选）；文档第一个块是一级
 *   标题时取其纯文本作为标题候选。标题行留在正文里与其余内容一起处理图片，用作标题时再由
 *   stripLeadingHeading 移除。
 * - 站点不渲染原始 HTML：`<img>` 转成 Markdown 图片、注释删除，其余标签只计数提示。
 * - 引用式图片 `![alt][ref]` 与 Obsidian 嵌入 `![[图片.png]]` 展开为行内图片，后续只需处理一种写法。
 * - 图片按来源分类：remote（网络图）、local（相对/本地路径）、data（内嵌 base64）、
 *   site（已是本站附件）。网络图由浏览器下载后走编辑器同一条上传链路（含审核）；
 *   服务端不抓取外链图片（见 moderationDecision.EvidenceExternal），下载失败的图片
 *   交由用户手动选择本地文件，仍未解决的在导入时降级为普通链接 / 替代文字。
 * - 本地图片按文档所在目录解析相对路径（含 `../`），在用户选择或拖入的文件夹里匹配。
 *
 * 所有改写只发生在正文文本里，围栏/缩进代码块与行内代码原样保留。
 */
import { MAX_IMAGE_UPLOAD_BYTES } from './image'
import { markdownPreview } from './markdown'

export type ImportImageKind = 'remote' | 'local' | 'data' | 'site'

export interface ImportImage {
  /** 文档里书写的图片地址原文，同时作为改写时的匹配键 */
  src: string
  alt: string
  kind: ImportImageKind
}

export interface MarkdownImportResult {
  /** 标题候选：front matter 的 title 或首行一级标题；没有则为空串 */
  title: string
  titleSource: 'front-matter' | 'heading' | null
  /** 正文；标题来自首个一级标题时仍含该标题，用作标题时交给 stripLeadingHeading 移除 */
  body: string
  /** 按出现顺序去重后的图片 */
  images: ImportImage[]
  /** 残留的原始 HTML 标签数：站点按文本显示，导入前提示用户 */
  htmlTagCount: number
}

export const MARKDOWN_IMPORT_MAX_BYTES = 2 * 1024 * 1024
export const MARKDOWN_IMPORT_ACCEPT = '.md,.markdown,.mdown,.mkd,.txt,text/markdown,text/plain'

const MARKDOWN_FILE_RE = /\.(md|markdown|mdown|mkd|txt)$/i
const IMAGE_FILE_RE = /\.(png|jpe?g|gif|webp|avif|svg|bmp|heic|heif)$/i
const WIKI_IMAGE_RE = /!\[\[([^\]|\n]+?\.(?:png|jpe?g|gif|webp|avif|svg|bmp))(?:\|[^\]\n]*)?\]\]/gi
const FRONT_MATTER_RE = /^---[ \t]*\n([\s\S]*?)\n(?:---|\.\.\.)[ \t]*(?:\n|$)/
// 引用定义的地址：尖括号内可含空格与转义；否则到空白为止
const REFERENCE_DEF_RE = /^ {0,3}\[([^\]\n]+)\]:[ \t]*(?:<((?:[^<>\n\\]|\\.)*)>|([^\s<]\S*))(?:[ \t]+(?:"[^"\n]*"|'[^'\n]*'|\([^)\n]*\)))?[ \t]*$/gm
const ASCII_PUNCTUATION = /[!-/:-@[-`{-~]/
const HTML_TAG_RE = /<\/?[a-zA-Z][a-zA-Z0-9-]*(?:\s[^<>]*)?\/?>/g
const INLINE_CODE_RE = /(?<!`)(`+)(?!`)((?:(?!\n\n)[\s\S])*?[^`])\1(?!`)/g
const PLACEHOLDER_RE = /\u0000(\d+)\u0000/g

export function isMarkdownFile(file: File): boolean {
  return MARKDOWN_FILE_RE.test(file.name) || file.type === 'text/markdown'
}

export function isImportImageFile(file: File): boolean {
  return file.type.startsWith('image/') || IMAGE_FILE_RE.test(file.name)
}

/** 围栏/缩进代码块所在行：只在代码之外改写正文 */
function codeLines(source: string): boolean[] {
  const lines = new Array<boolean>(source.split('\n').length).fill(false)
  for (const token of markdownPreview.parse(source, {})) {
    if ((token.type === 'fence' || token.type === 'code_block') && token.map) {
      for (let line = token.map[0]; line < token.map[1]; line++) lines[line] = true
    }
  }
  return lines
}

/** 对代码块之外的正文执行 transform；行内代码先换成占位符，改写后还原。 */
function transformProse(source: string, transform: (prose: string) => string): string {
  const lines = source.split('\n')
  const code = codeLines(source)
  const out: string[] = []
  let start = 0
  while (start < lines.length) {
    let end = start
    while (end < lines.length && code[end] === code[start]) end++
    const chunk = lines.slice(start, end).join('\n')
    if (code[start]) {
      out.push(chunk)
    } else {
      const spans: string[] = []
      const masked = chunk.replace(INLINE_CODE_RE, (match) => `\u0000${spans.push(match) - 1}\u0000`)
      out.push(transform(masked).replace(PLACEHOLDER_RE, (_, index: string) => spans[Number(index)] ?? ''))
    }
    start = end
  }
  return out.join('\n')
}

interface ImageSpan {
  start: number
  end: number
  /** 方括号内的原文（含转义） */
  alt: string
  /** 去掉尖括号与反斜杠转义后的地址 */
  src: string
}

function unescapeDestination(value: string): string {
  return value.replace(/\\(.)/g, (match, char: string) => (ASCII_PUNCTUATION.test(char) ? char : match))
}

/**
 * 按 CommonMark 的行内图片语法切分：地址可以是尖括号（可含空格）或不含空白、括号配平的
 * 原文（支持转义），后面可跟标题。解析与改写共用，保证替换的正是被识别的那一段。
 */
function scanImages(prose: string): ImageSpan[] {
  const spans: ImageSpan[] = []
  let from = 0
  for (;;) {
    const start = prose.indexOf('![', from)
    if (start < 0) return spans
    from = start + 2
    let slashes = 0
    while (start - slashes - 1 >= 0 && prose[start - slashes - 1] === '\\') slashes++
    if (slashes % 2) continue
    const span = imageAt(prose, start)
    if (span) {
      spans.push(span)
      from = span.end
    }
  }
}

function imageAt(prose: string, start: number): ImageSpan | null {
  let index = start + 2
  for (let depth = 0; ; index++) {
    const char = prose[index]
    if (char === undefined || char === '\n') return null
    if (char === '\\') index++
    else if (char === '[') depth++
    else if (char === ']' && depth-- === 0) break
  }
  const alt = prose.slice(start + 2, index)
  if (prose[index + 1] !== '(') return null
  index = skipSpace(prose, index + 2)
  let src: string
  if (prose[index] === '<') {
    let close = index + 1
    for (; prose[close] !== '>'; close++) {
      const char = prose[close]
      if (char === undefined || char === '\n' || char === '<') return null
      if (char === '\\') close++
    }
    src = unescapeDestination(prose.slice(index + 1, close))
    index = close + 1
  } else {
    const begin = index
    for (let depth = 0; ; index++) {
      const char = prose[index]
      if (char === undefined || /[\s\u0000-\u001f]/.test(char)) break
      if (char === '\\' && ASCII_PUNCTUATION.test(prose[index + 1] ?? '')) index++
      else if (char === '(') depth++
      else if (char === ')' && depth-- === 0) break
    }
    if (index === begin) return null
    src = unescapeDestination(prose.slice(begin, index))
  }
  const afterDestination = index
  index = skipSpace(prose, index)
  const opener = prose[index]
  if (index > afterDestination && (opener === '"' || opener === "'" || opener === '(')) {
    const closer = opener === '(' ? ')' : opener
    for (index++; prose[index] !== closer; index++) {
      if (prose[index] === undefined || prose.startsWith('\n\n', index)) return null
      if (prose[index] === '\\') index++
    }
    index = skipSpace(prose, index + 1)
  }
  if (prose[index] !== ')' || !src) return null
  return { start, end: index + 1, alt, src }
}

function skipSpace(prose: string, index: number): number {
  let newline = false
  for (; prose[index] === ' ' || prose[index] === '\t' || (prose[index] === '\n' && !newline); index++) {
    if (prose[index] === '\n') newline = true
  }
  return index
}

function replaceImages(prose: string, replace: (span: ImageSpan) => string): string {
  let out = ''
  let cursor = 0
  for (const span of scanImages(prose)) {
    out += prose.slice(cursor, span.start) + replace(span)
    cursor = span.end
  }
  return out + prose.slice(cursor)
}

function htmlAttribute(tag: string, name: string): string {
  const match = new RegExp(`\\b${name}\\s*=\\s*(?:"([^"]*)"|'([^']*)'|([^\\s>]+))`, 'i').exec(tag)
  return (match?.[1] ?? match?.[2] ?? match?.[3] ?? '').trim()
}

function escapeAlt(alt: string): string {
  return alt.replace(/[[\]\\]/g, (char) => `\\${char}`).replace(/\s+/g, ' ').trim()
}

function unescapeAlt(alt: string): string {
  return alt.replace(/\\(.)/g, '$1')
}

function linkDestination(url: string): string {
  return /[\s()<>]/.test(url) ? `<${url.replace(/[<>]/g, encodeURIComponent)}>` : url
}

export function imageMarkdown(alt: string, url: string): string {
  return `![${escapeAlt(alt)}](${linkDestination(url)})`
}

function normalizeLabel(label: string): string {
  return label.trim().replace(/\s+/g, ' ').toLowerCase()
}

function stripFrontMatter(text: string): { text: string; title: string } {
  const match = FRONT_MATTER_RE.exec(text)
  if (!match) return { text, title: '' }
  const titleLine = /^title:[ \t]*(.+?)[ \t]*$/m.exec(match[1])
  const title = (titleLine?.[1] ?? '').replace(/^(['"])(.*)\1$/, '$2').trim()
  return { text: text.slice(match[0].length), title }
}

/** `<img>` 转 Markdown 图片、删除 HTML 注释、展开引用式图片。 */
function normalizeProse(prose: string, definitions: Map<string, string>): string {
  return prose
    .replace(/<!--[\s\S]*?-->/g, '')
    .replace(/<img\b[^>]*>/gi, (tag) => {
      const src = htmlAttribute(tag, 'src')
      return src ? imageMarkdown(htmlAttribute(tag, 'alt'), src) : ''
    })
    .replace(/!\[((?:\\.|[^\]\\\n])*)\]\[([^\]\n]*)\]/g, (match, alt: string, label: string) => {
      const url = definitions.get(normalizeLabel(label || unescapeAlt(alt)))
      return url ? `![${alt}](${linkDestination(url)})` : match
    })
    .replace(/!\[((?:\\.|[^\]\\\n])+)\](?![([:])/g, (match, alt: string) => {
      const url = definitions.get(normalizeLabel(unescapeAlt(alt)))
      return url ? `![${alt}](${linkDestination(url)})` : match
    })
    .replace(WIKI_IMAGE_RE, (_, target: string) => `![](${linkDestination(target.trim())})`)
}

/** 文档第一个块是一级标题时返回它的行范围与纯文本（代码、HTML 等其他块不算）。 */
function leadingHeading(text: string): { title: string; end: number } | null {
  const tokens = markdownPreview.parse(text, {})
  const open = tokens[0]
  if (open?.type !== 'heading_open' || open.tag !== 'h1' || !open.map) return null
  let title = ''
  for (const child of tokens[1]?.children ?? []) {
    if (child.type === 'text' || child.type === 'code_inline') title += child.content
    else if (child.type === 'softbreak' || child.type === 'hardbreak') title += ' '
  }
  title = title.replace(/\s+/g, ' ').trim()
  return title ? { title, end: open.map[1] } : null
}

/** 去掉正文开头作为标题使用的一级标题（图片改写之后调用，行范围不变）。 */
export function stripLeadingHeading(body: string): string {
  const heading = leadingHeading(body)
  if (!heading) return body
  return body.split('\n').slice(heading.end).join('\n').replace(/^\s*\n/, '')
}

export function classifyImageSrc(src: string, siteOrigin = ''): ImportImageKind {
  if (/^data:image\//i.test(src)) return 'data'
  if (/^(?:https?:)?\/\//i.test(src)) {
    try {
      const url = new URL(src, siteOrigin || 'https://invalid.local')
      return siteOrigin && url.origin === siteOrigin ? 'site' : 'remote'
    } catch {
      return 'remote'
    }
  }
  if (src.startsWith('/file/img/')) return 'site'
  return 'local'
}

export function parseMarkdownImport(source: string, options: { siteOrigin?: string } = {}): MarkdownImportResult {
  const normalized = source.replace(/^\uFEFF/, '').replace(/\r\n?/g, '\n')
  const frontMatter = stripFrontMatter(normalized)

  const definitions = new Map<string, string>()
  transformProse(frontMatter.text, (prose) => {
    for (const match of prose.matchAll(REFERENCE_DEF_RE)) {
      const label = normalizeLabel(match[1])
      if (!definitions.has(label)) definitions.set(label, unescapeDestination(match[2] ?? match[3]))
    }
    return prose
  })

  const body = transformProse(frontMatter.text, (prose) => normalizeProse(prose, definitions)).replace(/^\s*\n/, '')
  let title = frontMatter.title
  let titleSource: MarkdownImportResult['titleSource'] = title ? 'front-matter' : null
  if (!title) {
    const heading = leadingHeading(body)
    if (heading) {
      title = heading.title
      titleSource = 'heading'
    }
  }

  const images: ImportImage[] = []
  const seen = new Set<string>()
  let htmlTagCount = 0
  transformProse(body, (prose) => {
    for (const { src, alt } of scanImages(prose)) {
      if (seen.has(src)) continue
      seen.add(src)
      images.push({ src, alt: unescapeAlt(alt), kind: classifyImageSrc(src, options.siteOrigin) })
    }
    htmlTagCount += prose.match(HTML_TAG_RE)?.length ?? 0
    return prose
  })

  return { title, titleSource, body: body.trimEnd(), images, htmlTagCount }
}

/**
 * 逐个改写正文里的图片：resolve 返回替换文本则替换整段图片语法，返回 undefined 保留原样。
 * 与解析同一套规则（只改代码之外的正文）。
 */
export function rewriteImportImages(body: string, resolve: (src: string, alt: string) => string | undefined): string {
  return transformProse(body, (prose) =>
    replaceImages(prose, (span) => resolve(span.src, unescapeAlt(span.alt)) ?? prose.slice(span.start, span.end)),
  )
}

/** 未能上传的图片：网络图降级为普通链接（信息不丢、也不外链显示），本地/内嵌图只保留替代文字。 */
export function unresolvedImageFallback(image: ImportImage): string {
  const label = escapeAlt(image.alt) || fileNameFromSrc(image.src)
  if (image.kind === 'remote') return `[${label}](${linkDestination(image.src)})`
  return escapeAlt(image.alt)
}

const MIME_EXTENSIONS: Record<string, string> = {
  'image/png': 'png',
  'image/jpeg': 'jpg',
  'image/gif': 'gif',
  'image/webp': 'webp',
  'image/avif': 'avif',
  'image/svg+xml': 'svg',
  'image/bmp': 'bmp',
}

export function fileNameFromSrc(src: string, mime = ''): string {
  let name = ''
  if (!src.startsWith('data:')) {
    const path = src.replace(/[?#].*$/, '').replace(/\\/g, '/')
    const last = path.split('/').filter(Boolean).pop() ?? ''
    try {
      name = decodeURIComponent(last)
    } catch {
      name = last
    }
  }
  if (!/^[^/]+\.[a-z0-9]{2,5}$/i.test(name)) {
    const extension = MIME_EXTENSIONS[mime.toLowerCase()] ?? 'png'
    name = `${name || 'image'}.${extension}`
  }
  return name
}

/** 导入时可用的本地文件：path 是相对所选文件夹（或拖入位置）的路径，用 `/` 分隔，含文件名。 */
export interface LocalImportFile {
  file: File
  path: string
}

/** 单次选择或拖入最多读取的文件数：误选整个磁盘目录时不至于卡住页面。 */
export const MAX_IMPORT_FILES = 5000

/** 文件选择框给出的文件：选择文件夹时浏览器提供 webkitRelativePath。 */
export function localImportFiles(files: readonly File[]): LocalImportFile[] {
  return files.slice(0, MAX_IMPORT_FILES).map((file) => ({ file, path: (file.webkitRelativePath || file.name).replace(/\\/g, '/') }))
}
const SKIPPED_DIRECTORIES = new Set(['node_modules', '.git', '.obsidian', '.trash'])

/**
 * 拖入的文件与文件夹（递归展开）。必须在 drop 事件里同步调用：DataTransfer 在事件结束后失效，
 * 所以先同步取出全部 entry 再异步读取。浏览器不支持 entry 时退回 dataTransfer.files。
 */
export async function collectDroppedFiles(dataTransfer: DataTransfer): Promise<LocalImportFile[]> {
  const entries = Array.from(dataTransfer.items ?? [])
    .filter((item) => item.kind === 'file')
    .map((item) => item.webkitGetAsEntry?.() ?? null)
  if (!entries.length || entries.some((entry) => !entry)) return localImportFiles(Array.from(dataTransfer.files ?? []))
  const out: LocalImportFile[] = []
  const visit = async (entry: FileSystemEntry, prefix: string): Promise<void> => {
    if (out.length >= MAX_IMPORT_FILES) return
    const path = prefix ? `${prefix}/${entry.name}` : entry.name
    if (entry.isFile) {
      const file = await new Promise<File | null>((resolve) => (entry as FileSystemFileEntry).file(resolve, () => resolve(null)))
      if (file && out.length < MAX_IMPORT_FILES) out.push({ file, path })
      return
    }
    if (!entry.isDirectory || SKIPPED_DIRECTORIES.has(entry.name)) return
    const reader = (entry as FileSystemDirectoryEntry).createReader()
    for (;;) {
      // readEntries 分批返回，直到返回空数组
      const batch = await new Promise<FileSystemEntry[]>((resolve) => reader.readEntries(resolve, () => resolve([])))
      if (!batch.length) return
      for (const child of batch) await visit(child, path)
    }
  }
  for (const entry of entries) await visit(entry!, '')
  return out
}

function pathSegments(path: string): string[] {
  return path.split('/').filter((segment) => segment && segment !== '.')
}

/** 本地图片地址的路径段：逐段解码（`%23` 还原成 `#`，`%2F` 不会变成目录分隔）。 */
function localPathSegments(src: string): { absolute: boolean; segments: string[] } {
  const path = src.replace(/[?#].*$/, '').replace(/^file:\/\//i, '').replace(/\\/g, '/').toLowerCase()
  const segments = pathSegments(path).map((segment) => {
    try {
      return decodeURIComponent(segment)
    } catch {
      // 保留原文：不合法的百分号编码按字面比较
      return segment
    }
  })
  return { absolute: /^(?:\/|[a-z]:\/)/.test(path), segments }
}

/**
 * 找出与文档里本地图片地址对应的文件：
 * 1. 相对地址按文档所在目录解析（含 `../`），与文件路径完全一致即命中；
 * 2. 否则比较路径末尾连续相同的段数（文件名至少相同），取唯一的最长匹配——用户只选了图片
 *    文件夹、或地址是另一台电脑上的绝对路径时也能找到；多个文件同样匹配时不猜，交给用户选择。
 */
export function matchLocalImage(src: string, files: readonly LocalImportFile[], markdownPath = ''): LocalImportFile | undefined {
  const { absolute, segments } = localPathSegments(src)
  if (!segments.length) return undefined
  const keyed = files.map((entry) => ({ entry, segments: pathSegments(entry.path.toLowerCase()) }))

  if (!absolute) {
    const resolved = pathSegments(markdownPath.toLowerCase()).slice(0, -1)
    let escaped = false
    for (const segment of segments) {
      if (segment !== '..') resolved.push(segment)
      else if (!resolved.pop()) escaped = true
    }
    // 逐段比较：解码出的 `/` 只是文件名里的字符
    const exact = escaped
      ? undefined
      : keyed.find((item) => item.segments.length === resolved.length && item.segments.every((segment, index) => segment === resolved[index]))
    if (exact) return exact.entry
  }

  const tail = segments.filter((segment) => segment !== '..' && !/^[a-z]:$/.test(segment))
  let best: LocalImportFile | undefined
  let bestScore = 0
  let tied = false
  for (const item of keyed) {
    let score = 0
    while (score < tail.length && score < item.segments.length && tail[tail.length - 1 - score] === item.segments[item.segments.length - 1 - score]) score++
    if (score === 0 || score < bestScore) continue
    tied = score === bestScore
    if (score > bestScore) {
      best = item.entry
      bestScore = score
    }
  }
  return tied ? undefined : best
}

export type ImportImageErrorCode = 'blocked' | 'http' | 'not-image' | 'too-large' | 'invalid'

export class ImportImageError extends Error {
  constructor(readonly code: ImportImageErrorCode, readonly status = 0) {
    super(code)
    this.name = 'ImportImageError'
  }
}

/**
 * 浏览器内下载网络图片：不带 Cookie、不发 Referer。图床未开放跨域（CORS）或网络失败时
 * 抛 blocked，由界面提示「手动保存后选择本地文件」。边下载边累计字节，超过上传上限立即
 * 中止（too-large），不会先把超大响应整个读进内存。
 */
export async function downloadRemoteImage(
  src: string,
  fetchImpl: typeof fetch = fetch,
  signal?: AbortSignal,
  maxBytes = MAX_IMAGE_UPLOAD_BYTES,
): Promise<File> {
  const url = src.startsWith('//') ? `https:${src}` : src
  let response: Response
  try {
    response = await fetchImpl(url, { mode: 'cors', credentials: 'omit', referrerPolicy: 'no-referrer', signal })
  } catch (error) {
    if (signal?.aborted) throw error
    throw new ImportImageError('blocked')
  }
  if (!response.ok) throw new ImportImageError('http', response.status)
  const type = (response.headers.get('content-type') ?? '').split(';')[0].trim().toLowerCase()
  if (!type.startsWith('image/')) {
    void response.body?.cancel().catch(() => {})
    throw new ImportImageError('not-image')
  }
  // Content-Length 只用于提前拒绝；实际字节仍逐块累计
  if (Number(response.headers.get('content-length')) > maxBytes) {
    void response.body?.cancel().catch(() => {})
    throw new ImportImageError('too-large')
  }
  const reader = response.body?.getReader()
  if (!reader) {
    const blob = await response.blob()
    if (blob.size > maxBytes) throw new ImportImageError('too-large')
    return new File([blob], fileNameFromSrc(src, type), { type })
  }
  const chunks: Uint8Array<ArrayBuffer>[] = []
  let total = 0
  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    total += value.byteLength
    if (total > maxBytes) {
      void reader.cancel().catch(() => {})
      throw new ImportImageError('too-large')
    }
    chunks.push(value as Uint8Array<ArrayBuffer>)
  }
  return new File(chunks, fileNameFromSrc(src, type), { type })
}

/** 内嵌 base64 图片转文件（不经 fetch：页面 CSP 的 connect-src 不放行 data:）。 */
export function dataUrlToFile(src: string): File {
  const match = /^data:(image\/[a-z0-9.+-]+)(;base64)?,(.*)$/is.exec(src)
  if (!match) throw new ImportImageError('invalid')
  const type = match[1].toLowerCase()
  let bytes: Uint8Array<ArrayBuffer>
  try {
    if (match[2]) {
      const binary = atob(match[3].replace(/\s+/g, ''))
      bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
    } else {
      bytes = new TextEncoder().encode(decodeURIComponent(match[3]))
    }
  } catch {
    throw new ImportImageError('invalid')
  }
  return new File([bytes], fileNameFromSrc('', type), { type })
}
