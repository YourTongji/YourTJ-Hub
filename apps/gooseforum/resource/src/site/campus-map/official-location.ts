import type { CampusData, CampusFeature } from './catalog'

export interface CampusMapTarget {
  campusId: string
  featureId: string
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
  const matches = campusMarkers.filter(([, marker]) => marker.test(value))
  return matches.length === 1 ? matches[0]?.[0] : undefined
}

function withoutCampusPrefix(value: string, campus = ''): string {
  let result = value.trim()
  if (campus && result.startsWith(campus)) result = result.slice(campus.length).trim()
  return result.replace(/^(?:同济大学)?(?:四平路|嘉定|沪西|沪北|张江|临港)(?:校区|基地)?\s*/u, '').trim()
}

export function parseOfficialLocation(value: string, campus = ''): { building: string; room: string } | null {
  const location = withoutCampusPrefix(value, campus)
  const match = location.match(/^(.+?(?:楼|馆|中心|大厦|栋)(?:（[^（）]+）|\([^()]+\))?|[北南东西]|[A-Za-z])\s*([A-Za-z]?\d{1,4}(?:[-/]\w{1,4})?)(?:\s*室)?$/u)
  if (!match) return null
  return { building: match[1]!, room: match[2]! }
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
  return values.flatMap((value) => value.split(/[、,，;；]/u)).filter(Boolean)
}

export function officialLocationTarget(
  campus: string,
  value: string,
  data?: CampusData | null,
): CampusMapTarget | undefined {
  const campusId = officialCampusId(campus)
  if (!campusId) return undefined
  const location = withoutCampusPrefix(value, campus)
  const parsed = parseOfficialLocation(location)
  const building = (parsed?.building ?? location).trim()
  if (!building) return undefined

  if (campusId === 'siping') {
    if (['北', '北楼', '教学北楼'].includes(building) && (parsed || building !== '北'))
      return { campusId, featureId: 'way/183383474' }
    if (['南', '南楼', '教学南楼'].includes(building) && (parsed || building !== '南'))
      return { campusId, featureId: 'way/183383472' }
  }
  if (campusId === 'jiading' && ['济事楼', '济事南楼', '济事北楼', '济事楼（软件学院）'].includes(building))
    return { campusId, featureId: 'way/135405205' }
  if (!data) return undefined

  const key = normalize(building)
  const matches = data.features.filter((feature) =>
    feature.properties.campus &&
    ['academic', 'library', 'place'].includes(feature.properties.category) &&
    aliases(feature).some((name) => normalize(name) === key),
  )
  if (matches.length !== 1 || matches[0]?.id == null) return undefined
  return { campusId, featureId: String(matches[0].id) }
}
