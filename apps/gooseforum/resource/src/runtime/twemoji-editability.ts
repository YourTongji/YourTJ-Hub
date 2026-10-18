/** The compiler and runtime use the same conservative contenteditable value rule. */
export function isContentEditableValue(value: string | null | undefined): boolean {
  return value != null && value.toLowerCase() !== 'false'
}

export const TWEMOJI_TEXT_FREE_TAGS = new Set([
  'code',
  'iframe',
  'kbd',
  'math',
  'option',
  'pre',
  'samp',
  'script',
  'select',
  'style',
  'svg',
  'textarea',
  'title',
])
