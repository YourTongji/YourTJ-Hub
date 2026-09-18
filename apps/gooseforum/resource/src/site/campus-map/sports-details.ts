import type { Feature, FeatureCollection, Geometry, Position } from 'geojson'
import {
  geometryPositions,
  type CampusData,
  type CampusFeature,
} from './catalog'

// Decorative markings follow the measured footprint and orientation. Lane count
// and field markings are cartographic illustrations, not a venue survey.
type Point = [number, number]
interface Frame {
  length: number
  width: number
  project: (point: Point) => Point
}
export function fieldFrame(feature: CampusFeature): Frame | undefined {
  if (!['Polygon', 'MultiPolygon'].includes(feature.geometry.type)) return
  const origin = feature.properties.center
  const sx = 111320 * Math.cos((origin[1] * Math.PI) / 180)
  const sy = 111320
  const points = geometryPositions(feature.geometry).map(
    (p) => [(p[0]! - origin[0]) * sx, (p[1]! - origin[1]) * sy] as Point,
  )
  let best:
    | {
        area: number
        a: number
        minX: number
        maxX: number
        minY: number
        maxY: number
      }
    | undefined
  for (let i = 1; i < points.length; i++) {
    const dx = points[i]![0] - points[i - 1]![0],
      dy = points[i]![1] - points[i - 1]![1]
    if (Math.hypot(dx, dy) < 3) continue
    const a = Math.atan2(dy, dx),
      c = Math.cos(a),
      s = Math.sin(a)
    const xs = points.map((p) => p[0] * c + p[1] * s),
      ys = points.map((p) => -p[0] * s + p[1] * c)
    const minX = Math.min(...xs),
      maxX = Math.max(...xs),
      minY = Math.min(...ys),
      maxY = Math.max(...ys)
    const area = (maxX - minX) * (maxY - minY)
    if (!best || area < best.area) best = { area, a, minX, maxX, minY, maxY }
  }
  if (!best || best.area < 10) return
  const b = best,
    c = Math.cos(b.a),
    s = Math.sin(b.a)
  const w = b.maxX - b.minX,
    h = b.maxY - b.minY
  return {
    length: Math.max(w, h),
    width: Math.min(w, h),
    project: ([x, y]) => {
      const px = (w >= h ? x : -y) + (b.minX + b.maxX) / 2,
        py = (w >= h ? y : x) + (b.minY + b.maxY) / 2
      return [
        origin[0] + (px * c - py * s) / sx,
        origin[1] + (px * s + py * c) / sy,
      ]
    },
  }
}
const rectangle = (l: number, w: number): Point[] => [
  [-l / 2, -w / 2],
  [l / 2, -w / 2],
  [l / 2, w / 2],
  [-l / 2, w / 2],
  [-l / 2, -w / 2],
]
function capsule(l: number, w: number): Point[] {
  const r = w / 2,
    d = Math.max(0, (l - w) / 2),
    ps: Point[] = []
  for (let i = 0; i <= 32; i++) {
    const a = -Math.PI / 2 + (i * Math.PI) / 32
    ps.push([d + r * Math.cos(a), r * Math.sin(a)])
  }
  for (let i = 0; i <= 32; i++) {
    const a = Math.PI / 2 + (i * Math.PI) / 32
    ps.push([-d + r * Math.cos(a), r * Math.sin(a)])
  }
  return [...ps, ps[0]!]
}
const ellipse = (x: number, y: number, rx: number, ry = rx): Point[] =>
  Array.from({ length: 49 }, (_, i) => [
    x + rx * Math.cos((i * Math.PI) / 24),
    y + ry * Math.sin((i * Math.PI) / 24),
  ])
