import type { Feature, FeatureCollection, Geometry, Position } from 'geojson'

export type PlaceCategory =
  | 'academic'
  | 'sport'
  | 'library'
  | 'food'
  | 'living'
  | 'place'
export type Category = 'all' | PlaceCategory
export interface MapProperties {
  'map:track'?: string
  source?: string
  category: PlaceCategory | 'water' | 'green'
  center: [number, number]
  campus: boolean
  name?: string
  'name:en'?: string
  alt_name?: string
  short_name?: string
  building?: string
  'building:levels'?: string
  leisure?: string
  sport?: string
  landuse?: string
  natural?: string
  waterway?: string
  highway?: string
  amenity?: string
}
export type CampusFeature = Feature<Geometry, MapProperties>
export type CampusData = FeatureCollection<Geometry, MapProperties>
export interface CampusPlace {
  id: string
  name: string
  aliases: string[]
  category: PlaceCategory
  sports: string[]
  indoor: boolean
  named: boolean
  center: [number, number]
  feature: CampusFeature
}
export const campusCenter: [number, number] = [121.49735, 31.28435]
export const campusBounds: [[number, number], [number, number]] = [
  [121.4915, 31.2794],
  [121.5035, 31.2896],
]
export const sportNames: Record<string, string> = {
  basketball: '篮球',
  badminton: '羽毛球',
  tennis: '网球',
  soccer: '足球',
  running: '跑步',
  swimming: '游泳',
  volleyball: '排球',
  gymnastics: '体操',
  fitness: '健身',
  weightlifting: '力量训练',
  table_tennis: '乒乓球',
  judo: '柔道',
  wushu: '武术',
  roller_skating: '轮滑',
  golf: '高尔夫球',
  dragon_boat: '龙舟',
}

export function sportsFor(p: MapProperties): string[] {
  // Some OSM facilities identify their activity only through leisure, without sport.
  const inferred =
    p.leisure === 'track' || p['map:track']
      ? 'running'
      : p.leisure === 'swimming_pool'
        ? 'swimming'
        : p.leisure === 'golf_course'
          ? 'golf'
          : ''
  return [...new Set([inferred, ...(p.sport ?? '').split(';')].filter(Boolean))]
}

export function makePlace(feature: CampusFeature): CampusPlace {
  const p = feature.properties
  const sports = sportsFor(p)
  const name =
    p.name ||
    (p.leisure === 'track' ? '田径场' : '') ||
    (p.leisure === 'sports_centre' ? '体育中心' : '') ||
    (sports.length
      ? `${sportNames[sports[0]!] ?? sports[0]}场地`
      : p.building
        ? '未标注名称的建筑'
        : '校园地点')
  return {
    id: String(feature.id),
    name,
    aliases: [p['name:en'], p.alt_name, p.short_name].filter((v): v is string =>
      Boolean(v),
    ),
    category:
      p.category === 'green' || p.category === 'water' ? 'place' : p.category,
    sports,
    indoor: Boolean(p.building) || p.leisure === 'sports_hall',
    named: Boolean(p.name),
    center: p.center,
    feature,
  }
}

export function buildCatalog(data: CampusData): CampusPlace[] {
  // Preserve unnamed sports polygons: searching by activity must find outdoor courts too.
  return data.features
    .filter((f) => {
      const p = f.properties
      if (!p.campus) return false
      if (p.category === 'sport') return true
      if (!p.name || p.highway || p.waterway) return false
      return (
        Boolean(p.building) ||
        (f.geometry.type === 'Point' && Boolean(p.amenity))
      )
    })
    .map(makePlace)
    .sort(
      (a, b) =>
        Number(b.named) - Number(a.named) ||
        a.name.localeCompare(b.name, 'zh-CN'),
    )
}

export function searchPlaces(
  places: CampusPlace[],
  query: string,
  category: Category,
  sport = '',
): CampusPlace[] {
  const terms = query.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean)
  return places
    .filter((place) => {
      if (category !== 'all' && place.category !== category) return false
      if (sport && !place.sports.includes(sport)) return false
      const text = [
        place.name,
        ...place.aliases,
        ...place.sports.map((s) => sportNames[s] ?? s),
      ]
        .join(' ')
        .toLocaleLowerCase()
      return terms.every((term) => text.includes(term))
    })
    .sort(
      (a, b) =>
        Number(b.name === query.trim()) - Number(a.name === query.trim()),
    )
}

export function geometryPositions(geometry: Geometry): Position[] {
  if (geometry.type === 'GeometryCollection')
    return geometry.geometries.flatMap(geometryPositions)
  const flatten = (value: unknown): Position[] => {
    if (!Array.isArray(value)) return []
    return typeof value[0] === 'number'
      ? [value as Position]
      : value.flatMap(flatten)
  }
  return flatten(geometry.coordinates)
}

export function featureBounds(
  feature: CampusFeature,
): [[number, number], [number, number]] {
  const ps = geometryPositions(feature.geometry)
  return [
    [Math.min(...ps.map((p) => p[0]!)), Math.min(...ps.map((p) => p[1]!))],
    [Math.max(...ps.map((p) => p[0]!)), Math.max(...ps.map((p) => p[1]!))],
  ]
}
