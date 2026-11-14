// @vitest-environment happy-dom
import { describe, expect, test, vi } from 'vitest'
import { renderMarkdownPreview } from '../src/runtime/markdown'
import {
  ImportImageError,
  classifyImageSrc,
  dataUrlToFile,
  downloadRemoteImage,
  fileNameFromSrc,
  collectDroppedFiles,
  MAX_IMPORT_FILES,
  localImportFiles,
  matchLocalImage,
  parseMarkdownImport,
  rewriteImportImages,
  stripLeadingHeading,
  unresolvedImageFallback,
} from '../src/runtime/markdown-import'

const SITE = 'https://forum.example.test'

describe('Markdown 导入解析（issue #1630）', () => {
  test('front matter 的 title 作为标题候选，正文去掉 front matter、BOM 并统一换行', () => {
    const result = parseMarkdownImport('\uFEFF---\r\ntitle: "期末复习指南"\r\ntags: [a]\r\n---\r\n\r\n第一段\r\n\r\n第二段\r\n')
    expect(result.title).toBe('期末复习指南')
    expect(result.titleSource).toBe('front-matter')
    expect(result.body).toBe('第一段\n\n第二段')
  })

  test('没有 front matter 时首个一级标题作为标题候选，用作标题时才从正文移除；二级标题不当作标题', () => {
    const heading = parseMarkdownImport('# 选课经验\n\n正文')
    expect(heading).toMatchObject({ title: '选课经验', titleSource: 'heading', body: '# 选课经验\n\n正文' })
    expect(stripLeadingHeading(heading.body)).toBe('正文')
    expect(parseMarkdownImport('选课经验\n===\n\n正文').title).toBe('选课经验')

    const second = parseMarkdownImport('## 小节\n\n正文')
    expect(second).toMatchObject({ title: '', titleSource: null, body: '## 小节\n\n正文' })
  })

  test('按来源分类图片并按出现顺序去重', () => {
    const result = parseMarkdownImport(
      [
        '![封面](https://img.example.com/a.png "封面图")',
        '![本地](./assets/b.jpg)',
        '![站内](/file/img/2026/c.webp)',
        `![同源](${SITE}/file/img/d.png)`,
        '![内嵌](data:image/png;base64,iVBORw0KGgo=)',
        '![重复](https://img.example.com/a.png)',
      ].join('\n\n'),
      { siteOrigin: SITE },
    )
    expect(result.images).toEqual([
      { src: 'https://img.example.com/a.png', alt: '封面', kind: 'remote' },
      { src: './assets/b.jpg', alt: '本地', kind: 'local' },
      { src: '/file/img/2026/c.webp', alt: '站内', kind: 'site' },
      { src: `${SITE}/file/img/d.png`, alt: '同源', kind: 'site' },
      { src: 'data:image/png;base64,iVBORw0KGgo=', alt: '内嵌', kind: 'data' },
    ])
  })

  test('代码块与行内代码里的图片语法和 HTML 原样保留，不计入图片', () => {
    const source = [
      '```md',
      '![示例](https://img.example.com/code.png)',
      '<img src="https://img.example.com/code2.png">',
      '```',
      '',
      '行内 `![x](https://img.example.com/inline.png)` 不算',
      '',
      '    ![缩进代码](https://img.example.com/indented.png)',
    ].join('\n')
    const result = parseMarkdownImport(source)
    expect(result.images).toEqual([])
    expect(result.htmlTagCount).toBe(0)
    expect(result.body).toBe(source)
  })

  test('<img> 转成 Markdown 图片，HTML 注释删除，其余 HTML 标签计数提示', () => {
    const result = parseMarkdownImport('<!-- 草稿备注 -->\n<p align="center"><img src="https://img.example.com/logo.svg" alt="Logo" width="120"></p>\n\n<details>折叠</details>')
    expect(result.body).toBe('<p align="center">![Logo](https://img.example.com/logo.svg)</p>\n\n<details>折叠</details>')
    expect(result.images).toEqual([{ src: 'https://img.example.com/logo.svg', alt: 'Logo', kind: 'remote' }])
    expect(result.htmlTagCount).toBe(4)
  })

  test('引用式图片展开为行内图片', () => {
    const result = parseMarkdownImport('![图一][fig]\n\n![fig]\n\n[fig]: https://img.example.com/fig.png "说明"')
    expect(result.body.split('\n\n').slice(0, 2)).toEqual([
      '![图一](https://img.example.com/fig.png)',
      '![fig](https://img.example.com/fig.png)',
    ])
    expect(result.images).toHaveLength(1)
  })

  test('首行缩进代码不是标题，原样保留', () => {
    const result = parseMarkdownImport('    # 代码标题\n\n正文')
    expect(result).toMatchObject({ title: '', titleSource: null, body: '    # 代码标题\n\n正文' })
    expect(stripLeadingHeading(result.body)).toBe(result.body)
  })

  test('一级标题里的图片进入同一图片流程，标题只取纯文本', () => {
    const result = parseMarkdownImport('# ![封面](https://img.example/a.png) 期末 `复习`\n\n正文')
    expect(result.title).toBe('期末 复习')
    expect(result.images.map((image) => image.src)).toEqual(['https://img.example/a.png'])
    const rewritten = rewriteImportImages(result.body, () => '![封面](/file/img/a.webp)')
    expect(rewritten).toBe('# ![封面](/file/img/a.webp) 期末 `复习`\n\n正文')
    expect(stripLeadingHeading(rewritten)).toBe('正文')
    // 只有图片、没有文字的一级标题不作为标题候选
    expect(parseMarkdownImport('# ![封面](https://img.example/a.png)\n\n正文').titleSource).toBeNull()
  })

  test('图片地址按 CommonMark 切分：配平/转义括号与带空格的尖括号引用定义', () => {
    const source = '![括号](https://img.example/a_(b).png) ![转义](https://img.example/c\\).png "标题") ![引用][r]\n\n[r]: <images/my shot.png> "说明"'
    const result = parseMarkdownImport(source)
    expect(result.images.map((image) => image.src)).toEqual(['https://img.example/a_(b).png', 'https://img.example/c).png', 'images/my shot.png'])
    const rewritten = rewriteImportImages(result.body, (src) => (src.startsWith('https') ? '[IMG]' : undefined))
    expect(rewritten.split('\n')[0]).toBe('[IMG] [IMG] ![引用](<images/my shot.png>)')
    const fallback = rewriteImportImages('![括号](https://img.example/a_(b).png)', (src, alt) => unresolvedImageFallback({ src, alt, kind: 'remote' }))
    expect(fallback).toBe('[括号](<https://img.example/a_(b).png>)')
  })

  test('包含空格的本地路径用尖括号目标书写也能识别', () => {
    const result = parseMarkdownImport('![截图](<images/my shot.png>)')
    expect(result.images).toEqual([{ src: 'images/my shot.png', alt: '截图', kind: 'local' }])
  })
})

