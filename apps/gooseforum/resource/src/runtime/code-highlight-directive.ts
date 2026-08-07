import type { ObjectDirective } from 'vue'

type CodeHighlighter = Pick<typeof import('./code-highlighting'), 'highlightCode'>
type HighlighterLoader = () => Promise<CodeHighlighter>

const languageClassPrefix = 'language-'

export function extractCodeLanguage(element: Pick<HTMLElement, 'className'>): string | null {
  const languageClass = element.className
    .split(/\s+/)
    .find(className => className.startsWith(languageClassPrefix) && className.length > languageClassPrefix.length)
  return languageClass?.slice(languageClassPrefix.length) || null
}

export async function enhanceCodeBlocks(
  root: ParentNode,
  loadHighlighter: HighlighterLoader = () => import('./code-highlighting'),
) {
  const candidates = Array.from(root.querySelectorAll<HTMLElement>('pre > code[class*="language-"]'))
    .map(element => ({ element, language: extractCodeLanguage(element) }))
    .filter((candidate): candidate is { element: HTMLElement, language: string } => (
      Boolean(candidate.language) && !candidate.element.dataset.codeHighlight
    ))

  if (candidates.length === 0) return

  for (const { element } of candidates) element.dataset.codeHighlight = 'loading'

  let highlighter: CodeHighlighter
  try {
    highlighter = await loadHighlighter()
  }
  catch {
    for (const { element } of candidates) delete element.dataset.codeHighlight
    return
  }

  for (const { element, language } of candidates) {
    const highlighted = highlighter.highlightCode(element.textContent || '', language)
    if (highlighted === null) {
      element.dataset.codeHighlight = 'unsupported'
      continue
    }

    element.innerHTML = highlighted
    element.classList.add('hljs')
    element.dataset.codeHighlight = 'done'
  }
}

function runCodeHighlighting(element: HTMLElement) {
  void enhanceCodeBlocks(element)
}

export const codeHighlightDirective: ObjectDirective<HTMLElement> = {
  mounted: runCodeHighlighting,
  updated: runCodeHighlighting,
}
