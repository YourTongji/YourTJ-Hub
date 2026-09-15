import { expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { parse } from 'yaml'
import Ajv2020 from 'ajv/dist/2020.js'
import addFormats from 'ajv-formats'
import { serveSnapshot, type SnapshotStore } from '../server/snapshots'
import { loadConfig } from '../server/config'
const components = JSON.parse(JSON.stringify(parse(readFileSync('api/components.yaml','utf8'))).replaceAll('"#/','"#/$defs/'))
const ajv = new Ajv2020({ strict:false,allErrors:true }); addFormats(ajv)
const validate = ajv.compile({ $defs:components,$ref:'#/$defs/StatusResponse' })
it('validates the public fixture and rejects extra private fields', () => {
  const fixture = JSON.parse(readFileSync('test/fixtures/status-connected.json','utf8'))
  expect(validate(fixture),JSON.stringify(validate.errors)).toBe(true)
  fixture.result.server.data.ip='private'
  expect(validate(fixture)).toBe(false)
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
