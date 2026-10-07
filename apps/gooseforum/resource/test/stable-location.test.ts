import { readFileSync } from 'node:fs'
import { expect, it } from 'vitest'
import type { CampusData } from '../src/site/campus-map/catalog'
import { officialLocationApplies, officialLocationTarget, officialLocationTargets } from '../src/site/campus-map/official-location'

const load = (campus: string) => JSON.parse(readFileSync(new URL(`../src/site/campus-map/data/${campus}.geojson`, import.meta.url), 'utf8')) as CampusData
const siping = load('siping'), jiading = load('jiading')

it.each([{}, { calendarId: 122 }, { calendarId: 123 }, { term: '未知新学期' }])('locates stable buildings and unseen simple rooms without term authorization: %j', context => {
  expect(officialLocationTarget('四平路校区', '北楼9999室', siping, context)?.featureId).toBe('way/183383474')
  expect(officialLocationTarget('嘉定校区', '博楼B999', jiading, context)?.featureId).toBe('way/263922904')
})

it.each(['北', '南', '北二楼', '南二楼'])('does not expand a north/south shorthand without a numeric room: %s', raw => {
  expect(officialLocationTarget('四平路校区', raw, siping)).toBeUndefined()
})

it('retains date suffixes but cannot confirm them from a teaching week', () => {
  const [location] = officialLocationTargets('四平路校区', '北楼115室（9月24日）', siping)
  expect(location).toMatchObject({ raw: '北楼115室（9月24日）', building: '北教学楼', room: '115室', time: ['9月24日'] })
  expect(location?.target?.featureId).toBe('way/183383474')
  expect(officialLocationApplies(location!, { week: 2, day: 2 })).toBe(false)
})

it('filters parity suffixes instead of silently treating them as unconditional', () => {
  const [location] = officialLocationTargets('四平路校区', '北楼115室（单周）', siping)
  expect(location?.time).toEqual(['单周'])
  expect(officialLocationApplies(location!, { week: 1, day: 2 })).toBe(true)
  expect(officialLocationApplies(location!, { week: 2, day: 2 })).toBe(false)
})

it('resolves each verified letter alias before considering room continuation', () => {
  const locations = officialLocationTargets('嘉定校区', 'A101、B201', jiading)
  expect(locations.map(location => location.target?.featureId)).toEqual(['way/266167562', 'way/263922904'])
  expect(officialLocationTarget('嘉定校区', 'A101、B201', jiading)).toBeUndefined()
  expect(officialLocationTargets('嘉定校区', '安楼A101、A102', jiading).map(location => location.target?.featureId)).toEqual(['way/266167562', 'way/266167562'])
  expect(officialLocationTargets('四平路校区', '工程试验馆303、307', siping).map(location => location.room)).toEqual(['303', '307'])
})

it('never inherits an unknown letter abbreviation or guesses an unknown building', () => {
  const locations = officialLocationTargets('嘉定校区', '安楼A101、Z201、202', jiading)
  expect(locations.map(location => location.target?.featureId)).toEqual(['way/266167562', undefined, undefined])
  expect(officialLocationTarget('四平路校区', '未知北楼115', siping)).toBeUndefined()
  expect(officialLocationTarget('未知校区', '北楼115', siping)).toBeUndefined()
})

it('preserves explicit floors without inferring a floor from room digits', () => {
  expect(officialLocationTargets('嘉定校区', '体育中心二楼', jiading)[0]?.room).toBe('二楼')
  expect(officialLocationTargets('嘉定校区', '博楼B110', jiading)[0]?.room).toBe('B110')
})

it('blocks schedules with unknown notes or ambiguous condition attachment', () => {
  for (const raw of ['北楼115室（时间另行通知）', '北楼115、116（单周）', '北楼115（单周）、南楼116']) {
    const locations = officialLocationTargets('四平路校区', raw, siping)
    expect(locations.every(location => !officialLocationApplies(location, { week: 1, day: 2 }))).toBe(true)
  }
})

it('keeps a source building alias with parentheses distinct from a time annotation', () => {
  const [location] = officialLocationTargets('嘉定校区', '博楼（B楼）', jiading)
  expect(location?.target?.featureId).toBe('way/263922904')
  expect(location?.conditions).toEqual([])
  expect(officialLocationApplies(location!, { week: 2, day: 2 })).toBe(true)
})

it('keeps a flagged override authoritative even without or outside the source term', () => {
  for (const context of [{}, { calendarId: 122 }, { calendarId: 123 }]) {
    const [location] = officialLocationTargets('嘉定校区', '学院教室', jiading, context)
    expect(location).toMatchObject({ needsReview: true, hint: 'review' })
    expect(location?.target).toBeUndefined()
  }
})
