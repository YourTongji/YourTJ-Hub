import { expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { loadConfig } from '../server/config'
import type { SnapshotStore } from '../server/snapshots'
import worker, { cachedSnapshot, type Env } from '../worker/index'

function harness() {
  const bodies = new Map<string, Response>()
  const cache = {
    match: vi.fn(async (request: RequestInfo | URL) => bodies.get((request as Request).url)?.clone()),
    put: vi.fn(async (request: RequestInfo | URL, response: Response) => { bodies.set((request as Request).url, response.clone()) }),
  }
  const pending: Promise<unknown>[] = []
  const ctx = { waitUntil: (p: Promise<unknown>) => { pending.push(p) } }
  const store = { read: vi.fn(async () => null), write: vi.fn(async () => false) } satisfies SnapshotStore
  const config = loadConfig({ STATUS_ENABLED: 'true', UPTIME_URL: 'https://uptime.example.com', UPTIME_SLUG: 'a' })
  async function request(query = '', method = 'GET', current = config, version = 'v1') {
    const result = await cachedSnapshot(new Request(`https://status.example.com/api/status${query}`, { method }), store, current, version, cache, ctx)
    await Promise.all(pending.splice(0))
    return result
  }
  return { store, cache, config, request }
}

it('caches canonical scopes for 30 seconds while keeping browser responses noncacheable', async () => {
  const h = harness()
  const first = await h.request()
  const body = await first.json()
  expect(first.headers.get('Cache-Control')).toBe('no-store')
  expect(h.cache.put.mock.calls[0][1].headers.get('Cache-Control')).toBe('public, max-age=30')
  expect(await (await h.request('?deviceRange=7d&serverRange=1h&range=24h')).json()).toEqual(body)
  expect(h.store.read).toHaveBeenCalledTimes(1)
  const head = await h.request('', 'HEAD')
  expect(await head.text()).toBe('')
  expect(head.status).toBe(200)
  expect(h.store.read).toHaveBeenCalledTimes(1)
  for (const query of ['?range=7d', '?serverRange=6h', '?deviceRange=30d']) await h.request(query)
  expect(h.store.read).toHaveBeenCalledTimes(4)
})

it('invalidates edge entries after a deploy, configuration change, credential revocation or disable', async () => {
  const h = harness()
  await h.request()
  await h.request('', 'GET', h.config, 'v2')
  await h.request('', 'GET', { ...h.config, uptime: { ...h.config.uptime, id: 'other' } })
  await h.request('', 'GET', { ...h.config, umami: { ...h.config.umami, deviceRevision: 'changed' } })
  expect(h.store.read).toHaveBeenCalledTimes(4)
  const disabled = await h.request('', 'GET', { ...h.config, enabled: false })
  expect((await disabled.json()).result.uptime.state).toBe('unconfigured')
  expect(h.store.read).toHaveBeenCalledTimes(4)
  expect(h.cache.put).toHaveBeenCalledTimes(5)
  expect(JSON.stringify(h.cache.put.mock.calls.map(([r]) => (r as Request).url))).not.toContain('changed')
})

it.each(['?range=invalid', '?range=7d&range=24h', '?url=https://private.example.com'])('rejects %s before any cache or storage access', async query => {
  const h = harness()
  expect((await h.request(query)).status).toBe(400)
  expect(h.cache.match).not.toHaveBeenCalled()
  expect(h.cache.put).not.toHaveBeenCalled()
  expect(h.store.read).not.toHaveBeenCalled()
})

it('does not cache failed storage reads or allow public collection', async () => {
  const h = harness()
  h.store.read.mockRejectedValue(new Error('private bucket error'))
  const response = await h.request()
  expect(response.status).toBe(503)
  expect(await response.text()).not.toContain('private')
  expect(h.cache.put).not.toHaveBeenCalled()
  expect((await h.request('', 'POST')).status).toBe(405)
  const badRoute = await worker.fetch(new Request('https://status.example.com/api/collect-current'), {} as Env, { waitUntil() {} })
  expect(badRoute.status).toBe(404)
})

it('falls back to durable storage when edge caching fails', async () => {
  const h = harness()
  h.cache.match.mockRejectedValue(new Error('cache unavailable'))
  h.cache.put.mockRejectedValue(new Error('cache unavailable'))
  expect((await h.request()).status).toBe(200)
  expect(h.store.read).toHaveBeenCalledTimes(1)
})

it('keeps the Worker read-only and preview storage isolated', () => {
  const config = JSON.parse(readFileSync('wrangler.jsonc', 'utf8'))
  const production = config.env.production
  expect(config.triggers.crons).toEqual([])
  expect(production.triggers.crons).toEqual([])
  expect('scheduled' in worker).toBe(false)
  expect(config.vars.STATUS_ENABLED).toBe('false')
  expect(config.r2_buckets[0].bucket_name).not.toBe(production.r2_buckets[0].bucket_name)
  expect(production.routes).toEqual([{ pattern: 'status.yourtj.de', custom_domain: true }])
  expect(config.assets.run_worker_first).toEqual(['/api/*'])
})