describe('Markdown 导入改写', () => {
  test('rewriteImportImages 只替换正文中的图片，代码块不动', () => {
    const body = '![a](https://x.test/a.png)\n\n```\n![a](https://x.test/a.png)\n```'
    const next = rewriteImportImages(body, (src, alt) => (src === 'https://x.test/a.png' ? `![${alt}](/file/img/a.webp)` : undefined))
    expect(next).toBe('![a](/file/img/a.webp)\n\n```\n![a](https://x.test/a.png)\n```')
  })

  test('未上传成功的网络图降级为链接，本地与内嵌图只保留替代文字', () => {
    expect(unresolvedImageFallback({ src: 'https://x.test/p/photo.jpg', alt: '', kind: 'remote' })).toBe('[photo.jpg](https://x.test/p/photo.jpg)')
    expect(unresolvedImageFallback({ src: 'https://x.test/a.png', alt: '示意 [1]', kind: 'remote' })).toBe('[示意 \\[1\\]](https://x.test/a.png)')
    expect(unresolvedImageFallback({ src: './a.png', alt: '流程图', kind: 'local' })).toBe('流程图')
  })

  test('文件名取自地址，缺扩展名时按类型补齐', () => {
    expect(fileNameFromSrc('https://x.test/img/%E5%9B%BE.png?x=1')).toBe('图.png')
    expect(fileNameFromSrc('https://x.test/render', 'image/webp')).toBe('render.webp')
    expect(fileNameFromSrc('data:image/png;base64,AAAA', 'image/jpeg')).toBe('image.jpg')
  })

  test('本地图片先按文档目录解析相对路径，再取唯一的最长路径后缀匹配', () => {
    const image = (path: string) => ({ file: new File(['x'], path.split('/').pop()!, { type: 'image/png' }), path })
    const nested = image('notes/assets/shot.png')
    const loose = image('shot.png')
    const shared = image('vault/attachments/diagram.png')
    const other = image('vault/notes/assets/diagram.png')
    expect(matchLocalImage('./assets/shot.png', [loose, nested])).toBe(nested)
    expect(matchLocalImage('C:\\Users\\me\\Pictures\\SHOT.png', [loose])).toBe(loose)
    expect(matchLocalImage('missing.png', [loose])).toBeUndefined()
    // 文档在子目录，`../` 回到上级目录
    expect(matchLocalImage('../attachments/diagram.png', [other, shared], 'vault/notes/post.md')).toBe(shared)
    expect(matchLocalImage('assets/diagram.png', [shared, other], 'vault/notes/post.md')).toBe(other)
    // 百分号编码的空格、file:// 绝对路径
    const spaced = image('pics/my shot.png')
    expect(matchLocalImage('pics/my%20shot.png', [spaced])).toBe(spaced)
    expect(matchLocalImage('file:///Users/me/blog/pics/my%20shot.png', [spaced])).toBe(spaced)
    // 只有文件名相同且有多个候选时不猜
    expect(matchLocalImage('diagram.png', [shared, other])).toBeUndefined()
  })

  test('localImportFiles 保留文件夹内的相对路径', () => {
    const file = new File(['x'], 'a.png', { type: 'image/png' })
    Object.defineProperty(file, 'webkitRelativePath', { value: 'blog/img/a.png' })
    const loose = new File(['y'], 'b.png', { type: 'image/png' })
    expect(localImportFiles([file, loose]).map((entry) => entry.path)).toEqual(['blog/img/a.png', 'b.png'])
  })

  test('单次最多读取的文件数在文件选择与拖入退回路径上同样生效', async () => {
    const files = Array.from({ length: MAX_IMPORT_FILES + 1 }, (_, index) => new File(['x'], `${index}.png`))
    expect(localImportFiles(files)).toHaveLength(MAX_IMPORT_FILES)
    const fallback = { items: [], files } as unknown as DataTransfer
    expect(await collectDroppedFiles(fallback)).toHaveLength(MAX_IMPORT_FILES)
  })

  test('本地图片地址逐段解码：%23 匹配文件名里的 #，%2F 不当作目录分隔', () => {
    const hash = { file: new File(['x'], 'a#b.png'), path: 'post/pics/a#b.png' }
    const slash = { file: new File(['x'], 'b.png'), path: 'post/a/b.png' }
    expect(matchLocalImage('pics/a%23b.png', [hash], 'post/post.md')).toBe(hash)
    expect(matchLocalImage('a%2Fb.png', [slash], 'post/post.md')).toBeUndefined()
  })

  test('collectDroppedFiles 递归展开拖入的文件夹并跳过依赖目录', async () => {
    const fileEntry = (name: string) => ({ name, isFile: true, isDirectory: false, file: (ok: (file: File) => void) => ok(new File(['x'], name)) })
    const dirEntry = (name: string, children: unknown[]) => ({
      name, isFile: false, isDirectory: true,
      createReader: () => {
        let read = false
        return { readEntries: (ok: (entries: unknown[]) => void) => { ok(read ? [] : children); read = true } }
      },
    })
    const root = dirEntry('blog', [fileEntry('post.md'), dirEntry('img', [fileEntry('a.png')]), dirEntry('node_modules', [fileEntry('x.png')])])
    const transfer = { items: [{ kind: 'file', webkitGetAsEntry: () => root }], files: [] } as unknown as DataTransfer
    const files = await collectDroppedFiles(transfer)
    expect(files.map((entry) => entry.path)).toEqual(['blog/post.md', 'blog/img/a.png'])
  })

  test('Obsidian 图片嵌入转换为行内图片', () => {
    const result = parseMarkdownImport('正文 ![[附件 1.png|300]] 与 ![[notes.md]]')
    expect(result.body).toBe('正文 ![](<附件 1.png>) 与 ![[notes.md]]')
    expect(result.images).toEqual([{ src: '附件 1.png', alt: '', kind: 'local' }])
  })

  test('classifyImageSrc：协议相对地址按网络图处理', () => {
    expect(classifyImageSrc('//cdn.example.com/a.png', SITE)).toBe('remote')
  })
})

