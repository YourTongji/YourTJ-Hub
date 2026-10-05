import { readFileSync } from 'node:fs'
import { expect, it } from 'vitest'
import type { CampusData } from '../src/site/campus-map/catalog'
import { officialLocationTargets } from '../src/site/campus-map/official-location'

const siping = JSON.parse(readFileSync(new URL('../src/site/campus-map/data/siping.geojson', import.meta.url), 'utf8')) as CampusData

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

it('does not reuse semester-specific extraction for a different or unspecified semester', () => {
  for (const context of [{ calendarId: 121 }, {}]) {
    const [location] = officialLocationTargets('四平路校区', '北115', siping, context)
    expect(location?.target).toBeUndefined()
    expect(location?.hint).toBe('missing')
  }
})
