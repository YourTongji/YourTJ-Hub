const ZERO_WIDTH_CHARACTERS = /[\u200b-\u200f\ufeff\u2060-\u2064\u2066-\u2069]/g

function normalizeSensitiveText(value: string): string {
  return value.normalize('NFKC').replace(ZERO_WIDTH_CHARACTERS, '').toLowerCase()
}

export function containsSensitiveText(value: string, words: string | readonly string[]): boolean {
  const haystack = normalizeSensitiveText(value)
  const queries = Array.isArray(words) ? words : [words]
  return queries.some((word) => {
    const query = normalizeSensitiveText(word.trim())
    return Boolean(query && haystack.includes(query))
  })
}
