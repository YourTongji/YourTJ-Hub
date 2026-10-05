import { expect, it } from 'vitest'
import { auditPlaceText, matchPlaceText } from '../scripts/campus-locations/audit-place-text.mjs'
import table from '../src/site/campus-map/data/locations/2026-2027-1.json'
import { parseOfficialLocations } from '../src/site/campus-map/official-location'

const places = ['南教学楼', '北教学楼']

it.each([['南101', '南教学楼', '101'], ['北115室', '北教学楼', '115室']])('recognizes %s without requiring any map feature', (raw, place, room) => {
  expect(matchPlaceText(raw, places, '四平路校区')).toEqual([{ place, room, sourceText: raw, index: 0,
    basis: 'user-confirmed-north-south-numeric-room' }])
})

it('recognizes both directions independently and requires digits and the source campus', () => {
  expect(matchPlaceText('北115、南101', places, '四平路校区').map((match: { place: string }) => match.place)).toEqual(['北教学楼', '南教学楼'])
  for (const raw of ['南', '北', '南二楼', '北楼']) expect(matchPlaceText(raw, places, '四平路校区')).toEqual([])
  expect(matchPlaceText('北115', places, '嘉定校区')).toEqual([])
})

it('prefers a full candidate name to an overlapping substring', () => {
  expect(matchPlaceText('工程实践中心A楼', ['工程实践中心', '工程实践中心A楼'], '四平路校区').map((match: { place: string }) => match.place)).toEqual(['工程实践中心A楼'])
})

it('recognizes all 143 source shorthands consistently with the runtime parser', () => {
  const rows = Object.keys(table.dictionary['四平路校区']).filter(raw => /^[南北]\d+$/u.test(raw))
  expect(rows).toHaveLength(143)
  for (const raw of rows) {
    const place = raw.startsWith('南') ? '南教学楼' : '北教学楼'
    expect(matchPlaceText(raw, places, '四平路校区')[0]).toMatchObject({ place, room: raw.slice(1) })
    expect(parseOfficialLocations(raw, '四平路校区', {})[0]).toMatchObject({ building: place, room: raw.slice(1), kind: 'named' })
  }
})

it('reports place recognition separately from map targets and does not count nonphysical text as a place', () => {
  const report = auditPlaceText(table)
  expect(report).toMatchObject({ inputs: 702, matched: 685, unmatched: 17 })
  expect(report.campuses['四平路校区']).toMatchObject({ matched: 403, unmatched: 5 })
  expect(report.rows.filter((row: { raw: string }) => ['线上课堂', '不排教室'].includes(row.raw)).every((row: { matches: unknown[] }) => row.matches.length === 0)).toBe(true)
})
