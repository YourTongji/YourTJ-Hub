import { readFileSync } from 'node:fs'
import { S3Client } from '@aws-sdk/client-s3'
import { collect } from '../server/collect'
import { configured, devicesConfigured, loadConfig } from '../server/config'
import { s3Store } from '../server/s3'
import { publishViews } from '../server/views'

const deployment = JSON.parse(readFileSync(new URL('../wrangler.jsonc', import.meta.url), 'utf8'))
const production = deployment.env.production
const kind = process.argv[2]
if (kind !== 'public' && kind !== 'devices') throw new Error('Expected public or devices collection')
const target = process.argv[3]
if (target !== 'production' && target !== 'preview') throw new Error('Explicit production or preview target required')
const bucket = target === 'production' ? production.r2_buckets[0].bucket_name : deployment.r2_buckets[0].bucket_name
const { R2_ACCESS_KEY_ID: accessKeyId, R2_SECRET_ACCESS_KEY: secretAccessKey, UMAMI_USERNAME, UMAMI_PASSWORD } = process.env
if (!accessKeyId || !secretAccessKey) throw new Error('Missing private R2 collector credentials')
const config = loadConfig({ ...production.vars, UMAMI_USERNAME, UMAMI_PASSWORD })
// Revocation must disable the writer too, even if old credentials remain in CI.
if (kind === 'devices' && !config.umami.deviceRevision) throw new Error('Device collection is disabled: no public device revision')
if (kind === 'devices' && devicesConfigured(config) && (!UMAMI_USERNAME || !UMAMI_PASSWORD)) throw new Error('Configure UMAMI_USERNAME and UMAMI_PASSWORD in the status-collector environment')
const client = new S3Client({
  region: 'auto', endpoint: `https://${deployment.account_id}.r2.cloudflarestorage.com`,
  credentials: { accessKeyId, secretAccessKey }, maxAttempts: 3,
})
const durable = s3Store(client, bucket)
let failed = false
const store: typeof durable = {
  read: durable.read,
  async write(key, value, etag) {
    if (value.failed) { failed = true; console.warn('Source collection failed:', key.split('/')[0], key.split('/')[2]) }
    return durable.write(key, value, etag)
  },
}
try {
  if (kind === 'devices') await collect('devices', store, config)
  else {
    if (!['uptime', 'komari', 'umami'].some(p => configured(config, p as 'uptime' | 'komari' | 'umami'))) throw new Error('Public sources are disabled')
    await collect('current', store, config)
    await collect('history', store, config)
    await publishViews(durable, config)
  }
  if (failed) throw new Error('One or more sources failed; retained snapshots keep their original timestamps')
  console.log(`Completed ${kind} snapshot collection (${target})`)
} finally { client.destroy() }
