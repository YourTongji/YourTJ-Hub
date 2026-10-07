import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { expect, it } from 'vitest'
import { assembleDictionary, checkAssets, mergeDictionaries, prepareRequests, validateDictionary, validateResult } from '../scripts/campus-locations/dictionary.mjs'
import table from '../src/site/campus-map/data/locations/2026-2027-1.json'
import schema from '../scripts/campus-locations/result.schema.json'
import type { ExtractedLocation, LocationExtraction, LocationKind, LocationRelation } from '../src/site/campus-map/location-types'

const member: ExtractedLocation = { source_text: '示例楼101', kind: 'named', place: '示例楼', detail: '101', campus_text: null, address: null, conditions: [], time: null }
const result: LocationExtraction = { relation: 'single', locations: [member], unassigned_conditions: [], needs_review: false, review_reasons: [] }

it('validates the complete published candidate and all v2 prompt examples', async () => {
  expect(await checkAssets()).toEqual({ inputs: 702, members: 779, flagged: 89, reviewStatus: 'review_2_pending', fewShotPairs: 12 })
  expect(schema.required.sort()).toEqual(Object.keys(result).sort())
  expect(schema.properties.locations.items.required.sort()).toEqual(Object.keys(member).sort())
  const kinds: Record<LocationKind, true> = { named: true, generic: true, online: true, pending: true, no_room: true, unknown: true }
  const relations: Record<LocationRelation, true> = { single: true, list: true, conditional: true, alternative: true, mixed: true, unclear: true }
  expect(schema.properties.locations.items.properties.kind.enum.sort()).toEqual(Object.keys(kinds).sort())
  expect(schema.properties.relation.enum.sort()).toEqual(Object.keys(relations).sort())
})

it('rejects schema drift, source rewrites and untraceable time conditions', () => {
  for (const change of [
    { ...member, time: [] }, { ...member, time: ['单周'] }, { ...member, source_text: '别的楼' },
    { ...member, kind: 'online' }, { ...member, target: 'invented' }, { ...member, kind: 'building' },
  ]) expect(() => validateResult({ ...result, locations: [change] }, member.source_text)).toThrow()
  const missing = structuredClone(result) as unknown as { locations: Record<string, unknown>[] }
  delete missing.locations[0]!.time
  expect(() => validateResult(missing, member.source_text)).toThrow()
  expect(() => validateDictionary({ ...table, _meta: { ...table._meta, input_count: 701 } })).toThrow(/input_count/)
})

it('merges only new same-term entries and never overwrites reviewed values or promotes status', () => {
  const candidate = { _meta: { ...table._meta, input_count: 1 }, dictionary: { '四平路校区': { [member.source_text]: result } } }
  const merged = mergeDictionaries({ ...table, _meta: { ...table._meta, review_status: 'reviewed' } }, candidate)
  expect(merged.dictionary['四平路校区']['北115']).toEqual(table.dictionary['四平路校区']['北115'])
  expect(merged._meta.review_status).toBe('review_2_pending')
  expect(merged._meta.input_count).toBe(703)
  expect(validateDictionary(table).inputs).toBe(702)
  expect(() => mergeDictionaries(table, { ...candidate, _meta: { ...candidate._meta, calendar_id: 123 } })).toThrow(/calendar_id/)
  const changed = structuredClone(table)
  changed.dictionary['四平路校区']['北115'].locations[0]!.detail = '999'
  expect(() => mergeDictionaries(table, changed)).toThrow(/Conflicting/)
})

it('prepares bounded, deduplicated offline prompts without overwriting inputs or including extra fields', async () => {
  const directory = await mkdtemp(join(tmpdir(), 'yourtj-location-request-'))
  try {
    const input = join(directory, 'input.json'), output = join(directory, 'requests')
    const entries = Array.from({ length: 21 }, (_, index) => ({ course_campus: '四平路校区', raw: `示例楼${index}` }))
    await writeFile(input, JSON.stringify([...entries, entries[0]]))
    expect(await prepareRequests(input, output, 122)).toEqual({ inputs: 21, batches: 2 })
    const request = JSON.parse(await readFile(join(output, 'batch-0001.json'), 'utf8'))
    expect(request.result_schema.required).toHaveLength(20)
    expect(request.messages[0].content).toContain('time')
    expect(request.messages.at(-1).role).toBe('user')
    expect(JSON.parse(request.messages.at(-1).content).entries).toHaveLength(20)
    const manifest = JSON.parse(await readFile(join(output, 'manifest.json'), 'utf8'))
    const results = Object.fromEntries(manifest.entries.map((entry: { id: string; raw: string }) => [entry.id,
      { ...result, locations: [{ ...member, source_text: entry.raw }] }]))
    const assembled = assembleDictionary(manifest, results, table._meta)
    expect(validateDictionary(assembled).inputs).toBe(21)
    expect(assembled._meta.review_status).toBe('review_2_pending')
    expect(() => assembleDictionary(manifest, {}, table._meta)).toThrow(/IDs/)
    expect(() => assembleDictionary(manifest, { ...results, extra: result }, table._meta)).toThrow(/IDs/)
    expect(() => assembleDictionary(manifest, results, { ...table._meta, calendar_id: 123 })).toThrow(/Calendar/)
    await expect(prepareRequests(input, output, 122)).rejects.toThrow()
    await writeFile(input, JSON.stringify([{ ...entries[0], studentId: 'must not be sent' }]))
    await expect(prepareRequests(input, join(directory, 'invalid'), 122)).rejects.toThrow(/only/)
  } finally { await rm(directory, { recursive: true, force: true }) }
})
