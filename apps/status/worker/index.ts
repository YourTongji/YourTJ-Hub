import { createHash } from 'node:crypto'
import type { ExecutionContext, R2Bucket } from '@cloudflare/workers-types'
import { loadConfig, type Config } from '../server/config'
import { r2Store } from '../server/r2'
import { serveSnapshot, snapshotQuery, type SnapshotStore } from '../server/snapshots'

export interface Env {
  SNAPSHOTS: R2Bucket
  VERSION: { id: string }
  STATUS_ENABLED?: string
  UPTIME_URL?: string
  UPTIME_SLUG?: string
  KOMARI_URL?: string
  KOMARI_NODE_ID?: string
  UMAMI_URL?: string
  UMAMI_SHARE_ID?: string
  UMAMI_DEVICE_REVISION?: string
}
type Context = Pick<ExecutionContext, 'waitUntil'>
type SnapshotCache = Pick<Cache, 'match' | 'put'>

export async function cachedSnapshot(request: Request, store: SnapshotStore, config: Config, version: string, cache: SnapshotCache, ctx: Context): Promise<Response> {
  const query = snapshotQuery(request)
  // Validate before looking in the cache: extra or repeated parameters must still fail.
  if (!query || !['GET', 'HEAD'].includes(request.method)) return serveSnapshot(request, store, config)
  const key = new URL(request.url)
  key.search = new URLSearchParams(query).toString()
  // A deployment/configuration change must not expose cached data from revoked sources.
  key.searchParams.set('revision', createHash('sha256').update(JSON.stringify([version, config])).digest('hex'))
  const cacheRequest = new Request(key, { method: 'GET' })
  let response: Response | undefined
  try { response = await cache.match(cacheRequest) } catch { /* Cache failure falls back to durable snapshots. */ }
  if (!response) {
    response = await serveSnapshot(new Request(request.url), store, config)
    if (response.ok) {
      const cached = response.clone()
      cached.headers.set('Cache-Control', 'public, max-age=30')
      ctx.waitUntil(cache.put(cacheRequest, cached).catch(() => {}))
    }
  }
  const headers = new Headers(response.headers)
  headers.set('Cache-Control', 'no-store')
  return new Response(request.method === 'HEAD' ? null : response.body, { status: response.status, headers })
}

export default {
  async fetch(request: Request, env: Env, ctx: Context) {
    const path = new URL(request.url).pathname
    // Static assets are served before this handler. No HTTP route can start collection.
    if (path !== '/api/status') return new Response('Not found', { status: 404, headers: { 'Cache-Control': 'no-store' } })
    try {
      const cache = (caches as CacheStorage & { default: Cache }).default
      const response = await cachedSnapshot(request, r2Store(env.SNAPSHOTS), loadConfig({ ...env, SNAPSHOTS: undefined, VERSION: undefined }), env.VERSION.id, cache, ctx)
      return request.method === 'HEAD' ? new Response(null, response) : response
    } catch {
      return new Response(request.method === 'HEAD' ? null : JSON.stringify({ error: 'Snapshot storage unavailable' }), {
        status: 503, headers: { 'Cache-Control': 'no-store', 'Content-Type': 'application/json', 'X-Content-Type-Options': 'nosniff' },
      })
    }
  },
}
