import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import {
  buildCatalog,
  featureBounds,
  navigationHref,
  searchPlaces,
  type CampusData,
} from '../src/site/campus-map/catalog'
import { makeMapStyle } from '../src/site/campus-map/style'

const data = JSON.parse(
  readFileSync(
    new URL('../src/site/campus-map/data/siping.geojson', import.meta.url),
    'utf8',
  ),
) as CampusData
const places = buildCatalog(data)

describe('campus map navigation destinations', () => {
  it('links only real buildings with valid coordinates on calibrated campuses', () => {
    const building = places.find((place) => place.feature.properties.building)!
    const href = navigationHref(building, undefined, 'iPhone')
    expect(href).toBeDefined()
    const url = new URL(href!)
    expect(url.hostname).toBe('maps.apple.com')
    expect(url.searchParams.get('daddr')).toBe(
      `${building.center[1]},${building.center[0]}`,
    )
    expect(url.searchParams.get('dirflg')).toBe('w')
    expect(url.searchParams.get('q')).toBe(building.name)

    expect(navigationHref(building, 'schematic')).toBeUndefined()
    const outdoor = places.find((place) => !place.feature.properties.building)!
    expect(navigationHref(outdoor)).toBeUndefined()
    expect(navigationHref({ ...building, center: undefined as never })).toBeUndefined()
    for (const center of [
      [NaN, 31],
      [181, 31],
      [121, -91],
    ] as [number, number][]) {
      expect(navigationHref({ ...building, center })).toBeUndefined()
    }
  })
  it('offers a native Android destination and a WGS84 web route fallback', () => {
    const building = places.find((place) => place.feature.properties.building)!
    const [lon, lat] = building.center
    expect(navigationHref(building, undefined, 'Android')).toBe(
      `geo:${lat},${lon}?q=${encodeURIComponent(`${lat},${lon}(${building.name})`)}`,
    )
    const web = new URL(navigationHref(building, undefined, '')!)
    expect(web.hostname).toBe('api.map.baidu.com')
    expect(web.searchParams.get('destination')).toBe(`latlng:${lat},${lon}|name:${building.name}`)
    expect(web.searchParams.get('coord_type')).toBe('wgs84')
    expect(web.searchParams.get('mode')).toBe('walking')
    expect(web.searchParams.get('output')).toBe('html')
  })
})

describe('campus map discovery', () => {
  it.each(['basketball', 'swimming', 'table_tennis'])(
    'finds every %s facility by its source activity tag', (activity) => {
      const matches = searchPlaces(places, activity, 'sport')
      expect(matches.length).toBeGreaterThan(0)
      expect(matches.map((place) => place.id)).toEqual(
        searchPlaces(places, '', 'sport', activity).map((place) => place.id),
      )
    },
  )

  it('finds unnamed outdoor courts by sport, without requiring a building or name', () => {
    const basketball = searchPlaces(places, '篮球', 'sport', 'basketball')
    const outdoor = basketball.filter((place) => !place.named && !place.indoor)
    expect(outdoor.length).toBeGreaterThan(0)
    expect(
      outdoor.every((place) =>
        ['Polygon', 'MultiPolygon'].includes(place.feature.geometry.type),
      ),
    ).toBe(true)
    expect(searchPlaces(places, '篮球', 'library')).toEqual([])
  })

  it('searches aliases and keeps the original feature ID for shared links', () => {
    const pool = searchPlaces(places, 'Tongji Swimming', 'all')
    expect(pool).toHaveLength(1)
    expect(pool[0]).toMatchObject({
      id: 'way/125465017',
      name: '同济大学游泳馆',
      category: 'sport',
      indoor: true,
    })
    expect(
      searchPlaces(places, '图书馆', 'library').some(
        (place) => place.name === '同济大学图书馆',
      ),
    ).toBe(true)
  })

  it('keeps surrounding schools as context rather than campus destinations', () => {
    expect(
      data.features.some(
        (feature) => feature.properties.name === '杨浦高级中学图书馆',
      ),
    ).toBe(true)
    expect(searchPlaces(places, '杨浦高级中学', 'all')).toEqual([])
    expect(places.every((place) => place.feature.properties.campus)).toBe(true)
  })

  it('computes a building extent enclosing its supplied label center', () => {
    const pool = places.find((place) => place.id === 'way/125465017')!
    const [min, max] = featureBounds(pool.feature)
    expect(min[0]).toBeLessThan(pool.center[0])
    expect(min[1]).toBeLessThan(pool.center[1])
    expect(max[0]).toBeGreaterThan(pool.center[0])
    expect(max[1]).toBeGreaterThan(pool.center[1])
  })

  it('ships no third-party tile, font or sprite requests in its map style', () => {
    const style = makeMapStyle(data, true)
    expect(style.glyphs).toBeUndefined()
    expect(style.sprite).toBeUndefined()
    expect(
      Object.values(style.sources).every((source) => source.type === 'geojson'),
    ).toBe(true)
    for (const campus of campuses)
      expect(JSON.stringify(campusData(campus.id))).not.toMatch(/"(?:uid|user|changeset)":/)
  })
})

