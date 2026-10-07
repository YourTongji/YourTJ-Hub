import { expect, it } from 'vitest'
import { findLocationOverride, officialLocationTargets } from '../src/site/campus-map/official-location'
import { parseStableLocation } from '../src/site/campus-map/deterministic-location'
import type { LocationOverride, StablePlace } from '../src/site/campus-map/location-types'
import places from '../src/site/campus-map/data/locations/places.json'
import overrides from '../src/site/campus-map/data/locations/overrides.json'
import { checkRuntimeAssets, validateRuntimeConfig } from '../scripts/campus-locations/runtime-config.mjs'

const rule: LocationOverride = { campusId: 'siping', raw: '北楼999室', action: 'replace', reviewPending: false,
  reason: 'Synthetic scope test', source: 'test', result: { relation: 'single', needs_review: false, review_reasons: [], unassigned_conditions: [],
    locations: [{ source_text: '北楼999室', kind: 'named', place: '南教学楼', detail: '999室', campus_text: null, address: null, conditions: [], time: null }] } }

it('validates runtime schemas, source snippets, identities and pending migrated exceptions', async () => {
  expect(await checkRuntimeAssets()).toEqual({ places: 37, overrides: 370, blocked: 89, termScoped: 39 })
})

it('prefers a matching scoped override, never leaks it, and checks both provided term identities', () => {
  const scoped: LocationOverride = { ...rule, scope: { calendarId: 122, termNames: ['已注册学期'] } }
  expect(findLocationOverride('siping', rule.raw, {}, [scoped])).toBeUndefined()
  expect(findLocationOverride('siping', rule.raw, { calendarId: 123 }, [scoped])).toBeUndefined()
  expect(findLocationOverride('siping', rule.raw, { calendarId: 123, term: '已注册学期' }, [scoped])).toBeUndefined()
  expect(findLocationOverride('siping', rule.raw, { term: '已注册学期' }, [rule, scoped])).toBe(scoped)
  expect(findLocationOverride('siping', rule.raw, { calendarId: 123 }, [rule, scoped])).toBe(rule)
  expect(findLocationOverride('siping', rule.raw + ' ', { calendarId: 122 }, [scoped])).toBeUndefined()
})

it('does not reuse faculty-derived interpretations in another term', () => {
  expect(officialLocationTargets('嘉定校区', '学院专教', null, { calendarId: 122 })[0]?.building).toBe('汽车与能源学院专教')
  for (const context of [{}, { calendarId: 123 }]) {
    expect(officialLocationTargets('嘉定校区', '学院专教', null, context)[0]).toMatchObject({ building: '', hint: 'missing' })
  }
})

it('does not add semester scope to an explicit generic source place', () => {
  for (const context of [{}, { calendarId: 122 }, { calendarId: 123 }]) {
    expect(officialLocationTargets('嘉定校区', '嘉定机房F214A', null, context)[0]).toMatchObject({ building: '嘉定机房', kind: 'generic' })
  }
})

it('rejects duplicate scopes, source rewrites, stray fields and nonblocking concerns', () => {
  for (const rules of [
    [rule, rule], [{ ...rule, extra: true }], [{ ...rule, result: { ...rule.result, needs_review: true } }],
    [{ ...rule, result: { ...rule.result, locations: [{ ...rule.result.locations[0], source_text: '篡改' }] } }],
    [{ ...rule, scope: { calendarId: 122, termNames: ['同名学期'] } }, { ...rule, scope: { calendarId: 123, termNames: ['同名学期'] } }],
  ]) expect(() => validateRuntimeConfig(places, { schema_version: 'campus-location-overrides/v1', rules })).toThrow()
})

it('keeps duplicate building identities ambiguous, and does not match inside unknown names', () => {
  const place: StablePlace = { name: '楼馆', aliases: ['甲楼'], featureId: 'one', evidence: 'test' }
  const catalog = { siping: [place, { ...place, featureId: 'two' }] }
  expect(parseStableLocation('甲楼101', 'siping', catalog).locations[0]?.place).toBeNull()
  expect(parseStableLocation('未知甲楼101', 'siping', { siping: [place] }).locations[0]?.place).toBeNull()
})

it('has an effective block override even when its text can be parsed by the catalog', () => {
  const blocked = overrides.rules.find(value => value.action === 'block' && parseStableLocation(value.raw, value.campusId).locations.some(member => member.place))!
  expect(blocked).toBeDefined()
  const campus = blocked.campusId === 'siping' ? '四平路校区' : '嘉定校区'
  for (const context of [{}, { calendarId: 122 }, { calendarId: 123 }]) {
    const locations = officialLocationTargets(campus, blocked.raw, null, context)
    expect(locations.every(location => !location.target && location.needsReview)).toBe(true)
  }
})
