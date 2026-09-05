// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { enhanceRenderedContent, contentEnhancementsDirective } from '../src/runtime/content-enhancements'
import {
  enhanceMermaid,
  MAX_MERMAID_TEXT_LENGTH,
  MERMAID_INITIALIZE_OPTIONS,
  nextMermaidDiagramId,
  resolveMermaidTheme,
} from '../src/runtime/content-enhancements/mermaid'
import type { MermaidAPI, MermaidRenderDeps } from '../src/runtime/content-enhancements/mermaid'

function diagramRoot(html: string): HTMLElement {
  const root = document.createElement('div')
  root.innerHTML = html
  document.body.appendChild(root)
  return root
}

function mermaidBlockHtml(source = 'graph TD\n  A --> B'): string {
  const escaped = source.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  return `<pre><code class="language-mermaid">${escaped}</code></pre>`
}

function fakeMermaid(): MermaidAPI {
  return {
    render: vi.fn(async () => ({ svg: '<svg xmlns="http://www.w3.org/2000/svg"></svg>' })),
  } as unknown as MermaidAPI
}

function renderDeps(overrides: Partial<MermaidRenderDeps> = {}): MermaidRenderDeps {
  return {
    loadMermaid: vi.fn(async () => fakeMermaid()),
    renderDiagram: vi.fn(async () => ({ svg: '<svg xmlns="http://www.w3.org/2000/svg"></svg>' })),
    resolveTheme: () => 'default',
    ...overrides,
  }
}

describe('enhanceMermaid', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  test('does not load the Mermaid chunk without diagram blocks', async () => {
    const root = diagramRoot('<pre><code>plain</code></pre><p>text</p>')
    const deps = renderDeps()

    await enhanceMermaid(root, deps)

    expect(deps.loadMermaid).not.toHaveBeenCalled()
    expect(deps.renderDiagram).not.toHaveBeenCalled()
  })

  test('renders a valid diagram and replaces the source block', async () => {
    const root = diagramRoot(mermaidBlockHtml())
    const deps = renderDeps()

    await enhanceMermaid(root, deps)

    expect(deps.renderDiagram).toHaveBeenCalledTimes(1)
    expect((deps.renderDiagram as ReturnType<typeof vi.fn>).mock.calls[0][2]).toBe('graph TD\n  A --> B')
    expect(root.querySelector('pre')).toBeNull()
    const diagram = root.querySelector('.gf-content-diagram-mermaid')
    expect(diagram).not.toBeNull()
    expect(diagram!.dataset.gfEnhanced).toBe('mermaid')
    expect(diagram!.innerHTML).toContain('<svg')
  })

  test('invokes mermaid bind functions on the rendered diagram', async () => {
    const root = diagramRoot(mermaidBlockHtml())
    const bindFunctions = vi.fn()
    const deps = renderDeps({ renderDiagram: vi.fn(async () => ({ svg: '<svg></svg>', bindFunctions })) })

    await enhanceMermaid(root, deps)

    expect(bindFunctions).toHaveBeenCalledTimes(1)
    expect(bindFunctions.mock.calls[0][0]).toBe(root.querySelector('.gf-content-diagram-mermaid'))
  })

  test('keeps the source block and marks the error when rendering fails', async () => {
    const root = diagramRoot(mermaidBlockHtml('graph TD\n  A --> '))
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const deps = renderDeps({
      renderDiagram: vi.fn(async () => {
        throw new Error('parse error')
      }),
    })

    await enhanceMermaid(root, deps)

    const pre = root.querySelector('pre')
    expect(pre).not.toBeNull()
    expect(pre!.dataset.gfEnhancement).toBe('mermaid-error')
    expect(pre!.querySelector('code')!.textContent).toBe('graph TD\n  A --> ')
    expect(root.querySelector('.gf-content-diagram')).toBeNull()
    warn.mockRestore()
  })

  test('does not render over-long input and never loads the chunk', async () => {
    const huge = 'x'.repeat(MAX_MERMAID_TEXT_LENGTH + 1)
    const root = diagramRoot(mermaidBlockHtml(huge))
    const deps = renderDeps()

    await enhanceMermaid(root, deps)

    expect(deps.loadMermaid).not.toHaveBeenCalled()
    expect(deps.renderDiagram).not.toHaveBeenCalled()
    const pre = root.querySelector('pre')!
    expect(pre.dataset.gfEnhancement).toBe('mermaid-error')
    expect(pre.textContent).toContain('xxx')
  })

  test('renders input exactly at the size boundary', async () => {
    const source = 'a'.repeat(MAX_MERMAID_TEXT_LENGTH)
    const root = diagramRoot(mermaidBlockHtml(source))
    const deps = renderDeps()

    await enhanceMermaid(root, deps)

    expect(deps.renderDiagram).toHaveBeenCalledTimes(1)
  })

  test('replaces the whole code-copy wrapper so no dangling copy button remains', async () => {
    const root = diagramRoot(`<div class="gf-code-block">${mermaidBlockHtml()}<button class="gf-code-copy">Copy</button></div>`)
    const deps = renderDeps()

    await enhanceMermaid(root, deps)

    expect(root.querySelector('.gf-code-block')).toBeNull()
    expect(root.querySelector('.gf-code-copy')).toBeNull()
    expect(root.querySelector('.gf-content-diagram-mermaid')).not.toBeNull()
  })

  test('skips blocks already claimed by a previous pass', async () => {
    const root = diagramRoot(mermaidBlockHtml())
    root.querySelector('pre')!.dataset.gfEnhancement = 'mermaid'
    const deps = renderDeps()

    await enhanceMermaid(root, deps)

    expect(deps.loadMermaid).not.toHaveBeenCalled()
  })
})

