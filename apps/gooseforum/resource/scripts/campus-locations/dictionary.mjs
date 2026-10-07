import { createHash } from 'node:crypto'
import { readFile, writeFile, mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import { isDeepStrictEqual } from 'node:util'
import Ajv2020 from 'ajv/dist/2020.js'

const readJSON = async path => JSON.parse(await readFile(path, 'utf8'))
const asset = new URL('../../src/site/campus-map/data/locations/2026-2027-1.json', import.meta.url)
const schema = await readJSON(new URL('result.schema.json', import.meta.url))
const ajv = new Ajv2020({ allErrors: true })
const checkResult = ajv.compile(schema)
const string = { type: 'string', minLength: 1 }
const hash = { type: 'string', pattern: '^[a-f0-9]{64}$' }
const metaProperties = {
  schema_version: { const: 'campus-location-extraction/v2' },
  calendar_id: { type: 'integer', minimum: 1 }, term: string,
  term_names: { type: 'array', minItems: 1, uniqueItems: true, items: string },
  audience: { const: 'undergraduate_pk_aggregate' }, synced_at: { type: 'string', pattern: '^\\d{4}-\\d{2}-\\d{2}$' },
  review_status: { enum: ['review_2_pending', 'reviewed'] }, input_count: { type: 'integer', minimum: 1 },
  source_sha256: hash, candidate_sha256: hash, model_requested: string,
  run_fingerprint: hash, source_prompt_sha256: hash, correction_rule: string,
}
const sourceMeta = { type: 'object', additionalProperties: false, required: Object.keys(metaProperties), properties: metaProperties }
const checkDictionary = ajv.compile({ type: 'object', additionalProperties: false, required: ['_meta', 'dictionary'], properties: {
  _meta: { ...sourceMeta, properties: { ...metaProperties, sources: { type: 'array', minItems: 2, items: sourceMeta } } },
  dictionary: { type: 'object', minProperties: 1, propertyNames: string,
    additionalProperties: { type: 'object', minProperties: 1, propertyNames: string, additionalProperties: schema } },
} })
const digest = value => createHash('sha256').update(value).digest('hex')
const same = isDeepStrictEqual

export function validateResult(result, raw) {
  if (!checkResult(result)) throw new Error(ajv.errorsText(checkResult.errors))
  for (const location of result.locations) {
    if (!raw.includes(location.source_text)) throw new Error('source_text must be an exact source substring')
    for (const condition of [...location.conditions, ...(location.time ?? [])]) {
      if (!raw.includes(condition)) throw new Error('Conditions must preserve source text')
    }
    if (['online', 'pending', 'no_room'].includes(location.kind) && location.place !== null)
      throw new Error('Nonphysical members cannot name a physical place')
  }
  for (const condition of result.unassigned_conditions) {
    if (!raw.includes(condition)) throw new Error('Unassigned conditions must preserve source text')
  }
  if (result.relation === 'single' && result.locations.length !== 1) throw new Error('single requires one member')
}

export function validateDictionary(table) {
  if (!checkDictionary(table)) throw new Error(ajv.errorsText(checkDictionary.errors))
  if (!table._meta.term_names.includes(table._meta.term)) throw new Error('term_names must include the source term')
  let inputs = 0, members = 0, flagged = 0
  for (const entries of Object.values(table.dictionary)) {
    for (const [raw, result] of Object.entries(entries)) {
      validateResult(result, raw)
      inputs++
      members += result.locations.length
      if (result.needs_review) flagged++
    }
  }
  if (inputs !== table._meta.input_count) throw new Error('input_count does not match dictionary keys')
  return { inputs, members, flagged, reviewStatus: table._meta.review_status }
}

export function mergeDictionaries(current, candidate) {
  validateDictionary(current)
  validateDictionary(candidate)
  for (const key of ['schema_version', 'calendar_id', 'term', 'term_names', 'audience']) {
    if (!same(current._meta[key], candidate._meta[key])) throw new Error(`Cannot merge different ${key}`)
  }
  const merged = structuredClone(current)
  for (const [campus, entries] of Object.entries(candidate.dictionary)) {
    for (const [raw, value] of Object.entries(entries)) {
      const oldEntries = Object.hasOwn(merged.dictionary, campus) ? merged.dictionary[campus] : undefined
      if (oldEntries && Object.hasOwn(oldEntries, raw)) {
        if (!same(oldEntries[raw], value)) throw new Error(`Conflicting entry: ${campus} / ${raw}; review explicitly`)
      } else {
        // Define own properties so even source keys such as __proto__ remain ordinary data.
        if (!oldEntries) Object.defineProperty(merged.dictionary, campus, { value: {}, enumerable: true, configurable: true })
        Object.defineProperty(merged.dictionary[campus], raw, { value, enumerable: true, configurable: true })
      }
    }
  }
  merged._meta.input_count = Object.values(merged.dictionary).reduce((count, entries) => count + Object.keys(entries).length, 0)
  // A merged candidate is never silently promoted to reviewed, nor attributed to one source run.
  merged._meta.review_status = 'review_2_pending'
  merged._meta.sources = [current, candidate].flatMap(table => table._meta.sources ?? [table._meta])
  merged._meta.source_sha256 = digest(JSON.stringify([current._meta.source_sha256, candidate._meta.source_sha256]))
  merged._meta.candidate_sha256 = digest(JSON.stringify([current, candidate]))
  merged._meta.run_fingerprint = digest(JSON.stringify([current._meta.run_fingerprint, candidate._meta.run_fingerprint]))
  merged._meta.source_prompt_sha256 = digest(JSON.stringify([current._meta.source_prompt_sha256, candidate._meta.source_prompt_sha256]))
  merged._meta.correction_rule = 'explicit-incremental-merge/v1'
  merged._meta.model_requested = [current._meta.model_requested, candidate._meta.model_requested].filter((value, index, all) => all.indexOf(value) === index).join('; ')
  merged._meta.synced_at = current._meta.synced_at < candidate._meta.synced_at ? current._meta.synced_at : candidate._meta.synced_at
  validateDictionary(merged)
  return merged
}

export async function checkAssets() {
  const table = await readJSON(asset)
  const result = validateDictionary(table)
  const messages = await readJSON(new URL('prompt.fewshot.messages.json', import.meta.url))
  if (messages.length % 2) throw new Error('Few-shot messages must be user/assistant pairs')
  for (let index = 0; index < messages.length; index += 2) {
    const input = messages[index], output = messages[index + 1]
    if (input.role !== 'user' || output.role !== 'assistant') throw new Error('Invalid few-shot roles')
    const entries = JSON.parse(input.content).entries
    const results = JSON.parse(output.content)
    if (!same(Object.keys(results).sort(), entries.map(entry => entry.id).sort())) throw new Error('Few-shot IDs differ')
    for (const entry of entries) validateResult(results[entry.id], entry.raw)
  }
  return { ...result, fewShotPairs: messages.length / 2 }
}

export function assembleDictionary(manifest, results, metadata) {
  if (manifest.calendarId !== metadata.calendar_id) throw new Error('Calendar differs from frozen inputs')
  if (!Array.isArray(manifest.entries) || !manifest.entries.length || !results || typeof results !== 'object' || Array.isArray(results))
    throw new Error('Expected frozen inputs and a result object')
  const ids = manifest.entries.map(entry => entry.id)
  if (new Set(ids).size !== ids.length || !same([...ids].sort(), Object.keys(results).sort()))
    throw new Error('Missing, duplicate or extra result IDs')
  const entries = new Map()
  for (const entry of manifest.entries) {
    if (!entry || Object.keys(entry).sort().join(',') !== 'course_campus,id,raw' ||
      typeof entry.course_campus !== 'string' || !entry.course_campus || typeof entry.raw !== 'string' || !entry.raw)
      throw new Error('Invalid frozen source entry')
    if (entry.id !== digest(JSON.stringify([manifest.calendarId, entry.course_campus, entry.raw])))
      throw new Error('Frozen input ID no longer matches its source')
    validateResult(results[entry.id], entry.raw)
    if (!entries.has(entry.course_campus)) entries.set(entry.course_campus, [])
    entries.get(entry.course_campus).push([entry.raw, results[entry.id]])
  }
  const { sources: _sources, ...source } = metadata
  const table = { _meta: { ...source, review_status: 'review_2_pending', input_count: ids.length,
    source_sha256: manifest.source_sha256, source_prompt_sha256: manifest.prompt_sha256,
    candidate_sha256: digest(JSON.stringify(results)), run_fingerprint: digest(JSON.stringify([manifest, results])),
    correction_rule: 'offline-extraction/v2' },
  dictionary: Object.fromEntries([...entries].map(([campus, values]) => [campus, Object.fromEntries(values)])) }
  validateDictionary(table)
  return table
}

// Produces request materials only: no model client, credential file or network access.
export async function prepareRequests(inputPath, outputDirectory, calendarId) {
  if (!Number.isSafeInteger(calendarId) || calendarId < 1) throw new Error('A positive calendar ID is required')
  const source = await readJSON(inputPath)
  if (!Array.isArray(source) || !source.length) throw new Error('Expected a nonempty array of {course_campus, raw}')
  const entries = [], seen = new Set()
  for (const entry of source) {
    if (!entry || Object.keys(entry).sort().join(',') !== 'course_campus,raw' ||
      typeof entry.course_campus !== 'string' || !entry.course_campus || typeof entry.raw !== 'string' || !entry.raw)
      throw new Error('Inputs must contain only nonempty course_campus and raw strings')
    const key = JSON.stringify([calendarId, entry.course_campus, entry.raw])
    if (seen.has(key)) continue
    seen.add(key)
    entries.push({ id: digest(key), ...entry })
  }
  const system = await readFile(new URL('prompt.system.zh.txt', import.meta.url), 'utf8')
  const examples = await readJSON(new URL('prompt.fewshot.messages.json', import.meta.url))
  // Refuse to overwrite existing requests or human edits.
  await mkdir(outputDirectory, { recursive: false })
  await writeFile(resolve(outputDirectory, 'manifest.json'), JSON.stringify({ calendarId, entries,
    source_sha256: digest(JSON.stringify(source)), prompt_sha256: digest(system),
    examples_sha256: digest(JSON.stringify(examples)), schema_sha256: digest(JSON.stringify(schema)),
    review_status: 'review_2_pending' }, null, 2) + '\n', { flag: 'wx' })
  for (let offset = 0; offset < entries.length; offset += 20) {
    const batch = entries.slice(offset, offset + 20)
    const request = { messages: [{ role: 'system', content: system }, ...examples,
      { role: 'user', content: JSON.stringify({ entries: batch }) }],
    result_schema: { type: 'object', additionalProperties: false, required: batch.map(entry => entry.id),
      properties: Object.fromEntries(batch.map(entry => [entry.id, schema])) } }
    await writeFile(resolve(outputDirectory, `batch-${String(offset / 20 + 1).padStart(4, '0')}.json`), JSON.stringify(request, null, 2) + '\n', { flag: 'wx' })
  }
  return { inputs: entries.length, batches: Math.ceil(entries.length / 20) }
}

async function main() {
  const [command = 'check', ...args] = process.argv.slice(2)
  if (command === 'check') console.log(args[0] ? validateDictionary(await readJSON(args[0])) : await checkAssets())
  else if (command === 'prepare' && args.length === 3) console.log(await prepareRequests(args[0], args[1], Number(args[2])))
  else if (command === 'assemble' && args.length === 4) {
    const table = assembleDictionary(await readJSON(args[0]), await readJSON(args[1]), await readJSON(args[2]))
    await writeFile(args[3], JSON.stringify(table, null, 2) + '\n', { flag: 'wx' })
    console.log(validateDictionary(table))
  }
  else if (command === 'merge' && args.length === 3) {
    const merged = mergeDictionaries(await readJSON(args[0]), await readJSON(args[1]))
    await writeFile(args[2], JSON.stringify(merged, null, 2) + '\n', { flag: 'wx' })
    console.log(validateDictionary(merged))
  } else throw new Error('Usage: dictionary.mjs check [dictionary] | prepare inputs output-dir calendar-id | assemble manifest results metadata new-output | merge current candidate new-output')
}
if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  main().catch(error => { console.error(error.message); process.exitCode = 1 })
}
