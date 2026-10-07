import type { CampusData } from './catalog'
import overrides from './data/locations/overrides.json'
import { missingExtraction, parseStableLocation, placeCatalog } from './deterministic-location'
import type { LocationExtraction, LocationHint, LocationKind, LocationLookupContext, LocationOverride, LocationOverrideConfig, LocationRelation } from './location-types'
export type { LocationLookupContext } from './location-types'

export interface CampusMapTarget {
  campusId: string
  featureId: string
}
export interface OfficialMapLocation {
  raw: string
  building: string
  room: string
  condition: string
  kind?: LocationKind
  relation?: LocationRelation
  time?: string[] | null
  conditions?: string[]
  address?: string | null
  campusText?: string | null
  unassignedConditions?: string[]
  needsReview?: boolean
  reviewPending?: boolean
  hint?: LocationHint
  alternative?: boolean
  target?: CampusMapTarget
}

const campusNames: Record<string, string> = {
  '四平': 'siping', '四平路': 'siping', '四平校区': 'siping', '四平路校区': 'siping',
  '嘉定': 'jiading', '嘉定校区': 'jiading', '沪西': 'huxi', '沪西校区': 'huxi',
  '沪北': 'hubei', '沪北校区': 'hubei', '张江': 'zhangjiang', '张江校区': 'zhangjiang',
  '临港': 'lingang', '临港校区': 'lingang', '临港基地': 'lingang',
}
export function officialCampusId(value: string): string | undefined {
  const key = value.trim().replace(/^同济大学/u, '')
  return Object.hasOwn(campusNames, key) ? campusNames[key] : undefined
}

/** Exact source text; matching scoped overrides beat campus-wide rules. Misses use stable parsing. */
export function findLocationOverride(campusId: string, raw: string, context: LocationLookupContext,
  rules: readonly LocationOverride[] = (overrides as LocationOverrideConfig).rules): LocationOverride | undefined {
  const matches = rules.filter(rule => rule.campusId === campusId && rule.raw === raw && (!rule.scope || (
    (context.calendarId !== undefined || context.term !== undefined) &&
    (context.calendarId === undefined || context.calendarId === rule.scope.calendarId) &&
    (context.term === undefined || rule.scope.termNames.includes(context.term))
  )))
  return matches.find(rule => rule.scope) ?? matches[0]
}

function present(result: LocationExtraction, reviewPending: boolean, block: boolean): OfficialMapLocation[] {
  return result.locations.map(member => ({
    raw: member.source_text,
    building: member.place ?? '', room: member.detail ?? '',
    condition: [...(member.time ?? []), ...member.conditions].join('且'),
    kind: member.kind, relation: result.relation,
    time: member.time ? [...member.time] : null, conditions: [...member.conditions],
    address: member.address, campusText: member.campus_text,
    unassignedConditions: [...result.unassigned_conditions],
    needsReview: result.needs_review || block, reviewPending,
    alternative: ['alternative', 'mixed', 'unclear'].includes(result.relation),
    hint: result.needs_review || block ? 'review' : member.kind === 'unknown' ? 'missing' : member.kind,
  }))
}

export function parseOfficialLocations(value: string, campus = '', context: LocationLookupContext = {}): OfficialMapLocation[] {
  if (!value.trim()) return []
  const campusId = officialCampusId(campus)
  if (!campusId) return present(missingExtraction(value), false, false)
  const rule = findLocationOverride(campusId, value, context)
  // A matching block/concern is final: it cannot fall through to a guessed destination.
  return rule ? present(rule.result, rule.reviewPending, rule.action === 'block') : present(parseStableLocation(value, campusId), false, false)
}

export function parseOfficialLocation(value: string, campus = '', context: LocationLookupContext = {}): { building: string; room: string } | null {
  const locations = parseOfficialLocations(value, campus, context)
  const location = locations[0]
  return locations.length === 1 && location?.building && location.room && !location.needsReview
    ? { building: location.building, room: location.room } : null
}

