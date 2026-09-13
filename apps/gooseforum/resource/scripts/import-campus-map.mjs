// Import a pinned YourTJ Pulse / OpenStreetMap snapshot. Geometry stays in WGS84;
// OSM contributor account metadata is deliberately not part of the product asset.
import osmtogeojson from 'osmtogeojson'
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const campusId = process.argv[3] ? process.argv[2] : 'siping'
const input = process.argv[3] || process.argv[2]
if (!input)
  throw new Error(
    'Usage: node scripts/import-campus-map.mjs /path/to/full.geojson',
  )
const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const config = JSON.parse(
  readFileSync(resolve(root, 'src/site/campus-map/data/campuses.json')),
).find((c) => c.id === campusId)
if (!config) throw new Error('Unknown campus')
const bounds = config.bounds.map((v, i) => v + (i < 2 ? -0.0008 : 0.0008))
const raw = JSON.parse(readFileSync(input, 'utf8'))
const source = raw.elements ? osmtogeojson(raw, { flatProperties: true }) : raw
const extraPath = resolve(
  root,
  `src/site/campus-map/data/${campusId}-supplement.geojson`,
)
if (existsSync(extraPath)) {
  const extras = JSON.parse(readFileSync(extraPath)).features
  const ids = new Set(extras.map((f) => f.id))
  source.features = source.features
    .filter(
      (f) =>
        !ids.has(f.id) &&
        !(
          campusId === 'hubei' &&
          f.properties?.building &&
          f.id === 'way/1504272142'
        ),
    )
    .concat(extras)
}
function positions(coordinates) {
  return typeof coordinates[0] === 'number'
    ? [coordinates]
    : coordinates.flatMap(positions)
}
function center(geometry) {
  const ps = positions(geometry.coordinates)
  return [
    (Math.min(...ps.map((p) => p[0])) + Math.max(...ps.map((p) => p[0]))) / 2,
    (Math.min(...ps.map((p) => p[1])) + Math.max(...ps.map((p) => p[1]))) / 2,
  ]
}
function category(p) {
  if (
    p.sport ||
    [
      'pitch',
      'track',
      'stadium',
      'swimming_pool',
      'sports_hall',
      'sports_centre',
      'golf_course',
    ].includes(p.leisure)
  )
    return 'sport'
  if (p.amenity === 'library' || /图书馆/.test(p.name ?? '')) return 'library'
  if (
    ['restaurant', 'cafe', 'fast_food', 'food_court'].includes(p.amenity) ||
    /餐厅|食堂|饮食广场/.test(p.name ?? '')
  )
    return 'food'
  if (
    ['dormitory', 'apartments'].includes(p.building) ||
    /宿舍|西南.*楼|西北.*楼/.test(p.name ?? '')
  )
    return 'living'
  if (p.building) return 'academic'
  if (p.natural === 'water' || p.waterway) return 'water'
  if (
    p.natural === 'tree' ||
    ['grass', 'forest', 'village_green'].includes(p.landuse) ||
    ['garden', 'park'].includes(p.leisure)
  )
    return 'green'
  return 'place'
}
const campus = source.features.find((f) => f.id === config.boundaryId)
if (!campus || !['Polygon', 'MultiPolygon'].includes(campus.geometry.type))
  throw new Error('Expected campus boundary: ' + config.boundaryId)
function inRing([x, y], ring) {
  let inside = false
  for (let i = 0, j = ring.length - 1; i < ring.length; j = i++) {
    const [xi, yi] = ring[i]
    const [xj, yj] = ring[j]
    if (yi > y !== yj > y && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi)
      inside = !inside
  }
  return inside
}
function onCampus(point) {
  const polys =
    campus.geometry.type === 'Polygon'
      ? [campus.geometry.coordinates]
      : campus.geometry.coordinates
  return polys.some(
    ([outer, ...holes]) =>
      inRing(point, outer) && !holes.some((ring) => inRing(point, ring)),
  )
}
const features = source.features
  .filter((f) => {
    if (!f.geometry || !f.id || f.geometry.type === 'GeometryCollection')
      return false
    if (
      String(f.id).startsWith('relation/') &&
      !['Polygon', 'MultiPolygon'].includes(f.geometry.type)
    )
      return false
    if (
      ![
        'name',
        'building',
        'leisure',
        'landuse',
        'natural',
        'waterway',
        'highway',
        'amenity',
        'barrier',
        'source',
        'map:track',
        'sport',
      ].some((k) => f.properties?.[k])
    )
      return false
    const [x, y] = center(f.geometry)
    return x >= bounds[0] && y >= bounds[1] && x <= bounds[2] && y <= bounds[3]
  })
  .map((f) => {
    const p = { ...f.properties }
    if (f.id === config.boundaryId) p.amenity = 'university'
    const point = center(f.geometry)
    const isCampus = onCampus(point)
    const properties = {
      category: isCampus ? category(p) : p.building ? 'place' : category(p),
      center: point,
      campus: isCampus,
    }
    for (const key of [
      'name',
      'name:en',
      'alt_name',
      'short_name',
      'building',
      'building:levels',
      'leisure',
      'sport',
      'landuse',
      'natural',
      'waterway',
      'highway',
      'amenity',
      'entrance',
      'barrier',
      'source',
      'map:track',
    ]) {
      if (typeof p[key] === 'string') properties[key] = p[key]
    }
    return { type: 'Feature', id: f.id, properties, geometry: f.geometry }
  })
const destination = resolve(
  root,
  `src/site/campus-map/data/${campusId}.geojson`,
)
mkdirSync(dirname(destination), { recursive: true })
// One feature per line keeps snapshot changes reviewable without reformatting every coordinate.
writeFileSync(
  destination,
  '{"type":"FeatureCollection","features":[\n' +
    features.map((f) => JSON.stringify(f)).join(',\n') +
    '\n]}\n',
)
console.log(`Imported ${features.length} features to ${destination}`)
