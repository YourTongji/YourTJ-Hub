import type { CampusData, CampusFeature } from './catalog'
import table from './data/locations/2026-2027-1.json'
import type { LocationDictionary, LocationExtraction, LocationHint, LocationKind, LocationLookupContext, LocationRelation } from './location-types'
export type { LocationLookupContext } from './location-types'

// The asset is validated against the extraction schema in CI; it is not a model call at runtime.
const dictionary = table as LocationDictionary

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
  return campusNames[value.trim().replace(/^同济大学/u, '')]
}

function extraction(campus: string, raw: string, context: LocationLookupContext): LocationExtraction | undefined {
  const meta = dictionary._meta
  // Never apply one term's faculty-derived names to other terms or an unknown calendar.
  if (context.calendarId !== undefined && context.calendarId !== meta.calendar_id) return
  if (context.term !== undefined && !meta.term_names.includes(context.term)) return
  if (context.calendarId === undefined && context.term === undefined) return
  // Keys are source strings, not normalized aliases. Do not silently combine source identities.
  const entries = Object.hasOwn(dictionary.dictionary, campus) ? dictionary.dictionary[campus] : undefined
  return entries && Object.hasOwn(entries, raw) ? entries[raw] : undefined
}

export function parseOfficialLocations(value: string, campus = '', context: LocationLookupContext = {}): OfficialMapLocation[] {
  if (!value.trim()) return []
  const result = extraction(campus, value, context)
  if (!result) return [{ raw: value, building: '', room: '', condition: '', kind: 'unknown', hint: 'missing', time: null }]
  return result.locations.map(member => ({
    raw: member.source_text,
    building: member.place ?? '',
    room: member.detail ?? '',
    condition: [...(member.time ?? []), ...member.conditions].join('且'),
    kind: member.kind,
    relation: result.relation,
    time: member.time ? [...member.time] : null,
    conditions: [...member.conditions],
    address: member.address,
    campusText: member.campus_text,
    unassignedConditions: [...result.unassigned_conditions],
    needsReview: result.needs_review,
    reviewPending: dictionary._meta.review_status === 'review_2_pending',
    alternative: ['alternative', 'mixed', 'unclear'].includes(result.relation),
    hint: result.needs_review ? 'review' : member.kind,
  }))
}

export function parseOfficialLocation(value: string, campus = '', context: LocationLookupContext = {}): { building: string; room: string } | null {
  const locations = parseOfficialLocations(value, campus, context)
  const location = locations[0]
  return locations.length === 1 && location?.building && location.room
    ? { building: location.building, room: location.room } : null
}

function normalize(value: string): string {
  return value.normalize('NFKC').toLowerCase().replace(/[\s\p{P}\p{S}]/gu, '')
}
function aliases(feature: CampusFeature): string[] {
  const { name = '', alt_name = '', short_name = '' } = feature.properties
  const values = [name, alt_name, short_name, name.replace(/[（(][^（）()]*[）)]/gu, '').trim()]
  const letter = name.match(/[（(]([A-Za-z])楼[）)]/u)?.[1]
  if (letter) values.push(letter, `${letter}楼`)
  return values.flatMap(value => value.split(/[、,，;；]/u))
    .flatMap(value => [value, value.replace(/^(?:同济大学)?(?:四平(?:路)?|嘉定|沪西|沪北|张江|临港)(?:校区|基地)?\s*/u, '').replace(/^同济大学/u, '')])
    .map(normalize).filter(Boolean)
}

// Curated building identities are separate from text extraction and never parse a room.
const confirmedPlaces: Record<string, Record<string, string>> = {
  siping: {
    '北教学楼': 'way/183383474', '北楼': 'way/183383474', '教学北楼': 'way/183383474',
    '南教学楼': 'way/183383472', '南楼': 'way/183383472', '教学南楼': 'way/183383472',
  },
  jiading: {
    '济事楼': 'way/135405205', '济事南楼': 'way/135405205', '济事北楼': 'way/135405205', '济事楼（软件学院）': 'way/135405205',
    // The recorded sports_hall hosts these activities; this identifies its building, not an indoor room.
    '体育中心游泳馆': 'way/1456432428', '体育中心篮球馆': 'way/1456432428', '体育中心乒乓馆': 'way/1456432428',
  },
}

interface PlaceIndex {
  featureIds: Set<string>
  names: Map<string, Set<string>>
}
// Campus datasets are immutable after loading; reuse their name projection across schedule rows.
const placeIndexes = new WeakMap<CampusData, PlaceIndex>()
function placeIndex(data: CampusData): PlaceIndex {
  const existing = placeIndexes.get(data)
  if (existing) return existing
  const index: PlaceIndex = { featureIds: new Set(), names: new Map() }
  for (const feature of data.features) {
    if (feature.id == null || !feature.properties.campus ||
      !['academic', 'library', 'place', 'sport'].includes(feature.properties.category)) continue
    const id = String(feature.id)
    index.featureIds.add(id)
    for (const key of aliases(feature)) {
      const ids = index.names.get(key) ?? new Set<string>()
      ids.add(id)
      index.names.set(key, ids)
    }
  }
  placeIndexes.set(data, index)
  return index
}

export function officialLocationTargets(campus: string, value: string, data?: CampusData | null, context: LocationLookupContext = {}): OfficialMapLocation[] {
  const campusId = officialCampusId(campus)
  const index = data ? placeIndex(data) : undefined
  return parseOfficialLocations(value, campus, context).map(location => {
    if (location.hint === 'missing' || location.needsReview) return location
    if (!['named', 'generic'].includes(location.kind ?? '')) return location
    if (!campusId || (context.dataCampusId && context.dataCampusId !== campusId) ||
      (location.campusText && officialCampusId(location.campusText) !== campusId)) {
      location.hint = 'campus'
      return location
    }
    const confirmed = confirmedPlaces[campusId]?.[location.building]
    const ids = confirmed
      ? new Set(!data || index?.featureIds.has(confirmed) ? [confirmed] : [])
      : index?.names.get(normalize(location.building)) ?? new Set<string>()
    if (ids.size === 1) {
      location.target = { campusId, featureId: [...ids][0]! }
      location.hint = undefined
    } else location.hint = location.kind === 'generic' ? 'generic' : 'unmapped'
    return location
  })
}

export function officialLocationTarget(campus: string, value: string, data?: CampusData | null, context: LocationLookupContext = {}): CampusMapTarget | undefined {
  const locations = officialLocationTargets(campus, value, data, context)
  // Multiple members always require a choice, including two rooms in the same building.
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
