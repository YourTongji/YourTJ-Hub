import { createHash } from 'node:crypto'
import { readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createServer } from 'vite'

// Public, deduplicated PK inputs only. Outputs belong in ignored research/, not production assets.
const root = fileURLToPath(new URL('../../', import.meta.url))
const readJSON = async path => JSON.parse(await readFile(path, 'utf8'))
const [output, baselinePath] = process.argv.slice(2)
if (!output) throw new Error('Usage: audit.mjs new-output.json [baseline-report.json]')
const server = await createServer({ root, configFile: false, server: { middlewareMode: true }, optimizeDeps: { noDiscovery: true } })
try {
  const { officialCampusId, officialLocationTargets } = await server.ssrLoadModule('/src/site/campus-map/official-location.ts')
  const source = await readJSON(resolve(root, 'src/site/campus-map/data/locations/2026-2027-1.json'))
  const data = Object.fromEntries(await Promise.all(['siping', 'jiading', 'huxi'].map(async id => [id,
    await readJSON(resolve(root, `src/site/campus-map/data/${id}.geojson`))])))
  const rows = Object.entries(source.dictionary).flatMap(([campus, entries]) => Object.entries(entries).map(([raw]) => {
    const locations = officialLocationTargets(campus, raw, data[officialCampusId(campus)], { calendarId: source._meta.calendar_id })
    return { campus, raw, members: locations.map(location => ({ building: location.building, room: location.room,
      time: location.time ?? null, conditions: location.conditions ?? [], unassignedConditions: location.unassignedConditions ?? [],
      needsReview: location.needsReview ?? false, reviewPending: location.reviewPending ?? false,
      target: location.target?.featureId ?? null, hint: location.hint ?? null })) }
  }))
  const coverage = { full: 0, partial: 0, none: 0, members: 0, targets: 0 }
  for (const row of rows) {
    const targets = row.members.filter(member => member.target).length
    coverage[targets === row.members.length ? 'full' : targets ? 'partial' : 'none']++
    coverage.members += row.members.length
    coverage.targets += targets
  }
  const files = ['official-location.ts', 'location-types.ts'].map(path => `src/site/campus-map/${path}`)
  files.push('src/site/campus-map/data/locations/2026-2027-1.json', ...Object.keys(data).map(id => `src/site/campus-map/data/${id}.geojson`))
  for (const path of ['deterministic-location.ts', 'data/locations/places.json', 'data/locations/overrides.json']) {
    const file = `src/site/campus-map/${path}`
    try { await readFile(resolve(root, file)); files.push(file) } catch (error) { if (error.code !== 'ENOENT') throw error }
  }
  const hashes = Object.fromEntries(await Promise.all(files.map(async path => [path, createHash('sha256').update(await readFile(resolve(root, path))).digest('hex')])))
  const placeCandidates = Object.entries(source.dictionary).flatMap(([campus, entries]) =>
    [...new Set(Object.values(entries).flatMap(result => result.locations.map(member => member.place).filter(Boolean)))].sort().map(place => {
      const locations = officialLocationTargets(campus, place, data[officialCampusId(campus)])
      return { campus, place, stableTarget: locations.length === 1 ? locations[0].target?.featureId ?? null : null }
    }))
  let comparison
  if (baselinePath) {
    const baseline = await readJSON(baselinePath)
    const key = row => JSON.stringify([row.campus, row.raw])
    const before = new Map(baseline.rows.map(row => [key(row), row]))
    if (before.size !== rows.length || rows.some(row => !before.has(key(row)))) throw new Error('Comparison requires the exact same source keys')
    const changes = rows.flatMap(row => {
      const previous = before.get(key(row))
      return JSON.stringify(previous.members) === JSON.stringify(row.members) ? [] : [{ campus: row.campus, raw: row.raw, before: previous.members, after: row.members }]
    })
    comparison = { baselineHashes: baseline.hashes, coverageBefore: baseline.coverage, changedInputs: changes.length, changes }
  }
  const report = { source: 'Public undergraduate PK location inputs; coverage is not semantic accuracy', calendarId: source._meta.calendar_id,
    inputCount: rows.length, hashes, coverage, placeCandidates, rows, ...(comparison ? { comparison } : {}) }
  await writeFile(output, JSON.stringify(report, null, 2) + '\n', { flag: 'wx' })
  console.log(JSON.stringify({ inputs: rows.length, coverage, placeCandidates: placeCandidates.length, changedInputs: comparison?.changedInputs }, null, 2))
} finally { await server.close() }
