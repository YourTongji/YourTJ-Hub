import { readFileSync } from 'node:fs'
import { expect, it } from 'vitest'
import type { CampusData } from '../src/site/campus-map/catalog'
import { findLocationOverride, officialCampusId, officialLocationApplies, officialLocationTarget, officialLocationTargets, parseOfficialLocations, type OfficialMapLocation } from '../src/site/campus-map/official-location'
import table from '../src/site/campus-map/data/locations/2026-2027-1.json'

const context = { calendarId: 122 }
const data = Object.fromEntries(['siping', 'jiading', 'huxi'].map(campus => [campus,
  JSON.parse(readFileSync(new URL(`../src/site/campus-map/data/${campus}.geojson`, import.meta.url), 'utf8')) as CampusData]))

it('preserves all offline source members through equivalent parsing or pending exception overrides', () => {
  let count = 0
  for (const [campus, entries] of Object.entries(table.dictionary)) {
    for (const [raw, result] of Object.entries(entries)) {
      const locations = parseOfficialLocations(raw, campus, context)
      expect(locations).toHaveLength(result.locations.length)
      for (const [index, member] of result.locations.entries()) {
        expect(locations[index]).toMatchObject({ raw: member.source_text, building: member.place ?? '',
          room: member.detail ?? '', kind: member.kind, relation: result.relation, time: member.time,
          conditions: member.conditions, campusText: member.campus_text, address: member.address,
          unassignedConditions: result.unassigned_conditions, needsReview: result.needs_review,
          reviewPending: findLocationOverride(officialCampusId(campus)!, raw, context)?.reviewPending ?? false })
      }
      count++
    }
  }
  expect(count).toBe(702)
})

it('never maps flagged, nonphysical, unknown or ambiguous campus results', () => {
  for (const [campus, entries] of Object.entries(table.dictionary)) {
    for (const [raw, result] of Object.entries(entries)) {
      const locations = officialLocationTargets(campus, raw, data[officialCampusId(campus)!], context)
      for (const [index, member] of result.locations.entries()) {
        if (result.needs_review || !['named', 'generic'].includes(member.kind) ||
          (member.campus_text && officialCampusId(member.campus_text) !== officialCampusId(campus))) {
          expect(locations[index]?.target, `${campus} / ${raw}`).toBeUndefined()
        }
      }
    }
  }
})

it.each(['线上课堂', '不排教室', '学院安排'])("keeps nonphysical input %s classified", raw => {
  const [location] = officialLocationTargets('嘉定校区', raw, data.jiading, context)
  expect(location?.kind).toBe(({ '线上课堂': 'online', '不排教室': 'no_room', '学院安排': 'pending' })[raw])
  expect(location?.target).toBeUndefined()
})

it('keeps the user-confirmed faculty names visible without inventing a map identity', () => {
  const [location] = officialLocationTargets('嘉定校区', '学院专教', data.jiading, context)
  expect(location).toMatchObject({ building: '汽车与能源学院专教', kind: 'generic', needsReview: false })
  expect(location?.target).toBeUndefined()
  expect(officialLocationTargets('嘉定校区', '学院教室', data.jiading, context)[0]).toMatchObject({ needsReview: true, hint: 'review' })
})

it('accepts stable campus aliases and rooms across terms but never guesses unknown keys or campuses', () => {
  for (const [campus, raw] of [['嘉定校区', '北115'], ['四平路校区', '__proto__'], ['__proto__', '北115']] as const) {
    expect(officialLocationTargets(campus, raw, data.siping, context)[0]).toMatchObject({ raw, hint: 'missing' })
  }
  for (const campus of ['四平路校区', '四平']) {
    expect(officialLocationTarget(campus, '北115 ', data.siping, { calendarId: 121 })?.featureId).toBe('way/183383474')
  }
})

