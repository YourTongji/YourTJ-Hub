import type { ObjectDirective } from 'vue'
import { extractMathSegments } from './math-segments'

/**
 * Vue directive that renders Markdown math ($...$ / $$...$$) as KaTeX.
 *
 * Mirrors the code-highlight directive: content is decorated in place, KaTeX
 * is loaded lazily only after a marker is detected, and a load failure leaves
 * the original rendered HTML unchanged. Text nodes inside code blocks and
 * previously-rendered KaTeX output are skipped.
 */

type MathEnhancerModule = Pick<
  typeof import('./math-rendering'),
  'renderMath' | 'splitIntoMathParts' | 'buildFragment'
>

export interface TextNodeLike {
  readonly data: string
  replaceWith(node: Node | DocumentFragment): void
}

const SKIP_SELECTOR = 'pre, code, script, style, textarea, .katex, .katex-display, .katex-error'

export function collectTextNodes(root: ParentNode): TextNodeLike[] {
  const textNodes: TextNodeLike[] = []
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      const parent = (node as Text).parentElement
      if (!parent || parent.closest(SKIP_SELECTOR)) return NodeFilter.FILTER_REJECT
      return NodeFilter.FILTER_ACCEPT
    },
  })
  let current: Node | null
  while ((current = walker.nextNode())) textNodes.push(current as Text)
  return textNodes
}

export async function enhanceMathText(
  root: ParentNode,
  loadEnhancer: () => Promise<MathEnhancerModule> = () => import('./math-rendering'),
  collectTextNodesFn: (root: ParentNode) => TextNodeLike[] = collectTextNodes,
): Promise<boolean> {
  const textNodes = collectTextNodesFn(root)
  if (!textNodes.some((node) => extractMathSegments(node.data).length > 0)) return false

  let enhancer: MathEnhancerModule
  try {
    enhancer = await loadEnhancer()
  } catch {
    return false
  }

  let changed = false
  for (const node of textNodes) {
    const parts = enhancer.splitIntoMathParts(node.data, enhancer.renderMath)
    if (parts.length === 1 && parts[0].type === 'text' && parts[0].value === node.data) continue
    node.replaceWith(enhancer.buildFragment(parts))
    changed = true
  }
  return changed
}

function runMathRendering(element: HTMLElement) {
  void enhanceMathText(element)
}

export const mathRenderDirective: ObjectDirective<HTMLElement> = {
  mounted: runMathRendering,
  updated: runMathRendering,
}
