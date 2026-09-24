import { expect, it } from 'vitest'
import type { CampusData } from '../src/site/campus-map/catalog'
import { officialCampusId, officialLocationTarget, parseOfficialLocation } from '../src/site/campus-map/official-location'

it('splits only a clearly separated building phrase and room token for display', () => {
  expect(parseOfficialLocation('济事楼（软件学院） A101')).toEqual({
    building: '济事楼（软件学院）',
    room: 'A101',
  })
})

it.each([
  ['北115', { building: '北', room: '115' }],
  ['济事楼201', { building: '济事楼', room: '201' }],
  ['教学北楼 115 室', { building: '教学北楼', room: '115' }],
  ['嘉定校区 D404', { building: 'D', room: '404' }],
])('splits an explicit building prefix and room number for display: %s', (value, expected) => {
  expect(parseOfficialLocation(value, value.startsWith('嘉定') ? '嘉定校区' : '')).toEqual(expected)
})

it.each(['未知地点', '2024秋季1班'])('%s remains raw when it has no reliable building-room boundary', (value) => {
  expect(parseOfficialLocation(value)).toBeNull()
})

it.each([
  ['四平路校区', '北楼', { campusId: 'siping', featureId: 'way/183383474' }],
  ['四平路校区', '教学北楼', { campusId: 'siping', featureId: 'way/183383474' }],
  ['四平路校区', '南楼', { campusId: 'siping', featureId: 'way/183383472' }],
  ['四平路校区', '教学南楼', { campusId: 'siping', featureId: 'way/183383472' }],
  ['四平路校区', '北115', { campusId: 'siping', featureId: 'way/183383474' }],
  ['四平路校区', '北楼 115', { campusId: 'siping', featureId: 'way/183383474' }],
  ['四平路校区', '教学北楼115', { campusId: 'siping', featureId: 'way/183383474' }],
  ['四平路校区', '南楼 203', { campusId: 'siping', featureId: 'way/183383472' }],
  ['四平路校区', '教学南楼203', { campusId: 'siping', featureId: 'way/183383472' }],
  ['嘉定校区', '济事楼', { campusId: 'jiading', featureId: 'way/135405205' }],
  ['嘉定校区', '济事南楼', { campusId: 'jiading', featureId: 'way/135405205' }],
  ['嘉定校区', '济事北楼', { campusId: 'jiading', featureId: 'way/135405205' }],
  ['嘉定校区', '济事楼（软件学院）', { campusId: 'jiading', featureId: 'way/135405205' }],
  ['嘉定校区', '济事南楼 A101', { campusId: 'jiading', featureId: 'way/135405205' }],
  ['嘉定校区', '济事北楼A101', { campusId: 'jiading', featureId: 'way/135405205' }],
  ['嘉定校区', '济事楼（软件学院） A101', { campusId: 'jiading', featureId: 'way/135405205' }],
])('maps only confirmed campus/building aliases: %s %s', (campus, room, target) => {
  expect(officialLocationTarget(campus, room)).toEqual(target)
})

function mapData(...names: [string, string][]): CampusData {
  return {
    type: 'FeatureCollection',
    features: names.map(([id, name]) => ({
      type: 'Feature',
      id,
      properties: { category: 'academic', center: [121, 31], campus: true, name },
      geometry: { type: 'Point', coordinates: [121, 31] },
    })),
  } as CampusData
}

it('maps obvious course building abbreviations from the selected campus dataset', () => {
  const jiading = mapData(['a', '安楼（A楼）'], ['b', '博楼（B楼）'])
  expect(officialCampusId('同济大学嘉定校区')).toBe('jiading')
  expect(officialLocationTarget('嘉定校区', 'D404', jiading)).toBeUndefined()
  expect(officialLocationTarget('嘉定校区', 'A101', jiading)).toEqual({ campusId: 'jiading', featureId: 'a' })
  expect(officialLocationTarget('嘉定校区', '博楼201', jiading)).toEqual({ campusId: 'jiading', featureId: 'b' })
})

it('does not guess when a course record names multiple campuses', () => {
  expect(officialCampusId('四平路校区、嘉定校区')).toBeUndefined()
})

it('does not pin an abbreviation when it maps to multiple buildings', () => {
  const data = mapData(['a', '甲楼（A楼）'], ['b', '乙楼（A楼）'])
  expect(officialLocationTarget('嘉定校区', 'A101', data)).toBeUndefined()
})

it.each([
  ['嘉定校区', '北115'],
  ['四平路校区', '济事北楼 A101'],
  ['沪西校区', '北楼115'],
  ['四平路校区', '北'],
  ['四平路校区', '南'],
])('does not guess a map building for unconfirmed or incomplete locations: %s %s', (campus, room) => {
  expect(officialLocationTarget(campus, room)).toBeUndefined()
})