describe('resolveMermaidTheme', () => {
  afterEach(() => {
    delete document.documentElement.dataset.theme
  })

  test('uses dark when the site theme is dark', () => {
    document.documentElement.dataset.theme = 'gf-dark'
    expect(resolveMermaidTheme()).toBe('dark')
  })

  test('uses the default theme for light or unknown state', () => {
    document.documentElement.dataset.theme = 'gf-light'
    expect(resolveMermaidTheme()).toBe('default')

    delete document.documentElement.dataset.theme
    expect(resolveMermaidTheme()).toBe('default')
  })
})

describe('loadMermaid singleton behavior', () => {
  afterEach(() => {
    delete document.documentElement.dataset.theme
    vi.doUnmock('mermaid')
    vi.resetModules()
  })

  test('re-initializes the shared instance when the site theme flips during the SPA lifetime', async () => {
    vi.resetModules()
    vi.doMock('mermaid', () => ({
      default: { initialize: vi.fn(), render: vi.fn() },
    }))
    const mermaidModule = await import('mermaid')
    const { loadMermaid } = await import('../src/runtime/content-enhancements/mermaid')

    document.documentElement.dataset.theme = 'gf-light'
    const first = await loadMermaid()
    expect(mermaidModule.default.initialize).toHaveBeenCalledTimes(1)
    expect(mermaidModule.default.initialize).toHaveBeenLastCalledWith(expect.objectContaining({ theme: 'default' }))

    document.documentElement.dataset.theme = 'gf-dark'
    const second = await loadMermaid()
    expect(second).toBe(first)
    expect(mermaidModule.default.initialize).toHaveBeenCalledTimes(2)
    expect(mermaidModule.default.initialize).toHaveBeenLastCalledWith(expect.objectContaining({ theme: 'dark' }))

    // 同一主题下重复调用不重复 initialize。
    await loadMermaid()
    expect(mermaidModule.default.initialize).toHaveBeenCalledTimes(2)
  })

  test('resets the cached promise when the chunk fails to load, so a later call can retry', async () => {
    vi.resetModules()
    let attempt = 0
    vi.doMock('mermaid', () => {
      if (attempt === 0) throw new Error('chunk fetch failed')
      return { default: { initialize: vi.fn(), render: vi.fn() } }
    })
    const { loadMermaid } = await import('../src/runtime/content-enhancements/mermaid')

    // vitest 会包装 mock factory 抛出的错误，这里只断言「首次加载失败」这一行为。
    await expect(loadMermaid()).rejects.toThrow()
    attempt = 1
    await expect(loadMermaid()).resolves.toBeTruthy()
  })
})

describe('shared Mermaid configuration', () => {
  test('uses strict, non-self-starting Mermaid settings', () => {
    expect(MERMAID_INITIALIZE_OPTIONS).toMatchObject({
      startOnLoad: false,
      securityLevel: 'strict',
      suppressErrorRendering: true,
    })
  })

  test('issues unique diagram ids across render surfaces', () => {
    expect(nextMermaidDiagramId()).not.toBe(nextMermaidDiagramId())
  })
})

describe('content enhancement pipeline', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  test('enhanceRenderedContent leaves plain content untouched', async () => {
    const root = diagramRoot('<p>no diagrams here</p><pre><code>plain</code></pre>')

    await expect(enhanceRenderedContent(root)).resolves.toBeUndefined()

    expect(root.querySelector('.gf-content-diagram')).toBeNull()
  })

  test('directive mounted hook runs the pipeline without errors', async () => {
    const root = diagramRoot('<p>plain</p>')

    await (contentEnhancementsDirective.mounted as (el: HTMLElement) => void)(root)

    expect(root.querySelector('.gf-content-diagram')).toBeNull()
  })
})