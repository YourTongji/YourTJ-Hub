import type { CampusData, CampusFeature } from './catalog'

export interface CampusMapTarget {
  campusId: string
  featureId: string
}

export interface OfficialMapLocation {
  raw: string
  building: string
  room: string
  condition: string
  /** Explicit campus context also applies to subsequent entries until replaced. */
  campusIds?: string[]
  alternative?: boolean
  target?: CampusMapTarget
}

const campusMarkers: [string, RegExp][] = [
  ['siping', /四平/],
  ['jiading', /嘉定/],
  ['huxi', /沪西/],
  ['hubei', /沪北/],
  ['zhangjiang', /张江/],
  ['lingang', /临港/],
]

export function officialCampusId(value: string): string | undefined {
  const matches = explicitCampusIds(value)
  return matches.length === 1 ? matches[0] : undefined
}

function explicitCampusIds(value: string): string[] {
  return campusMarkers.filter(([, marker]) => marker.test(value)).map(([id]) => id)
}

function withoutCampusPrefix(value: string, campus = ''): string {
  let result = value.trim()
  if (campus && result.startsWith(campus)) result = result.slice(campus.length).trim()
  return result.replace(/^(?:同济大学)?(?:四平(?:路)?|嘉定|沪西|沪北|张江|临港)(?:校区|基地)?\s*/u, '').trim()
}

export function parseOfficialLocation(value: string, campus = ''): { building: string; room: string } | null {
  const locations = parseOfficialLocations(value, campus)
  const location = locations[0]
  return locations.length === 1 && location?.room
    ? { building: location.building, room: location.room }
    : null
}

function normalize(value: string): string {
  return value.normalize('NFKC').toLocaleLowerCase().replace(/[\s\p{P}\p{S}]/gu, '')
}

function aliases(feature: CampusFeature): string[] {
  const properties = feature.properties
  const name = properties.name?.trim() ?? ''
  const values = [name, properties.alt_name ?? '', properties.short_name ?? '']
  const alias = name.match(/[（(]([^（）()]+)[）)]/u)?.[1]?.trim()
  if (alias && /^(?:[a-z]\s*)?(?:楼|馆|栋)$|^[a-z]楼$/iu.test(alias)) {
    values.push(alias)
    if (/^[a-z]楼$/iu.test(alias)) values.push(alias.slice(0, -1))
  }
  if (name) values.push(name.replace(/[（(][^（）()]*[）)]/gu, '').trim())
  return values.flatMap((value) => value.split(/[、,，;；]/u))
    .flatMap((value) => [value.trim(), withoutCampusPrefix(value).replace(/^同济大学/u, '').trim()])
    .filter((value) => value && !/^\d+$/u.test(value))
}

function confirmedTarget(campusId: string, building: string, room: string): CampusMapTarget | undefined {
  if (campusId === 'siping') {
    if (['北', '北楼', '教学北楼'].includes(building) && (room || building !== '北'))
      return { campusId, featureId: 'way/183383474' }
    if (['南', '南楼', '教学南楼'].includes(building) && (room || building !== '南'))
      return { campusId, featureId: 'way/183383472' }
  }
  if (campusId === 'jiading' && ['济事楼', '济事南楼', '济事北楼', '济事楼（软件学院）'].includes(building))
    return { campusId, featureId: 'way/135405205' }
  return undefined
}

// Descriptions are retained verbatim; a building-level match is not an indoor coordinate.
const roomDescription = /^(?:阶\s*\d{1,3}|[A-Za-z]{0,3}\d{1,4}[A-Za-z]?(?:[-~～至][A-Za-z]?\d{1,4}[A-Za-z]?)?(?:\s*(?:室|小教室|中教室|大教室|教室|实验室|机房|阶梯教室|智慧教室|洁净室|楼)(?:[（(][^（）()]*[）)])?)?|[一二三四五六七八九十\d]+楼(?:.*)?|(?:专业|苹果)?(?:实验室|教室|机房|画室|小影院|报告厅|体操房|篮球馆|游泳馆|羽毛球场|乒乓馆|力量房|多功能房|健身房)(?:\s*[A-Za-z]?\d{1,4}[A-Za-z]?)?)$/u
const conditionPrefix = /^(?:上课)?(?:第?\d+(?:[-~～至]\d+)?周|前\d+周|后\d+周|单周|双周|周[一二三四五六日天]|【[^】]+】|其余周数)/u
const roomContinuation = /^(?:阶\s*\d{1,3}|[A-Za-z]{0,3}\d{1,4}[A-Za-z]?(?:[-~～至][A-Za-z]?\d{1,4}[A-Za-z]?)?)(?:\s*(?:室|教室|实验室|机房|楼))?$/u

