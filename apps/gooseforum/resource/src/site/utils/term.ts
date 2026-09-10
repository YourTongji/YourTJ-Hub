// 学期代码工具：统一将形如 "2025-2026-2" 的学期码缩写为 "25春/26秋" 等短格式。
// 1=秋（第一学期），2=春（第二学期）；按 startYear*10+semester 比较新旧。
export function termSortKey(term: string): number {
  const m = /^(\d{4})-(\d{4})-([12])$/.exec(term.trim())
  return m ? parseInt(m[1], 10) * 10 + parseInt(m[3], 10) : -1
}

export function shortTerm(term: string): string {
  const m = /^(\d{4})-(\d{4})-([12])$/.exec(term.trim())
  return m ? `${m[1].slice(2)}${m[3] === '1' ? '秋' : '春'}` : term
}

export function sortedRecentTerms(terms: string[] | undefined): string[] {
  // 标准学期码按时间降序（最新在前）；非标准码（如「其他」）恒置末尾，不参与排序。
  return [...(terms ?? [])].sort((a, b) => {
    const ka = termSortKey(a)
    const kb = termSortKey(b)
    if (ka < 0 && kb < 0) return 0
    if (ka < 0) return 1
    if (kb < 0) return -1
    return kb - ka
  })
}
