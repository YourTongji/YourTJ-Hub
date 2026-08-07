import hljs from 'highlight.js/lib/common'

export function highlightCode(source: string, language: string): string | null {
  const normalizedLanguage = language.trim().toLowerCase()
  if (!normalizedLanguage || !hljs.getLanguage(normalizedLanguage)) return null

  try {
    return hljs.highlight(source, {
      language: normalizedLanguage,
      ignoreIllegals: true,
    }).value
  }
  catch {
    return null
  }
}
