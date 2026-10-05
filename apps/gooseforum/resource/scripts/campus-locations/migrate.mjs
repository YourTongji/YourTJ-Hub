import { readFile, writeFile } from 'node:fs/promises'
import { basename, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createServer } from 'vite'
import { isDeepStrictEqual } from 'node:util'
import { validateDictionary } from './dictionary.mjs'

// Preparation only: never promotes an extraction or adds a model-produced building alias.
const root = fileURLToPath(new URL('../../', import.meta.url))
const [input, output] = process.argv.slice(2)
if (!input || !output) throw new Error('Usage: migrate.mjs dictionary.json new-candidate-overrides.json')
const table = JSON.parse(await readFile(input, 'utf8'))
validateDictionary(table)
const server = await createServer({ root, configFile: false, server: { middlewareMode: true }, optimizeDeps: { noDiscovery: true } })
try {
  const { parseStableLocation } = await server.ssrLoadModule('/src/site/campus-map/deterministic-location.ts')
  const { officialCampusId } = await server.ssrLoadModule('/src/site/campus-map/official-location.ts')
  const rules = []
  let deterministic = 0
  for (const [campus, entries] of Object.entries(table.dictionary)) {
    const campusId = officialCampusId(campus)
    if (!campusId) throw new Error(`Unregistered campus: ${campus}`)
    for (const [raw, result] of Object.entries(entries)) {
      const parsed = parseStableLocation(raw, campusId)
      if (!result.needs_review && isDeepStrictEqual(parsed, result)) { deterministic++; continue }
      // Only faculty/context-derived interpretations need a term. Explicit text and blocks do not.
      const facultyDerived = result.locations.some(member => member.place && !raw.includes(member.place) &&
        /学院|专教|实验室|机房/u.test(raw) && !parsed.locations.some(value => value.place === member.place))
      const action = result.needs_review ? 'block' : 'replace'
      rules.push({ campusId, raw, ...(facultyDerived && action !== 'block' ? { scope: { calendarId: table._meta.calendar_id, termNames: table._meta.term_names } } : {}),
        action, result, reviewPending: table._meta.review_status !== 'reviewed',
        reason: action === 'block' ? 'Preserved extraction concern; no parsing fallback' : facultyDerived ? 'Faculty-derived interpretation; source-term only' : 'Explicit non-simple source text or retained correction; not a general building alias',
        source: `${basename(input)}#${table._meta.candidate_sha256}` })
    }
  }
  const candidate = { schema_version: 'campus-location-overrides/v1', rules }
  await writeFile(output, JSON.stringify(candidate, null, 2) + '\n', { flag: 'wx' })
  console.log({ deterministic, overrides: rules.length, blocked: rules.filter(rule => rule.action === 'block').length,
    termScoped: rules.filter(rule => rule.scope).length, reviewPending: rules.filter(rule => rule.reviewPending).length })
} finally { await server.close() }