function splitLocationText(value: string): { parts: string[]; alternative: boolean } {
  const parts: string[] = []
  let alternative = false
  let start = 0
  let depth = 0
  for (let index = 0; index < value.length; index++) {
    const char = value[index]!
    if ('（([【'.includes(char)) depth++
    if ('）)]】'.includes(char)) depth = Math.max(0, depth - 1)
    // Conjunctions can be part of a name (e.g. 衷和楼); split only after a destination/room ending.
    const conjunction = /[和或]/u.test(char) && /[\dA-Za-z）)楼馆场厅室房河]$|中心$|大厦$/u.test(value.slice(start, index).trim())
    if (depth === 0 && (/[、,，;；+\\/]/u.test(char) || conjunction)) {
      if (char === '或') alternative = true
      parts.push(value.slice(start, index).trim())
      start = index + 1
    }
  }
  parts.push(value.slice(start).trim())
  return { parts: parts.filter(Boolean), alternative }
}

function parseBuilding(location: string): { building: string; room: string } | null {
  const match = location.match(/^(.+?(?:楼|馆|中心|大厦|栋|报告厅|教室|机房|实验室|专教|教)(?:[（(][^（）()]+[）)])?|[北南东西]|文|中法|[A-Za-z]{1,3}|机房|实验室)\s*(.+)$/u)
  if (!match || !roomDescription.test(match[2]!.trim())) return null
  return { building: match[1]!, room: match[2]!.trim().replace(/\s+室$/u, '') }
}

function parseLocations(value: string, campus: string, refine?: (location: OfficialMapLocation, text: string) => void): OfficialMapLocation[] {
  if (!value.trim()) return []
  // Announcements and remote lessons do not identify a campus destination.
  const { parts, alternative } = /线上|在线|Canvas|关注.*通知/iu.test(value)
    ? { parts: [value.trim()], alternative: false } : splitLocationText(value)
  const locations: OfficialMapLocation[] = []
  let previous: OfficialMapLocation | undefined
  let pendingCondition = ''
  let campusIds: string[] = []
  for (const raw of parts) {
    const prefix = raw.match(conditionPrefix)?.[0] ?? ''
    let text = raw.slice(prefix.length).replace(/^\s*[:：]\s*/u, '').trim()
    const condition = prefix || pendingCondition
    pendingCondition = ''
    if (!text && prefix) { pendingCondition = prefix; continue }
    const explicit = explicitCampusIds(raw)
    if (explicit.length) campusIds = explicit
    text = withoutCampusPrefix(text, campus)
    const parsed = parseBuilding(text)
    const continuation = roomContinuation.test(text)
    const inherited = continuation && (previous?.room || previous?.target)
    const unknownPrefix = continuation && previous && !previous.room && !previous.target
      ? previous.building.match(/^(.+?)[A-Za-z]{0,3}\d{1,4}[A-Za-z]?$/u)?.[1] : undefined
    const buildingContinuation = /^[A-Za-z]楼$/u.test(text) && previous && /[A-Za-z]楼$/u.test(previous.building)
      ? previous.building.replace(/[A-Za-z]楼$/u, text) : undefined
    const location: OfficialMapLocation = {
      raw,
      building: inherited ? previous!.building : unknownPrefix ?? buildingContinuation ?? parsed?.building ?? text,
      room: inherited || unknownPrefix ? text : parsed?.room ?? '',
      condition: condition || (inherited || unknownPrefix || buildingContinuation ? previous!.condition : ''),
      ...(campusIds.length ? { campusIds } : {}),
      ...(alternative ? { alternative: true } : {}),
    }
    // Refine before carrying context forward, so a whole map name remains the parent of its room list.
    refine?.(location, inherited || unknownPrefix || buildingContinuation ? location.building + location.room : text)
    locations.push(location)
    previous = location
  }
  if (pendingCondition) locations.push({ raw: pendingCondition, building: '', room: '', condition: pendingCondition })
  return locations
}