it('never mutates the shared dictionary through a returned value', () => {
  const raw = '单周致臻楼414，双周致臻楼416'
  const first = parseOfficialLocations(raw, '嘉定校区', context)
  first[0]!.time!.push('改变')
  first[0]!.unassignedConditions!.push('改变')
  expect(parseOfficialLocations(raw, '嘉定校区', context)[0]?.time).toEqual(['单周'])
  expect(parseOfficialLocations(raw, '嘉定校区', context)[0]?.unassignedConditions).toEqual([])
})

it('matches multiple explicit buildings but requires a choice even for multiple rooms in one building', () => {
  const raw = '瑞安楼403、505、507，物理馆319、301、302'
  const locations = officialLocationTargets('四平路校区', raw, data.siping, context)
  expect(locations.map(location => location.target?.featureId)).toEqual([
    'way/183383954', 'way/183383954', 'way/183383954', 'way/183383958', 'way/183383958', 'way/183383958',
  ])
  expect(officialLocationTarget('四平路校区', raw, data.siping, context)).toBeUndefined()
  expect(officialLocationTarget('四平路校区', '化学馆301,303', data.siping, context)).toBeUndefined()
})

it.each(['体育中心游泳馆', '体育中心篮球馆', '体育中心乒乓馆'])('maps %s to the recorded Jiading indoor sports building', raw => {
  const [location] = officialLocationTargets('嘉定校区', raw, data.jiading, context)
  expect(location).toMatchObject({ building: raw, room: '', target: { campusId: 'jiading', featureId: 'way/1456432428' } })
  const withoutHall: CampusData = { ...data.jiading!, features: data.jiading!.features.filter(feature => feature.id !== 'way/1456432428') }
  expect(officialLocationTarget('嘉定校区', raw, withoutHall, context)).toBeUndefined()
})

it('requires a unique feature and never falls back to a different campus or a missing curated feature', () => {
  const base = data.siping!.features.find(feature => feature.properties.name === '瑞安楼')!
  const ambiguous: CampusData = { type: 'FeatureCollection', features: [base, { ...base, id: 'duplicate' }] }
  expect(officialLocationTargets('四平路校区', '瑞安楼403、505、507，物理馆319、301、302', ambiguous, context).every(location => !location.target)).toBe(true)
  expect(officialLocationTarget('四平路校区', '北115', data.jiading, context)).toBeUndefined()
  expect(officialLocationTarget('四平路校区', '北115', data.siping, { ...context, dataCampusId: 'jiading' })).toBeUndefined()
  expect(officialCampusId('复旦大学四平校区')).toBeUndefined()
  expect(officialCampusId('嘉定校区、四平路校区')).toBeUndefined()
})

it('uses per-member time conditions from the exception override', () => {
  const locations = officialLocationTargets('嘉定校区', '单周致臻楼414，双周致臻楼416', data.jiading, context)
  expect(locations.map(location => officialLocationApplies(location, { week: 1, day: 2 }))).toEqual([true, false])
  expect(locations.map(location => officialLocationApplies(location, { week: 2, day: 2 }))).toEqual([false, true])
})

it.each([
  [['单周', '第3-4周'], 3, 1, true], [['单周', '第3-4周'], 4, 1, false],
  [['单周', '双周'], 1, 1, false], [['周四', '第3周'], 3, 4, true],
  [['周四', '第3周'], 3, 5, false], [['9月24日'], 2, 2, false],
  [['15:25～17:05，第一周自2026年9月7日始'], 1, 1, false],
  [['时间另行通知'], 1, 1, false], [['后8周'], 9, 1, false], [['前8周'], 8, 1, true],
] as const)('intersects assigned time conditions %j at week %i day %i', (time, week, day, applies) => {
  expect(officialLocationApplies({ raw: '', building: '', room: '', condition: '', time: [...time] }, { week, day })).toBe(applies)
})

it.each([
  { alternative: true }, { needsReview: true }, { unassignedConditions: ['其余'] }, { conditions: ['下雨时'] },
] satisfies Partial<OfficialMapLocation>[])('does not confirm scoped schedules when conditions are unresolved: %j', extra => {
  expect(officialLocationApplies({ raw: '', building: '', room: '', condition: '', time: null, ...extra }, { week: 1, day: 1 })).toBe(false)
})
