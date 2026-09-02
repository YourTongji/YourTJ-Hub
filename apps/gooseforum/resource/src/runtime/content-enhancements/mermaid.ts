import type { ContentEnhancer } from '@/runtime/content-enhancements'

/**
 * Mermaid diagram rendering as a client-side content enhancement.
 *
 * Fenced code blocks with `language-mermaid` are detected inside already
 * rendered HTML and replaced with rendered SVG. The Mermaid chunk (JS) is
 * imported lazily only after at least one renderable block is found, so pages
 * without diagrams never pay its load cost. Rendering uses strict Mermaid
 * settings (securityLevel `strict`, suppressErrorRendering), and every failure
 * — including over-long input, guarded before the chunk is loaded — keeps the
 * original source code block in place with a `data-gf-enhancement` marker.
 * Server-rendered HTML and the stored Markdown are never modified.
 *
 * The theme (light/dark) is resolved from `html[data-theme]` at the moment the
 * Mermaid chunk first loads; diagrams keep the theme they were rendered with
 * until the page is re-rendered.
 */

const mermaidSelector = 'pre > code.language-mermaid'

/** Upper bound for a single diagram source; longer blocks are left as code. */
export const MAX_MERMAID_TEXT_LENGTH = 50_000

/** Strict-mode initialization shared by every Mermaid render surface. */
export const MERMAID_INITIALIZE_OPTIONS = {
  startOnLoad: false,
  securityLevel: 'strict',
  suppressErrorRendering: true,
  fontFamily: 'inherit',
} as const

export type MermaidAPI = typeof import('mermaid').default

export interface MermaidRenderResult {
  svg: string
  bindFunctions?: (element: Element) => void
}

export interface MermaidRenderDeps {
  loadMermaid: () => Promise<MermaidAPI>
  renderDiagram: (mermaid: MermaidAPI, id: string, source: string) => Promise<MermaidRenderResult>
  resolveTheme: () => 'dark' | 'default'
}

let diagramSequence = 0
let mermaidPromise: Promise<MermaidAPI> | undefined
let renderQueue = Promise.resolve()

/** 站点深浅色由 html[data-theme] 决定（site-theme.ts applyTheme）。 */
export function resolveMermaidTheme(): 'dark' | 'default' {
  return document.documentElement.dataset.theme === 'gf-dark' ? 'dark' : 'default'
}

/** 全站共享的图表 id 序列，保证同一页面内多个渲染面 id 不冲突。 */
export function nextMermaidDiagramId(prefix = 'gf-mermaid') {
  return `${prefix}-${++diagramSequence}`
}

export function loadMermaid(resolveTheme: () => 'dark' | 'default' = resolveMermaidTheme): Promise<MermaidAPI> {
  if (!mermaidPromise) {
    mermaidPromise = import('mermaid').then(({ default: mermaid }) => {
      mermaid.initialize({ ...MERMAID_INITIALIZE_OPTIONS, theme: resolveTheme() })
      return mermaid
    })
  }
  return mermaidPromise
}

/** 串行渲染队列：mermaid.render 在同一线程内并发执行会互相干扰。 */
function renderDiagram(mermaid: MermaidAPI, id: string, source: string): Promise<MermaidRenderResult> {
  const result = renderQueue.then(() => mermaid.render(id, source))
  renderQueue = result.then(() => undefined, () => undefined)
  return result
}

export async function enhanceMermaid(
  root: HTMLElement,
  deps: MermaidRenderDeps = { loadMermaid, renderDiagram, resolveTheme: resolveMermaidTheme },
): Promise<void> {
  const blocks = Array.from(root.querySelectorAll<HTMLElement>(mermaidSelector))
    .filter((block) => block.parentElement?.dataset.gfEnhancement !== 'mermaid')
  if (blocks.length === 0) return

  // 超大输入不尝试渲染（也不加载 Mermaid chunk）：标记错误并保留原始代码块。
  for (const block of blocks) {
    const sourceElement = block.parentElement
    if (!sourceElement || sourceElement.dataset.gfEnhancement) continue
    if ((block.textContent?.length ?? 0) > MAX_MERMAID_TEXT_LENGTH) {
      sourceElement.dataset.gfEnhancement = 'mermaid-error'
    }
  }

  const pending = blocks.filter((block) => {
    const sourceElement = block.parentElement
    return sourceElement && !sourceElement.dataset.gfEnhancement
  })
  if (pending.length === 0) return

  const mermaid = await deps.loadMermaid()
  for (const block of pending) {
    const sourceElement = block.parentElement
    if (!sourceElement || !sourceElement.isConnected || sourceElement.dataset.gfEnhancement) continue

    sourceElement.dataset.gfEnhancement = 'mermaid'
    try {
      const id = nextMermaidDiagramId()
      const { svg, bindFunctions } = await deps.renderDiagram(mermaid, id, block.textContent || '')
      if (!sourceElement.isConnected || sourceElement.dataset.gfEnhancement !== 'mermaid') continue

      const diagram = document.createElement('div')
      diagram.className = 'gf-content-diagram gf-content-diagram-mermaid'
      diagram.dataset.gfEnhanced = 'mermaid'
      diagram.innerHTML = svg
      bindFunctions?.(diagram)
      sourceElement.replaceWith(diagram)
    } catch (error) {
      sourceElement.dataset.gfEnhancement = 'mermaid-error'
      console.warn('Unable to render Mermaid diagram; preserving its source code.', error)
    }
  }
}

export const mermaidContentEnhancer: ContentEnhancer = {
  name: 'Mermaid',
  enhance: enhanceMermaid,
}