describe('Markdown 导入图片获取', () => {
  test('下载网络图片：不带 Cookie 与 Referer，返回带扩展名的图片文件', async () => {
    const fetchImpl = vi.fn(async () => new Response(new Blob(['png'], { type: 'image/png' }), { status: 200 }))
    const file = await downloadRemoteImage('//img.example.com/a/cover', fetchImpl as unknown as typeof fetch)
    expect(fetchImpl).toHaveBeenCalledWith('https://img.example.com/a/cover', expect.objectContaining({ mode: 'cors', credentials: 'omit', referrerPolicy: 'no-referrer' }))
    expect(file.name).toBe('cover.png')
    expect(file.type).toBe('image/png')
  })

  test('跨域被拦截、HTTP 失败与非图片内容给出可区分的错误', async () => {
    const blocked = vi.fn(async () => { throw new TypeError('Failed to fetch') })
    await expect(downloadRemoteImage('https://x.test/a.png', blocked as unknown as typeof fetch)).rejects.toMatchObject({ code: 'blocked' })

    const missing = vi.fn(async () => new Response('', { status: 404 }))
    await expect(downloadRemoteImage('https://x.test/a.png', missing as unknown as typeof fetch)).rejects.toMatchObject({ code: 'http', status: 404 })

    const html = vi.fn(async () => new Response(new Blob(['<html>'], { type: 'text/html' }), { status: 200 }))
    await expect(downloadRemoteImage('https://x.test/a.png', html as unknown as typeof fetch)).rejects.toBeInstanceOf(ImportImageError)
  })

  test('网络图片超过上传上限时在下载过程中中止', async () => {
    const declared = vi.fn(async () => new Response('x', { status: 200, headers: { 'content-type': 'image/png', 'content-length': '11' } }))
    await expect(downloadRemoteImage('https://x.test/a.png', declared as unknown as typeof fetch, undefined, 10)).rejects.toMatchObject({ code: 'too-large' })

    let pulled = 0
    let cancelled = false
    const stream = new ReadableStream<Uint8Array>({
      pull(controller) {
        pulled++
        controller.enqueue(new Uint8Array(4))
      },
      cancel() {
        cancelled = true
      },
    })
    const streamed = vi.fn(async () => new Response(stream, { status: 200, headers: { 'content-type': 'image/png' } }))
    await expect(downloadRemoteImage('https://x.test/a.png', streamed as unknown as typeof fetch, undefined, 10)).rejects.toMatchObject({ code: 'too-large' })
    expect(pulled).toBeLessThan(5)
    expect(cancelled).toBe(true)

    const fits = vi.fn(async () => new Response(new Uint8Array(10), { status: 200, headers: { 'content-type': 'image/png; charset=binary' } }))
    const file = await downloadRemoteImage('https://x.test/a.png', fits as unknown as typeof fetch, undefined, 10)
    expect(file.size).toBe(10)
    expect(file.type).toBe('image/png')
  })

  test('内嵌 base64 图片转文件', async () => {
    const file = dataUrlToFile('data:image/gif;base64,R0lGODlhAQABAAAAACw=')
    expect(file.type).toBe('image/gif')
    expect(file.name).toBe('image.gif')
    expect(file.size).toBeGreaterThan(0)
    expect(() => dataUrlToFile('data:text/plain,hi')).toThrow(ImportImageError)
  })
})

describe('import preview stickers', () => {
  const stickers = new Map([['smile', '/s/smile.png']])

  test('renders resolvable tokens as published sticker images only in prose', () => {
    const html = renderMarkdownPreview(
      'Hi [:sticker:smile:] and [:sticker:gone:]\n\n`[:sticker:smile:]` [link [:sticker:smile:]](/x)',
      { stickers },
    )
    expect(html.match(/data-gf-sticker="smile"/g)).toHaveLength(1)
    expect(html).toContain('<img src="/s/smile.png" alt="smile" data-gf-sticker="smile">')
    expect(html).toContain('[:sticker:gone:]')
    expect(html).toContain('<code>[:sticker:smile:]</code>')
  })

  test('leaves tokens literal without a sticker map', () => {
    expect(renderMarkdownPreview('Hi [:sticker:smile:]')).toContain('[:sticker:smile:]')
  })
})
