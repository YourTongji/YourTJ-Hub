import type { R2Bucket } from '@cloudflare/workers-types'
import { loadConfig } from '../server/config'
import { r2Store } from '../server/r2'
import { serveSnapshot, snapshotQuery } from '../server/snapshots'
import { viewStore } from '../server/views'

export interface Env {
  SNAPSHOTS: R2Bucket
  STATUS_ENABLED?: string
  UPTIME_URL?: string
  UPTIME_SLUG?: string
  KOMARI_URL?: string
  KOMARI_NODE_ID?: string
  UMAMI_URL?: string
  UMAMI_SHARE_ID?: string
  UMAMI_DEVICE_REVISION?: string
}

export default {
  async fetch(request: Request, env: Env) {
    // Static assets bypass the handler. No HTTP route can start collection.
    if (new URL(request.url).pathname !== '/api/status') return new Response(request.method === 'HEAD' ? null : 'Not found', { status: 404, headers: { 'Cache-Control': 'no-store' } })
    const query = snapshotQuery(request), durable = r2Store(env.SNAPSHOTS)
    // Two bounded reads fit the R2 allowance. Avoid copying dynamic responses
    // into the edge cache: its write CPU spikes exceed the Workers Free budget.
    const response = await serveSnapshot(request, query ? viewStore(durable, query.range, query.serverRange) : durable, loadConfig({ ...env, SNAPSHOTS: undefined }))
    return request.method === 'HEAD' ? new Response(null, response) : response
  },
}
