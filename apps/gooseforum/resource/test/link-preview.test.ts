// @vitest-environment happy-dom
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, describe, expect, test, vi } from 'vitest'
import type { LinkPreview } from '@gooseforum/client'
import {
  cancelExternalLinkGuard,
  externalDomain,
  openExternalLinkGuard,
  registrableDomain,
  setExternalLinkGuardInternalOrigins,
  trustExternalDomain,
} from '../src/runtime/external-link-guard'
import { createLinkPreviewHintController, formatLinkPreviewHintDomains } from '../src/runtime/link-preview-hint'
import {
  findRenderedLinkCandidates,
  LINK_PREVIEW_REQUEST_LIMIT,
  MAX_LINK_PREVIEWS,
  scanMarkdownLinkCandidates,
  usableLinkPreview,
} from '../src/runtime/link-preview'
import { disposeLinkPreviews, enhanceLinkPreviews } from '../src/runtime/content-enhancements/link-preview'

const apiMocks = vi.hoisted(() => ({
  resolveLinkPreviews: vi.fn(),
}))

vi.mock('@/runtime/api', () => apiMocks)

interface CandidateFixture {
  cases: Array<{ name: string; markdown: string; urls: string[] }>
}

const fixturePath = resolve(process.cwd(), '../../../packages/api-contract/fixtures/link-preview-markdown-candidates.json')
const fixture = JSON.parse(readFileSync(fixturePath, 'utf8')) as CandidateFixture

function readyPreview(url: string): LinkPreview {
  return {
    requestedUrl: url,
    kind: 'external',
    status: 'ready',
    url,
    displayHost: new URL(url).hostname,
    title: 'Example',
  }
}

describe('shared standalone-link fixture', () => {
  for (const item of fixture.cases) {
    test(item.name, () => {
      expect(scanMarkdownLinkCandidates(item.markdown)).toEqual(item.urls)
    })
  }
})

describe('rendered standalone-link detection', () => {
  test('keeps ordinary links and nested content out of the preview pipeline', () => {
    const root = document.createElement('div')
    root.innerHTML = [
      '<p><a href="https://example.com/a">https://example.com/a</a></p>',
      '<p>Read <a href="https://example.com/b">example</a></p>',
      '<p><a href="https://evil.example/path">https://example.com/mismatched</a></p>',
      '<blockquote><p><a href="https://example.com/c">https://example.com/c</a></p></blockquote>',
      '<p><a href="https://example.com/image"><img src="/a.png" alt=""></a></p>',
    ].join('')
    expect(findRenderedLinkCandidates(root).map(item => item.url)).toEqual(['https://example.com/a'])
  })

  test('stops at MAX_LINK_PREVIEWS so a long post cannot flood the page', () => {
    const root = document.createElement('div')
    const urls = Array.from({ length: MAX_LINK_PREVIEWS + 2 }, (_, index) => `https://example.com/${index}`)
    root.innerHTML = urls.map(url => `<p><a href="${url}">${url}</a></p>`).join('')

    const candidates = findRenderedLinkCandidates(root)
    // 保留的是文档顺序里最靠前的几张，第 MAX_LINK_PREVIEWS + 1 个段落原样留成普通链接。
    expect(candidates.map(item => item.url)).toEqual(urls.slice(0, MAX_LINK_PREVIEWS))
    expect(candidates).toHaveLength(MAX_LINK_PREVIEWS)
    expect(MAX_LINK_PREVIEWS).toBe(LINK_PREVIEW_REQUEST_LIMIT)
  })

  test('keeps the largest paragraph count that still fits one resolve request', () => {
    // 上限一旦大于单次请求能带的 URL 数，第 6 个及以后的段落就会拿到「有过请求、
    // 没进 preflight」的空数组而永不出卡。这条把「能出卡的段落数」与请求上限钉死。
    const root = document.createElement('div')
    const urls = Array.from({ length: LINK_PREVIEW_REQUEST_LIMIT }, (_, index) => `https://example.com/${index}`)
    root.innerHTML = urls.map(url => `<p><a href="${url}">${url}</a></p>`).join('')

    expect(findRenderedLinkCandidates(root).map(item => item.url)).toEqual(urls)
  })
})