// Exercise the source data as well as the renderer: a correct algorithm with an
// indoor hall incorrectly classified as a track is still a misleading map.
import { makeSportsDetails } from '../src/site/campus-map/sports-details'
import {
  campuses,
  containingCampus,
  getCampus,
} from '../src/site/campus-map/campuses'
import { locationData, requestLocation } from '../src/site/campus-map/location'
import { vi } from 'vitest'
const campusData = (id: string): CampusData =>
  JSON.parse(
    readFileSync(
      new URL(`../src/site/campus-map/data/${id}.geojson`, import.meta.url),
      'utf8',
    ),
  )
describe('multi-campus atlas', () => {
  it('finds Jiading golf by its activity and keeps named and unnamed golf footprints selectable', () => {
    const jiadingPlaces = buildCatalog(campusData('jiading'))
    const golf = searchPlaces(jiadingPlaces, '高尔夫球', 'sport', 'golf')
    expect(golf.map((place) => place.id).sort()).toEqual(
      ['way/263927462', 'way/1456436384'].sort(),
    )
    expect(golf.find((place) => place.id === 'way/263927462')).toMatchObject({
      name: '高尔夫练习场',
      category: 'sport',
      sports: ['golf'],
      indoor: false,
      named: true,
      feature: { geometry: { type: 'Polygon' } },
    })
    expect(searchPlaces(jiadingPlaces, '高尔夫', 'all')).toHaveLength(2)
    expect(searchPlaces(jiadingPlaces, '高尔夫球', 'academic')).toEqual([])
  })
  it('includes a sports centre without an explicit sport tag in sports discovery', () => {
    expect(
      searchPlaces(places, '', 'sport').find(
        (place) => place.id === 'way/183388119',
      ),
    ).toMatchObject({ category: 'sport', indoor: true })
  })
  it('loads all six distinct campuses and their key buildings', () => {
    expect(campuses).toHaveLength(6)
    for (const [id, name] of [
      ['siping', '经纬楼'],
      ['jiading', '嘉定同济体育中心'],
      ['huxi', '体育馆'],
      ['hubei', '齐贤楼'],
      ['lingang', '实验楼'],
      ['zhangjiang', '闻智楼'],
    ]) {
      expect(buildCatalog(campusData(id!)).some((p) => p.name === name)).toBe(
        true,
      )
    }
    expect(getCampus('invalid').id).toBe('siping')
    expect(containingCampus(121.208, 31.286)?.id).toBe('jiading')
    expect(containingCampus(103.85, 1.29)).toBeUndefined()
    expect(containingCampus(0.001, 0.001)).toBeUndefined()
  })
  it('draws red ovals with green infields and lane lines, never on indoor halls', () => {
    for (const [id, count] of [
      ['siping', 2],
      ['jiading', 2],
      ['huxi', 1],
    ] as const) {
      const d = campusData(id),
        details = makeSportsDetails(d)
      const tracks = details.features.filter(
        (f) => f.properties?.part === 'track',
      )
      expect(tracks).toHaveLength(count)
      expect(
        details.features.filter((f) => f.properties?.part === 'infield'),
      ).toHaveLength(count)
      expect(
        details.features.filter((f) => f.properties?.part === 'lane').length,
      ).toBeGreaterThanOrEqual(count * 6)
      for (const f of tracks)
        expect(
          d.features.find((p) => String(p.id) === f.properties?.parent)
            ?.properties.building,
        ).toBeUndefined()
    }
  })
  it('keeps Hubei a straight running strip and names every traced building', () => {
    const d = campusData('hubei'),
      details = makeSportsDetails(d)
    const track = details.features.find((f) => f.properties?.part === 'track')!
    expect(track.geometry.type).toBe('Polygon')
    if (track.geometry.type === 'Polygon')
      expect(track.geometry.coordinates[0]).toHaveLength(5)
    expect(buildCatalog(d).filter((p) => p.indoor)).toHaveLength(14)
  })
})
describe('opt-in location', () => {
  it('rejects unavailable browsers and insecure contexts before requesting a fix', async () => {
    const call = vi.fn(),
      geo = { getCurrentPosition: call } as unknown as Geolocation
    await expect(requestLocation(geo, false)).rejects.toBe('insecure')
    await expect(requestLocation(undefined, true)).rejects.toBe('unsupported')
    expect(call).not.toHaveBeenCalled()
  })
  it('keeps the WGS84 fix and accuracy returned by the browser', async () => {
    const call = vi.fn((success: PositionCallback) =>
      success({
        coords: { longitude: 121.497, latitude: 31.285, accuracy: 18 },
        timestamp: 42,
      } as GeolocationPosition),
    )
    const point = await requestLocation(
      { getCurrentPosition: call } as unknown as Geolocation,
      true,
    )
    expect(point).toEqual({
      longitude: 121.497,
      latitude: 31.285,
      accuracy: 18,
      timestamp: 42,
    })
    expect(call.mock.calls[0]).toHaveLength(3)
    const features = locationData(point).features
    expect(features[1]!.geometry).toEqual({
      type: 'Point',
      coordinates: [121.497, 31.285],
    })
    const accuracy = features[0]!.geometry
    if (accuracy.type === 'Polygon') {
      expect(accuracy.coordinates[0]).toHaveLength(65)
      const north = accuracy.coordinates[0]![0]!
      expect((north[1]! - point.latitude) * 111195).toBeCloseTo(18, 1)
    }
    expect(locationData(null).features).toEqual([])
  })
  it.each([
    [1, 'denied'],
    [2, 'unavailable'],
    [3, 'timeout'],
  ])(
    'reports browser failure %s without fabricating a position',
    async (code, status) => {
      const geo = {
        getCurrentPosition: (
          _success: PositionCallback,
          error: PositionErrorCallback,
        ) => error({ code } as GeolocationPositionError),
      } as unknown as Geolocation
      await expect(requestLocation(geo, true)).rejects.toBe(status)
    },
  )
})