export function parseOfficialLocations(value: string, campus = ''): OfficialMapLocation[] {
  return parseLocations(value, campus)
}

function prefixEnd(value: string, key: string): number | undefined {
  let prefix = ''
  for (let index = 0; index < value.length; index++) {
    prefix += normalize(value[index]!)
    if (!key.startsWith(prefix)) return undefined
    if (prefix === key) {
      while (/[\s）)]/u.test(value[index + 1] ?? '') && index + 1 < value.length) index++
      return index + 1
    }
  }
  return undefined
}

export function officialLocationTargets(campus: string, value: string, data?: CampusData | null): OfficialMapLocation[] {
  const campusId = officialCampusId(campus)
  const names = (data?.features ?? []).filter((feature) => feature.properties.campus &&
    ['academic', 'library', 'place', 'sport'].includes(feature.properties.category))
    .flatMap((feature) => aliases(feature).map((name) => ({ feature, name, key: normalize(name) })))
  return parseLocations(value, campus, (location, raw) => {
    if (!campusId || location.campusIds?.some(id => id !== campusId) || /线上|在线|Canvas/iu.test(location.raw)) return
    // Exact whole names win over a room-looking name; only recognized suffixes permit prefix matching.
    const exact = names.filter((entry) => entry.key === normalize(raw))
    if (exact.length) { location.building = raw; location.room = '' }
    else {
      const prefixes = names.flatMap((entry) => {
        const end = prefixEnd(raw, entry.key)
        const suffix = end === undefined ? '' : raw.slice(end).trim()
        return suffix && roomDescription.test(suffix) ? [{ ...entry, suffix }] : []
      }).sort((a, b) => b.key.length - a.key.length)
      const longest = prefixes[0]
      if (longest) { location.building = longest.name; location.room = longest.suffix }
    }
    location.target = confirmedTarget(campusId, location.building, location.room)
    if (location.target) return
    const ids = new Set(names.filter((entry) => entry.key === normalize(location.building))
      .map((entry) => entry.feature.id).filter((id) => id != null).map(String))
    if (ids.size === 1) location.target = { campusId, featureId: [...ids][0]! }
  })
}

export function officialLocationTarget(campus: string, value: string, data?: CampusData | null): CampusMapTarget | undefined {
  const locations = officialLocationTargets(campus, value, data)
  if (!locations.length || locations.some((location) => !location.target)) return undefined
  const targets = new Map(locations.map((location) => [JSON.stringify(location.target), location.target!]))
  return targets.size === 1 ? [...targets.values()][0] : undefined
}

export function officialLocationApplies(location: OfficialMapLocation, context: { week: number; day: number }): boolean {
  if (location.alternative) return false
  const condition = location.condition.replace(/^上课/u, '')
  if (!condition) return true
  if (condition === '单周') return context.week % 2 === 1
  if (condition === '双周') return context.week % 2 === 0
  if (/^周[一二三四五六日天]$/u.test(condition)) return ('一二三四五六日'.indexOf(condition[1]!.replace('天', '日')) + 1) === context.day
  const weeks = condition.match(/^第?(\d+)(?:[-~～至](\d+))?周$/u)
  if (weeks) return context.week >= Number(weeks[1]) && context.week <= Number(weeks[2] ?? weeks[1])
  const first = condition.match(/^前(\d+)周$/u)
  if (first) return context.week <= Number(first[1])
  // Last-N weeks, dates and unspecified alternatives require a source-defined range.
  return false
}
