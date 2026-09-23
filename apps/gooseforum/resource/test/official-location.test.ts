import { expect, it } from 'vitest'
import { officialLocationTarget, parseOfficialLocation } from '../src/site/campus-map/official-location'

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
])('splits an explicit building prefix and room number for display: %s', (value, expected) => {
  expect(parseOfficialLocation(value)).toEqual(expected)
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

it.each([
  ['嘉定校区', '北115'],
  ['四平路校区', '济事北楼 A101'],
  ['沪西校区', '北楼115'],
  ['四平路校区', '北'],
  ['四平路校区', '南'],
  ['四平校区', '北楼115'],
])('does not guess a map building for unconfirmed or incomplete locations: %s %s', (campus, room) => {
  expect(officialLocationTarget(campus, room)).toBeUndefined()
})
