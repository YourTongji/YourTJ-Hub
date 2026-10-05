import { readFileSync } from 'node:fs'
import { expect, it } from 'vitest'
import type { CampusData } from '../src/site/campus-map/catalog'
import { officialLocationApplies, officialLocationTargets } from '../src/site/campus-map/official-location'

const siping = JSON.parse(readFileSync(new URL('../src/site/campus-map/data/siping.geojson', import.meta.url), 'utf8')) as CampusData
const jiading = JSON.parse(readFileSync(new URL('../src/site/campus-map/data/jiading.geojson', import.meta.url), 'utf8')) as CampusData

it('uses the offline north-building correction and preserves the extracted member type', () => {
  const [location] = officialLocationTargets('四平路校区', '北115', siping, { calendarId: 122 })
  expect(location).toMatchObject({ building: '北教学楼', room: '115', kind: 'named', time: null })
  expect(location?.target?.featureId).toBe('way/183383474')
})

it('does not fall back to the retired parser for an unseen room', () => {
  const raw = '北楼9999室（单周）'
  const [location] = officialLocationTargets('四平路校区', raw, siping, { calendarId: 122 })
  expect(location).toMatchObject({ raw, building: '', room: '', hint: 'missing' })
  expect(location?.target).toBeUndefined()
})

// Original PR review counterexamples must stay unlocated until an exact dictionary entry exists.
it.each(['北楼115室（9月24日）', '北楼115室（单周）'])(
  'does not swallow the unrecorded time suffix in %s', raw => {
    const locations = officialLocationTargets('四平路校区', raw, siping, { calendarId: 122 })
    expect(locations).toHaveLength(1)
    expect(locations[0]).toMatchObject({ raw, building: '', room: '', hint: 'missing' })
    expect(locations[0]?.target).toBeUndefined()
    for (const week of [1, 2]) expect(officialLocationApplies(locations[0]!, { week, day: 2 })).toBe(false)
  },
)

it.each(['A101', 'B201', 'A101、B201', '安楼A101、A102', '济事楼A101、B201、202'])(
  'does not infer or inherit a building for the unrecorded input %s', raw => {
    const locations = officialLocationTargets('嘉定校区', raw, jiading, { calendarId: 122 })
    expect(locations).toHaveLength(1)
    expect(locations[0]).toMatchObject({ raw, building: '', room: '', hint: 'missing' })
    expect(locations[0]?.target).toBeUndefined()
    expect(officialLocationApplies(locations[0]!, { week: 2, day: 2 })).toBe(false)
  },
)

it('does not reuse semester-specific extraction for a different or unspecified semester', () => {
  for (const context of [{ calendarId: 121 }, {}]) {
    const [location] = officialLocationTargets('四平路校区', '北115', siping, context)
    expect(location?.target).toBeUndefined()
    expect(location?.hint).toBe('missing')
  }
})