const normalize = (value: string) => value.normalize('NFKC').toLowerCase().replace(/[\s\p{P}\p{S}]/gu, '')
interface PlaceIndex {
  featureIds: Set<string>
  names: Map<string, Set<string>>
}
const placeIndexes = new WeakMap<CampusData, PlaceIndex>()
function placeIndex(data: CampusData): PlaceIndex {
  const existing = placeIndexes.get(data)
  if (existing) return existing
  const index: PlaceIndex = { featureIds: new Set(), names: new Map() }
  for (const feature of data.features) {
    const p = feature.properties
    if (feature.id == null || !p.campus || !['academic', 'library', 'place', 'sport'].includes(p.category)) continue
    const id = String(feature.id)
    index.featureIds.add(id)
    for (const name of [p.name ?? '', p.alt_name ?? '', p.short_name ?? ''].flatMap(name =>
      [name, name.replace(/[（(][^（）()]*[）)]/gu, '').trim()]).flatMap(name => name.split(/[、,，;；]/u))) {
      const key = normalize(name.replace(/^(?:同济大学)?(?:四平(?:路)?|嘉定|沪西|沪北|张江|临港)(?:校区|基地)?\s*/u, '').replace(/^同济大学/u, ''))
      if (!key) continue
      const ids = index.names.get(key) ?? new Set<string>()
      ids.add(id); index.names.set(key, ids)
    }
  }
  placeIndexes.set(data, index)
  return index
}

export function officialLocationTargets(campus: string, value: string, data?: CampusData | null, context: LocationLookupContext = {}): OfficialMapLocation[] {
  const campusId = officialCampusId(campus)
  const index = data ? placeIndex(data) : undefined
  return parseOfficialLocations(value, campus, context).map(location => {
    if (location.hint === 'missing' || location.needsReview || !['named', 'generic'].includes(location.kind ?? '')) return location
    if (!campusId || (context.dataCampusId && context.dataCampusId !== campusId) ||
      (location.campusText && officialCampusId(location.campusText) !== campusId)) {
      location.hint = 'campus'; return location
    }
    const entries = (placeCatalog[campusId] ?? []).filter(place => [place.name, ...place.aliases].some(name => normalize(name) === normalize(location.building)))
    const identities = new Set(entries.map(place => place.featureId))
    const sourceIds = new Set(entries.flatMap(place => [place.name, ...place.aliases].flatMap(name => [...(index?.names.get(normalize(name)) ?? [])])))
    const ids = entries.length ? identities : index?.names.get(normalize(location.building)) ?? new Set<string>()
    // Dataset removal or ambiguous source names cannot silently use a stale catalog target.
    if (ids.size === 1 && sourceIds.size <= 1 && (!sourceIds.size || sourceIds.has([...ids][0]!)) &&
      (!data || index?.featureIds.has([...ids][0]!))) {
      location.target = { campusId, featureId: [...ids][0]! }; location.hint = undefined
    } else location.hint = location.kind === 'generic' ? 'generic' : 'unmapped'
    return location
  })
}

export function officialLocationTarget(campus: string, value: string, data?: CampusData | null, context: LocationLookupContext = {}): CampusMapTarget | undefined {
  const locations = officialLocationTargets(campus, value, data, context)
  return locations.length === 1 ? locations[0]?.target : undefined
}

export function officialLocationApplies(location: OfficialMapLocation, context: { week: number; day: number }): boolean {
  if (location.alternative || location.needsReview || location.hint === 'missing' ||
    location.unassignedConditions?.length || location.conditions?.length) return false
  if (!Number.isInteger(context.week) || context.week < 1 || !Number.isInteger(context.day) || context.day < 1 || context.day > 7) return false
  const times = location.time ?? (location.condition ? location.condition.split('且') : [])
  return times.every(condition => conditionApplies(condition.replace(/^上课/u, ''), context))
}
function conditionApplies(condition: string, context: { week: number; day: number }): boolean {
  if (condition === '单周') return context.week % 2 === 1
  if (condition === '双周') return context.week % 2 === 0
  if (/^周[一二三四五六日天]$/u.test(condition)) return ('一二三四五六日'.indexOf(condition[1]!.replace('天', '日')) + 1) === context.day
  const weeks = condition.match(/^第?(\d+)(?:[-~～至](\d+))?周$/u)
  if (weeks) return context.week >= Number(weeks[1]) && context.week <= Number(weeks[2] ?? weeks[1])
  const first = condition.match(/^前(\d+)周$/u)
  if (first) return context.week <= Number(first[1])
  // Last-N weeks, dates, times and unspecified alternatives require more source context.
  return false
}