describe('link preview usability', () => {
  test('accepts campus cards without a server title and rejects empty remote previews', () => {
    const base = {
      requestedUrl: 'https://a.example',
      kind: 'external' as const,
      status: 'ready' as const,
      url: 'https://a.example',
    }
    expect(usableLinkPreview({ ...base, title: 'Title' })).toBe(true)
    // 校园网卡片的展示名来自部署配置，可能为空；兜底文案由客户端 i18n 出，
    // 所以只给 campus 标记的卡片同样必须挂载。
    expect(usableLinkPreview({ ...base, campus: true })).toBe(true)
    // 非校园网卡片缺标题时仍然不成卡（不会渲染成一张没有标题的空卡）。
    expect(usableLinkPreview({ ...base })).toBe(false)
    expect(usableLinkPreview({ ...base, campus: true, status: 'timeout' })).toBe(false)
    expect(usableLinkPreview(undefined)).toBe(false)
  })
})

describe('external navigation classification', () => {
  afterEach(() => {
    cancelExternalLinkGuard()
    setExternalLinkGuardInternalOrigins([])
    sessionStorage.clear()
  })

  test('skips same-origin links and identifies an external hostname', () => {
    expect(externalDomain('/topics/1', 'https://forum.example.test/topic')).toBeNull()
    expect(externalDomain('https://docs.example.org/a', 'https://forum.example.test/topic')).toBe('docs.example.org')
  })

  test('skips configured first-party absolute origins without suffix matching', () => {
    setExternalLinkGuardInternalOrigins(['https://forum.example.test'])
    expect(externalDomain('https://forum.example.test/topics/1', 'https://alias.example.test/')).toBeNull()
    expect(externalDomain('https://forum.example.test.evil.invalid/', 'https://alias.example.test/')).toBe('forum.example.test.evil.invalid')
  })

  test('preserves a modified-click new-tab intent for the modal continuation', () => {
    const anchor = document.createElement('a')
    anchor.href = 'https://docs.example.org/a'
    const event = new MouseEvent('click', { cancelable: true, ctrlKey: true })
    expect(openExternalLinkGuard(anchor, event)).toBe(true)
    expect(event.defaultPrevented).toBe(true)
  })

  test('keys normal session trust by registrable domain and never bypasses a risk state', () => {
    expect(registrableDomain('docs.team.example.co.uk')).toBe('example.co.uk')
    trustExternalDomain('example.co.uk')

    const trusted = document.createElement('a')
    trusted.href = 'https://status.example.co.uk/a'
    expect(openExternalLinkGuard(trusted, new MouseEvent('click', { cancelable: true }))).toBe(false)

    trusted.dataset.externalRisk = 'suspicious'
    expect(openExternalLinkGuard(trusted, new MouseEvent('click', { cancelable: true }))).toBe(true)
  })

  test('keeps private-suffix tenants isolated', () => {
    expect(registrableDomain('alice.github.io')).toBe('alice.github.io')
    expect(registrableDomain('bob.github.io')).toBe('bob.github.io')
  })
})