it('keeps the Hubei straight track west of the infield', () => {
  const details = makeSportsDetails(campusData('hubei')).features
  const centerX = (part: string) => {
    const f = details.find((f) => f.properties?.part === part)!
    if (f.geometry.type !== 'Polygon') throw new Error('Expected polygon')
    const ps = f.geometry.coordinates[0]!.slice(0, -1)
    return ps.reduce((sum, p) => sum + p[0]!, 0) / ps.length
  }
  expect(centerX('track')).toBeLessThan(centerX('infield'))
})

it('skips a degenerate track instead of aborting the sports details build', () => {
  // issue #700: a track footprint without any segment ≥3 m and under 10 m²
  // makes fieldFrame return undefined; the containment probe must skip it
  // rather than dereference it and crash the whole campus map style build.
  const degenerateTrack = {
    type: 'Feature',
    id: 'way/test-degenerate',
    properties: {
      category: 'sport',
      center: [121.5, 31.286],
      campus: true,
      leisure: 'track',
      name: '退化跑道',
    },
    geometry: {
      type: 'Polygon',
      coordinates: [
        [
          [121.5, 31.286],
          [121.500001, 31.286],
          [121.500001, 31.286001],
          [121.5, 31.286001],
          [121.5, 31.286],
        ],
      ],
    },
  }
  const pitch = {
    type: 'Feature',
    id: 'way/test-pitch',
    properties: {
      category: 'sport',
      center: [121.5, 31.2862],
      campus: true,
      sport: 'soccer',
      name: '测试球场',
    },
    geometry: {
      type: 'Polygon',
      coordinates: [
        [
          [121.4999, 31.286],
          [121.5001, 31.286],
          [121.5001, 31.2864],
          [121.4999, 31.2864],
          [121.4999, 31.286],
        ],
      ],
    },
  }
  const details = makeSportsDetails({
    type: 'FeatureCollection',
    features: [degenerateTrack, pitch],
  } as unknown as CampusData)
  expect(
    details.features.some((f) => f.properties?.parent === 'way/test-degenerate'),
  ).toBe(false)
  expect(
    details.features.some((f) => f.properties?.parent === 'way/test-pitch'),
  ).toBe(true)
})