export function makeSportsDetails(data: CampusData): FeatureCollection {
  const features: Feature<Geometry>[] = []
  const sports = data.features.filter(
    (f) =>
      f.properties.campus &&
      f.properties.category === 'sport' &&
      !f.properties.building &&
      !/馆/.test(f.properties.name ?? '') &&
      ['Polygon', 'MultiPolygon'].includes(f.geometry.type),
  )
  const trackIds = new Set(['way/176687553', 'way/379036006'])
  const tracks = sports.filter(
    (f) =>
      f.properties.leisure === 'track' ||
      trackIds.has(String(f.id)) ||
      f.properties['map:track'] === 'oval',
  )
  for (const f of sports) {
    const frame = fieldFrame(f)
    if (!frame) continue
    const { length: l, width: w, project } = frame
    const track = tracks.includes(f),
      straight = f.properties['map:track'] === 'straight'
    // A pitch inside an oval already gets its markings with the track.
    if (
      !track &&
      !straight &&
      tracks.some((t) => {
        const tf = fieldFrame(t)
        if (!tf) return false
        const c = f.properties.center,
          tc = t.properties.center
        return (
          Math.hypot((c[0] - tc[0]) * 95000, (c[1] - tc[1]) * 111320) <
          Math.min(tf.width, tf.length) * 0.3
        )
      })
    )
      continue
    const add = (points: Point[], part: string, polygon = false) =>
      features.push({
        type: 'Feature',
        properties: { part, parent: String(f.id) },
        geometry: polygon
          ? { type: 'Polygon', coordinates: [points.map(project)] }
          : { type: 'LineString', coordinates: points.map(project) },
      })
    const football = (length: number, width: number) => {
      add(rectangle(length, width), 'marking')
      add(
        [
          [0, -width / 2],
          [0, width / 2],
        ],
        'marking',
      )
      add(ellipse(0, 0, Math.min(width * 0.16, 9.15)), 'marking')
      for (const sign of [-1, 1]) {
        const x = (sign * length) / 2,
          d = Math.min(16.5, length * 0.16),
          small = Math.min(5.5, length * 0.055)
        add(
          [
            [x, -width * 0.31],
            [x - sign * d, -width * 0.31],
            [x - sign * d, width * 0.31],
            [x, width * 0.31],
          ],
          'marking',
        )
        add(
          [
            [x, -width * 0.14],
            [x - sign * small, -width * 0.14],
            [x - sign * small, width * 0.14],
            [x, width * 0.14],
          ],
          'marking',
        )
      }
    }
    if (track && l > 60 && w > 30) {
      const lane = Math.min(9, w * 0.14)
      add(capsule(l, w), 'track', true)
      for (let i = 1; i < 7; i++)
        add(capsule(l - (2 * lane * i) / 7, w - (2 * lane * i) / 7), 'lane')
      add(capsule(l - 2 * lane, w - 2 * lane), 'infield', true)
      const fl = Math.min(l - w * 0.5 - 2 * lane, l * 0.62),
        fw = (w - 2 * lane) * 0.82
      for (let i = 0; i < 10; i += 2)
        add(
          rectangle(fl / 10, fw).map(([x, y]) => [
            x - fl / 2 + (fl * (i + 0.5)) / 10,
            y,
          ]),
          'stripe',
          true,
        )
      football(fl, fw)
      add(
        [
          [-(l - w) / 2, -w / 2],
          [-(l - w) / 2, -w / 2 + lane],
        ],
        'marking',
      )
    } else if (straight) {
      // This explicit plan override uses NW, NE, SE, SW corner order.
      // Keep the strip on the supplied west edge and the grass disjoint from it.
      const corners = geometryPositions(f.geometry)
      const interpolate = (a: Position, b: Position, t: number): Position => [
        a[0]! + (b[0]! - a[0]!) * t,
        a[1]! + (b[1]! - a[1]!) * t,
      ]
      const nw = corners[0]!,
        ne = corners[1]!,
        se = corners[2]!,
        sw = corners[3]!
      const top = interpolate(nw, ne, 0.13),
        bottom = interpolate(sw, se, 0.13)
      for (const [part, ring] of [
        ['track', [nw, top, bottom, sw, nw]],
        ['infield', [top, ne, se, bottom, top]],
      ] as const)
        features.push({
          type: 'Feature',
          properties: { part, parent: String(f.id) },
          geometry: { type: 'Polygon', coordinates: [[...ring]] },
        })
      for (let i = 1; i < 5; i++)
        features.push({
          type: 'Feature',
          properties: { part: 'lane', parent: String(f.id) },
          geometry: {
            type: 'LineString',
            coordinates: [
              interpolate(nw, top, i / 5),
              interpolate(sw, bottom, i / 5),
            ],
          },
        })
    } else if (f.properties.sport === 'soccer' && l > 30) {
      football(l * 0.93, w * 0.91)
    } else if (
      ['basketball', 'tennis', 'volleyball', 'badminton'].includes(
        f.properties.sport ?? '',
      ) &&
      l < 48 &&
      w < 30
    ) {
      const cl = l * 0.86,
        cw = w * 0.82
      add(rectangle(cl, cw), 'marking')
      add(
        [
          [0, -cw / 2],
          [0, cw / 2],
        ],
        'marking',
      )
      if (f.properties.sport === 'basketball') {
        add(ellipse(0, 0, cw * 0.12), 'marking')
        for (const sign of [-1, 1]) {
          const x = (sign * cl) / 2
          add(
            [
              [x, -cw * 0.16],
              [x - sign * cl * 0.2, -cw * 0.16],
              [x - sign * cl * 0.2, cw * 0.16],
              [x, cw * 0.16],
            ],
            'marking',
          )
          add(ellipse(x - sign * cl * 0.2, 0, cw * 0.12), 'marking')
        }
      } else {
        add(
          [
            [-cl / 2, -cw * 0.36],
            [cl / 2, -cw * 0.36],
          ],
          'marking',
        )
        add(
          [
            [-cl / 2, cw * 0.36],
            [cl / 2, cw * 0.36],
          ],
          'marking',
        )
        for (const x of [-cl * 0.27, cl * 0.27])
          add(
            [
              [x, -cw * 0.36],
              [x, cw * 0.36],
            ],
            'marking',
          )
        add(
          [
            [-cl * 0.27, 0],
            [cl * 0.27, 0],
          ],
          'marking',
        )
      }
    } else if (f.properties.leisure === 'swimming_pool' && l < 65) {
      for (let i = 1; i < 7; i++)
        add(
          [
            [-l * 0.46, -w / 2 + (w * i) / 7],
            [l * 0.46, -w / 2 + (w * i) / 7],
          ],
          'lane',
        )
    }
  }
  return { type: 'FeatureCollection', features }
}
