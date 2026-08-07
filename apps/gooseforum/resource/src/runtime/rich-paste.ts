import TurndownService from 'turndown'

const turndown = new TurndownService({
  headingStyle: 'atx',
  bulletListMarker: '-',
  codeBlockStyle: 'fenced',
})

export function htmlToMarkdown(html: string) {
  return turndown.turndown(html).trim()
}

export function markdownFromClipboard(data: DataTransfer | null) {
  const html = data?.getData('text/html') || ''
  if (!html.trim()) return ''
  return htmlToMarkdown(html)
}

export function hasUnsupportedVisualMarkdown(markdown: string) {
  const hasTaskList = /^\s*(?:>\s*)*(?:[-+*]|\d+[.)])\s+\[[ xX]\]\s+/m.test(markdown)
  const hasFootnoteReference = /\[\^[^\]]+\](?!\()/.test(markdown)
  const hasFootnoteDefinition = /^[ \t]{0,3}(?:>\s*)*\[\^[^\]]+\]:/m.test(markdown)
  return hasTaskList || hasFootnoteReference || hasFootnoteDefinition
}