describe('rendered link-preview enhancement', () => {
  afterEach(() => {
    apiMocks.resolveLinkPreviews.mockReset()
    vi.unstubAllGlobals()
    document.body.replaceChildren()
  })

  test('mounts one card idempotently and deduplicates the URL request', async () => {
    const url = 'https://enhancer.example/ready'
    vi.stubGlobal('IntersectionObserver', undefined)
    apiMocks.resolveLinkPreviews.mockResolvedValue([readyPreview(url)])
    const root = document.createElement('div')
    root.innerHTML = `<p><a href="${url}">${url}</a></p>`
    document.body.append(root)

    await enhanceLinkPreviews(root)
    await vi.waitFor(() => expect(root.querySelectorAll('[data-gf-link-preview-host]')).toHaveLength(1))
    await enhanceLinkPreviews(root)
    await vi.waitFor(() => expect(root.querySelectorAll('[data-gf-link-preview-host]')).toHaveLength(1))

    expect(apiMocks.resolveLinkPreviews).toHaveBeenCalledOnce()
    disposeLinkPreviews(root)
  })

  test('keeps the original link when resolution fails', async () => {
    const url = 'https://enhancer.example/failure'
    vi.stubGlobal('IntersectionObserver', undefined)
    apiMocks.resolveLinkPreviews.mockRejectedValue(new Error('offline'))
    const root = document.createElement('div')
    root.innerHTML = `<p><a href="${url}">${url}</a></p>`
    document.body.append(root)

    await enhanceLinkPreviews(root)
    await vi.waitFor(() => expect(root.querySelector('p')?.dataset.gfLinkPreview).toBe('failed'))

    expect(root.querySelector('p')?.hidden).toBe(false)
    expect(root.querySelector('a')?.getAttribute('href')).toBe(url)
    disposeLinkPreviews(root)
  })

  test('keeps the mounted card in place when the content was not replaced', async () => {
    const url = 'https://enhancer.example/stable'
    vi.stubGlobal('IntersectionObserver', undefined)
    apiMocks.resolveLinkPreviews.mockResolvedValue([readyPreview(url)])
    const root = document.createElement('div')
    root.innerHTML = `<p><a href="${url}">${url}</a></p>`
    document.body.append(root)

    await enhanceLinkPreviews(root)
    await vi.waitFor(() => expect(root.querySelectorAll('[data-gf-link-preview-host]')).toHaveLength(1))
    const host = root.querySelector('[data-gf-link-preview-host]')
    const paragraph = root.querySelector('p')

    apiMocks.resolveLinkPreviews.mockClear()
    await enhanceLinkPreviews(root)

    // 与正文无关的重渲染不得拆掉已有卡片：宿主节点、段落节点、隐藏态全部原地保留，
    // 也不该为此再打一次 preview 请求。
    expect(root.querySelector('[data-gf-link-preview-host]')).toBe(host)
    expect(root.querySelector('p')).toBe(paragraph)
    expect(paragraph?.hidden).toBe(true)
    expect(paragraph?.dataset.gfLinkPreview).toBe('ready')
    expect(apiMocks.resolveLinkPreviews).not.toHaveBeenCalled()
    disposeLinkPreviews(root)
  })

  test('rebuilds the card when v-html swaps in fresh nodes', async () => {
    const url = 'https://enhancer.example/replaced'
    vi.stubGlobal('IntersectionObserver', undefined)
    apiMocks.resolveLinkPreviews.mockResolvedValue([readyPreview(url)])
    const root = document.createElement('div')
    root.innerHTML = `<p><a href="${url}">${url}</a></p>`
    document.body.append(root)

    await enhanceLinkPreviews(root)
    await vi.waitFor(() => expect(root.querySelectorAll('[data-gf-link-preview-host]')).toHaveLength(1))
    const firstHost = root.querySelector('[data-gf-link-preview-host]')

    // 模拟 v-html 重写：正文文本相同，但整棵子树换成新节点，此时必须重建。
    root.innerHTML = `<p><a href="${url}">${url}</a></p>`
    await enhanceLinkPreviews(root)
    await vi.waitFor(() => expect(root.querySelectorAll('[data-gf-link-preview-host]')).toHaveLength(1))

    expect(root.querySelector('[data-gf-link-preview-host]')).not.toBe(firstHost)
    expect(root.querySelector('p')?.hidden).toBe(true)
    disposeLinkPreviews(root)
  })

  test('does not re-request a failed preview when nothing changed', async () => {
    const url = 'https://enhancer.example/offline'
    vi.stubGlobal('IntersectionObserver', undefined)
    apiMocks.resolveLinkPreviews.mockRejectedValue(new Error('offline'))
    const root = document.createElement('div')
    root.innerHTML = `<p><a href="${url}">${url}</a></p>`
    document.body.append(root)

    await enhanceLinkPreviews(root)
    await vi.waitFor(() => expect(root.querySelector('p')?.dataset.gfLinkPreview).toBe('failed'))

    apiMocks.resolveLinkPreviews.mockClear()
    await enhanceLinkPreviews(root)

    // 失败的候选同样属于「已接管」：内容没变就不重试，避免每次无关重渲染都打一次
    // 请求（服务端对失败结果有负缓存，重试基本是纯浪费）。
    expect(root.querySelector('p')?.dataset.gfLinkPreview).toBe('failed')
    expect(root.querySelector('p')?.hidden).toBe(false)
    expect(apiMocks.resolveLinkPreviews).not.toHaveBeenCalled()
    disposeLinkPreviews(root)
  })
})

