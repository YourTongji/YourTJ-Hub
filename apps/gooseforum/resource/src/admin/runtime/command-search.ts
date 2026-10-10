import type { NavGroup, NavItem } from '@/admin/runtime/navigation'

export interface CommandEntry {
  item: NavItem
  group: string
}

const normalize = (value: string) => value.normalize('NFKC').toLocaleLowerCase().replace(/\s+/g, '')

// English words from the route so `users`, `mail` or `wiki` work in every UI language.
function pathWords(url: string): string[] {
  const parts = url.replace(/^\/admin\/?/, '').split(/[/-]/).filter(Boolean)
  return parts.length ? [...parts, parts.join('')] : ['dashboard', 'home', 'overview']
}

function isSubsequence(query: string, text: string) {
  let at = 0
  for (const char of text) if (char === query[at] && ++at === query.length) return true
  return query.length === 0
}

export function commandEntries(groups: NavGroup[]): CommandEntry[] {
  return groups.flatMap((group) => group.items.filter((item) => !item.external).map((item) => ({ item, group: group.title })))
}

/**
 * Rank admin pages for a query: title prefix, title substring, title characters in order
 * (so 用管 finds 用户管理), route word, then group name. Ties keep sidebar order.
 */
export function searchCommands(entries: CommandEntry[], query: string): CommandEntry[] {
  const q = normalize(query)
  if (!q) return entries
  return entries
    .map((entry, index) => {
      const title = normalize(entry.item.title)
      const score = title.startsWith(q)
        ? 0
        : title.includes(q)
          ? 1
          : isSubsequence(q, title)
            ? 2
            : pathWords(entry.item.url).some((word) => word.startsWith(q))
              ? 3
              : normalize(entry.group).includes(q)
                ? 4
                : -1
      return { entry, index, score }
    })
    .filter((row) => row.score >= 0)
    .sort((a, b) => a.score - b.score || a.index - b.index)
    .map((row) => row.entry)
}
