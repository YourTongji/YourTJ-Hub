import { readFileSync } from 'node:fs'
import { expect, it } from 'vitest'
import type { CampusData } from '../src/site/campus-map/catalog'
import { officialLocationApplies, officialLocationTarget, officialLocationTargets } from '../src/site/campus-map/official-location'

const siping = JSON.parse(readFileSync(new URL('../src/site/campus-map/data/siping.geojson', import.meta.url), 'utf8')) as CampusData
const jiading = JSON.parse(readFileSync(new URL('../src/site/campus-map/data/jiading.geojson', import.meta.url), 'utf8')) as CampusData

it('reuses the confirmed north shorthand without depending on an exact room or term entry', () => {
  for (const context of [{ calendarId: 122 }, { calendarId: 121 }, {}]) {
    const [location] = officialLocationTargets('四平路校区', '北115', siping, context)
    expect(location).toMatchObject({ building: '北教学楼', room: '115', kind: 'named', time: null })
    expect(location?.target?.featureId).toBe('way/183383474')
  }
})

it('does not run broad old heuristics on an unknown building', () => {
  const raw = '未知教学楼9999室（单周）'
  const [location] = officialLocationTargets('四平路校区', raw, siping)
  expect(location).toMatchObject({ raw, building: '', room: '', hint: 'missing' })
  expect(location?.target).toBeUndefined()
  expect(officialLocationApplies(location!, { week: 1, day: 2 })).toBe(false)
})

// The original blocker counterexamples remain tests, now asserting the intended parsed semantics.
it.each(['北楼115室（9月24日）', '北楼115室（单周）'])(
  'retains the original review time suffix in %s', raw => {
    const [location] = officialLocationTargets('四平路校区', raw, siping)
    expect(location).toMatchObject({ raw, building: '北教学楼', room: '115室' })
    expect(location?.time).toEqual([raw.includes('单周') ? '单周' : '9月24日'])
    expect(officialLocationApplies(location!, { week: 2, day: 2 })).toBe(false)
    expect(officialLocationApplies(location!, { week: 1, day: 2 })).toBe(raw.includes('单周'))
  },
)

it.each([
  ['A101', ['way/266167562']], ['B201', ['way/263922904']],
  ['A101、B201', ['way/266167562', 'way/263922904']],
  ['安楼A101、A102', ['way/266167562', 'way/266167562']],
  ['济事楼A101、B201、202', ['way/135405205', 'way/263922904', 'way/263922904']],
] as const)('does not merge a verified cross-building abbreviation in %s', (raw, ids) => {
  const locations = officialLocationTargets('嘉定校区', raw, jiading)
  expect(locations.map(location => location.target?.featureId)).toEqual(ids)
  if (ids.length > 1) expect(officialLocationTarget('嘉定校区', raw, jiading)).toBeUndefined()
})
