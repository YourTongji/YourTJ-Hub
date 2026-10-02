export interface BadgeStyleItem {
  color: string
  level: string
  iconUrl?: string
  name: string
  description?: string
}

export function badgeClass(color: string, level: string) {
  if (color === 'blue') return 'bg-blue-100 text-blue-700 ring-blue-200'
  if (color === 'emerald') return 'bg-emerald-100 text-emerald-700 ring-emerald-200'
  if (color === 'teal') return 'bg-teal-100 text-teal-700 ring-teal-200'
  if (color === 'sky') return 'bg-sky-100 text-sky-700 ring-sky-200'
  if (color === 'cyan') return 'bg-cyan-100 text-cyan-700 ring-cyan-200'
  if (color === 'rose') return 'bg-rose-100 text-rose-700 ring-rose-200'
  if (color === 'violet') return 'bg-violet-100 text-violet-700 ring-violet-200'
  if (color === 'purple') return 'bg-purple-100 text-purple-700 ring-purple-200'
  if (color === 'fuchsia') return 'bg-fuchsia-100 text-fuchsia-700 ring-fuchsia-200'
  if (color === 'indigo') return 'bg-indigo-100 text-indigo-700 ring-indigo-200'
  if (color === 'amber') return 'bg-amber-100 text-amber-700 ring-amber-200'
  if (color === 'orange') return 'bg-orange-100 text-orange-700 ring-orange-200'
  if (color === 'yellow') return 'bg-yellow-100 text-yellow-700 ring-yellow-200'
  if (color === 'slate') return 'bg-slate-100 text-slate-700 ring-slate-200'
  if (level === 'gold') return 'bg-amber-100 text-amber-700 ring-amber-200'
  if (level === 'special') return 'bg-indigo-100 text-indigo-700 ring-indigo-200'
  return 'bg-blue-100 text-blue-700 ring-blue-200'
}

export function badgeIconURL(badge: BadgeStyleItem) {
  return badge.iconUrl || '/static/badges/contributor.svg'
}

/**
 * 内置徽章图形（/static/badges/）已按统一光学尺寸自带边距（docs/decisions/0054），
 * 可以放大填充容器；自定义徽章的图形（如外链品牌 logo）通常铺满画布，需保持原有内边距。
 * 判定口径与移动端 _medallionArtworkSize 一致：只认相对路径（任何 host 的绝对 URL
 * 都算自定义图形）；空 URL 经 badgeIconURL 回退到内置 contributor 图形，按内置取景。
 */
export function isSystemBadgeArtwork(badge: BadgeStyleItem) {
  return badgeIconURL(badge).startsWith('/static/badges/')
}

export function badgeTooltip(badge: BadgeStyleItem) {
  return badge.description ? `${badge.name}：${badge.description}` : badge.name
}