describe('editor link-preview hint coordinator', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  test('ignores stale requests and never mutates Markdown', async () => {
    vi.useFakeTimers()
    const markdown = 'https://one.example'
    const changes: Array<string | null> = []
    let resolveFirst: ((value: LinkPreview[]) => void) | undefined
    const resolver = vi.fn((urls: readonly string[]) => new Promise<LinkPreview[]>((resolve) => {
      if (urls[0].includes('one.example')) resolveFirst = resolve
      else resolve([readyPreview(urls[0])])
    }))
    const controller = createLinkPreviewHintController({
      resolve: resolver,
      onChange: hint => changes.push(hint?.domains.join(',') ?? null),
    })

    controller.schedule(markdown)
    await vi.advanceTimersByTimeAsync(400)
    controller.schedule('https://two.example')
    await vi.advanceTimersByTimeAsync(400)
    resolveFirst?.([readyPreview(markdown)])
    await Promise.resolve()

    expect(changes.at(-1)).toBe('two.example')
    expect(markdown).toBe('https://one.example')
    controller.dispose()
  })

  test('waits until composition ends before resolving', async () => {
    vi.useFakeTimers()
    const resolver = vi.fn(async (urls: readonly string[]) => [readyPreview(urls[0])])
    const controller = createLinkPreviewHintController({ resolve: resolver, onChange: vi.fn() })
    controller.compositionStart()
    controller.schedule('https://example.com')
    await vi.advanceTimersByTimeAsync(800)
    expect(resolver).not.toHaveBeenCalled()
    controller.compositionEnd('https://example.com')
    await vi.advanceTimersByTimeAsync(400)
    expect(resolver).toHaveBeenCalledOnce()
    controller.dispose()
  })

  test('reports every card that will render, not just the first', async () => {
    vi.useFakeTimers()
    const markdown = ['https://one.example', '', 'https://two.example', '', 'https://three.example'].join('\n')
    const changes: Array<string[] | null> = []
    const controller = createLinkPreviewHintController({
      resolve: async () => [
        readyPreview('https://one.example'),
        // 抓取失败的候选取不到元数据，也不会成卡，因此不该出现在提示里。
        { requestedUrl: 'https://two.example', kind: 'external', status: 'timeout', url: 'https://two.example', displayHost: 'two.example' },
        readyPreview('https://three.example'),
      ],
      onChange: hint => changes.push(hint ? hint.domains : null),
    })

    controller.schedule(markdown)
    await vi.advanceTimersByTimeAsync(400)

    // 修复前只报第一个可用项（previews.find），作者看到 3 个链接却只被告知 1 个域名。
    expect(changes.at(-1)).toEqual(['one.example', 'three.example'])
    controller.dispose()
  })

  test('clears a published hint immediately when its URL changes', async () => {
    vi.useFakeTimers()
    const changes: Array<string[] | null> = []
    const controller = createLinkPreviewHintController({ resolve: async urls => [readyPreview(urls[0])], onChange: hint => changes.push(hint?.domains ?? null) })
    controller.schedule('https://old.example')
    await vi.advanceTimersByTimeAsync(400)
    expect(changes.at(-1)).toEqual(['old.example'])
    controller.schedule('https://new.example')
    expect(changes.at(-1)).toBeNull()
    controller.dispose()
  })

  test('empty composition result does not resurrect the previous URL', async () => {
    vi.useFakeTimers()
    const resolve = vi.fn(async urls => [readyPreview(urls[0])])
    const controller = createLinkPreviewHintController({ resolve, onChange: vi.fn() })
    controller.schedule('https://old.example')
    controller.compositionStart()
    controller.compositionEnd('')
    await vi.advanceTimersByTimeAsync(400)
    expect(resolve).not.toHaveBeenCalled()
    controller.dispose()
  })

  test('joins hint domains with the separator the locale provides', () => {
    expect(formatLinkPreviewHintDomains(['a.example', 'b.example'], '、')).toBe('a.example、b.example')
    expect(formatLinkPreviewHintDomains(['a.example', 'b.example'], ', ')).toBe('a.example, b.example')
    expect(formatLinkPreviewHintDomains(['a.example'], '、')).toBe('a.example')
    expect(formatLinkPreviewHintDomains([], '、')).toBe('')
  })
})
