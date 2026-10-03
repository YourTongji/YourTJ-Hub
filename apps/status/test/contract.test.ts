import { expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { parse } from 'yaml'
import Ajv2020 from 'ajv/dist/2020.js'
import addFormats from 'ajv-formats'
import { serveSnapshot, type SnapshotStore, type Stored } from '../server/snapshots'
import { cacheKey, loadConfig } from '../server/config'
const components = JSON.parse(JSON.stringify(parse(readFileSync('api/components.yaml','utf8'))).replaceAll('"#/','"#/$defs/'))
const ajv = new Ajv2020({ strict:false,allErrors:true }); addFormats(ajv)
const validate = ajv.compile({ $defs:components,$ref:'#/$defs/StatusResponse' })
const openapi = parse(readFileSync('api/openapi.yaml','utf8'))
const mobileSchemas = JSON.parse(JSON.stringify(openapi.components.schemas)
  .replaceAll('"#/components/schemas/', '"#/$defs/'))
const validateMobileCatalog = ajv.compile({ $defs:mobileSchemas,$ref:'#/$defs/MobileReleaseCatalog' })
it('validates the public fixture and rejects extra private fields', () => {
  const fixture = JSON.parse(readFileSync('test/fixtures/status-connected.json','utf8'))
  expect(validate(fixture),JSON.stringify(validate.errors)).toBe(true)
  fixture.result.server.data.ip='private'
  expect(validate(fixture)).toBe(false)
})
it('validates the shared receipt-backed mobile catalog fixture', () => {
  const fixture = JSON.parse(readFileSync('../mobile/packages/forum_app/test/fixtures/mobile_release_catalog.json','utf8'))
  expect(validateMobileCatalog(fixture), JSON.stringify(validateMobileCatalog.errors)).toBe(true)
})
it('validates actual unconfigured and unavailable handler responses for every scope', async () => {
  const store: SnapshotStore = {read:async()=>null,write:async()=>false}
  for (const enabled of [false,true]) for (const range of ['24h','7d','30d']) for (const serverRange of ['1h','6h','24h','7d']) {
    const config=loadConfig({STATUS_ENABLED:String(enabled),UPTIME_URL:'https://uptime.example.com',UPTIME_SLUG:'a'})
    const response=await serveSnapshot(new Request(`https://status.example.com/api/status?range=${range}&serverRange=${serverRange}`),store,config)
    expect(response.status).toBe(200)
    expect(validate(await response.json()),JSON.stringify(validate.errors)).toBe(true)
  }
})
it('validates the history-only fixture against an actual handler response', async () => {
  const fixture = JSON.parse(readFileSync('test/fixtures/status-history-only.json', 'utf8'))
  const config = loadConfig({ STATUS_ENABLED: 'true', KOMARI_URL: 'https://probe.example.com', KOMARI_NODE_ID: 'e643a364-0372-43b2-a341-b05d858866ad' })
  const { history, historyAvailable, historyFetchedAt } = fixture.result.server.data
  const store: SnapshotStore = {
    async read<T>(key: string) {
      return key === await cacheKey(config, 'komari', 'history-1h')
        ? { value: { attemptedAt: Date.parse(historyFetchedAt), fetchedAt: historyFetchedAt, failed: false, data: { history, historyAvailable } } as Stored<T>, etag: '1' } : null
    },
    async write() { return false },
  }
  const response = await serveSnapshot(new Request('https://status.example.com/api/status'), store, config, Date.parse('2026-09-14T12:21:00Z'))
  const body = await response.json()
  expect(body).toEqual(fixture)
  expect(validate(body), JSON.stringify(validate.errors)).toBe(true)
})
