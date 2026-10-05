import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import Ajv2020 from 'ajv/dist/2020.js'
import { validateResult } from './dictionary.mjs'

const readJSON = async path => JSON.parse(await readFile(path, 'utf8'))
const ajv = new Ajv2020({ allErrors: true })
ajv.addSchema(await readJSON(new URL('result.schema.json', import.meta.url)), 'urn:yourtj:campus-location-extraction:v2')
ajv.addSchema(await readJSON(new URL('runtime.schema.json', import.meta.url)))
const checkCatalog = ajv.compile({ $ref: 'urn:yourtj:campus-location-runtime:v1#/$defs/catalog' })
const checkOverrides = ajv.compile({ $ref: 'urn:yourtj:campus-location-runtime:v1#/$defs/overrides' })

export function validateRuntimeConfig(catalog, overrides) {
  if (!checkCatalog(catalog)) throw new Error(ajv.errorsText(checkCatalog.errors))
  if (!checkOverrides(overrides)) throw new Error(ajv.errorsText(checkOverrides.errors))
  const seen = new Map()
  for (const rule of overrides.rules) {
    validateResult(rule.result, rule.raw)
    if (rule.result.needs_review && rule.action !== 'block') throw new Error('Extraction concerns require an explicit block')
    const key = JSON.stringify([rule.campusId, rule.raw])
    const previous = seen.get(key) ?? []
    if (previous.some(other => (!other.scope && !rule.scope) || (other.scope && rule.scope &&
      (other.scope.calendarId === rule.scope.calendarId || other.scope.termNames.some(term => rule.scope.termNames.includes(term)))))) {
      throw new Error(`Conflicting override scope: ${key}`)
    }
    previous.push(rule); seen.set(key, previous)
  }
  return { places: Object.values(catalog.campuses).reduce((count, entries) => count + entries.length, 0),
    overrides: overrides.rules.length, blocked: overrides.rules.filter(rule => rule.action === 'block').length,
    termScoped: overrides.rules.filter(rule => rule.scope).length }
}

export async function checkRuntimeAssets() {
  const directory = new URL('../../src/site/campus-map/data/', import.meta.url)
  const catalog = await readJSON(new URL('locations/places.json', directory))
  const overrides = await readJSON(new URL('locations/overrides.json', directory))
  const summary = validateRuntimeConfig(catalog, overrides)
  for (const [campus, entries] of Object.entries(catalog.campuses)) {
    const data = await readJSON(new URL(`${campus}.geojson`, directory))
    for (const place of entries) {
      const feature = data.features.find(feature => String(feature.id) === place.featureId)
      if (!feature?.properties.campus || !['academic', 'library', 'place', 'sport'].includes(feature.properties.category))
        throw new Error(`Missing/invalid catalog destination: ${campus} / ${place.name}`)
    }
  }
  return summary
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  checkRuntimeAssets().then(console.log).catch(error => { console.error(error.message); process.exitCode = 1 })
}
