import { expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { cacheKey, loadConfig } from '../server/config'
import worker, { type Env } from '../worker/index'

async function harness() {
  const vars = { STATUS_ENABLED: 'true', UPTIME_URL: 'https://uptime.example.com', UPTIME_SLUG: 'a' }
  const key = await cacheKey(loadConfig(vars), 'uptime', 'current')
  const get = vi.fn(async () => ({ json: async () => ({ data: { [key]: { attemptedAt: Date.now(), fetchedAt: new Date().toISOString(), failed: false, data: { monitors: [] } } } }), etag: 'view' }))
  const put = vi.fn()
  const env = { ...vars, SNAPSHOTS: { get, put } } as unknown as Env
  const request = (query = '', method = 'GET', current = env) => worker.fetch(new Request(`https://status.example.com/api/status${query}`, { method }), current)
  return { env, get, put, request }
}

it('reads selected views without dynamic response caching or durable writes', async () => {
  const h = await harness()
  const first = await h.request()
  expect(first.headers.get('Cache-Control')).toBe('no-store')
  expect((await first.json()).result.uptime.state).toBe('ok')
  expect(h.get).toHaveBeenLastCalledWith('views/v1/24h/1h')
  await h.request('?range=7d&serverRange=6h&deviceRange=30d')
  expect(h.get).toHaveBeenLastCalledWith('views/v1/7d/6h')
  await h.request()
  expect(h.get).toHaveBeenCalledTimes(3)
  expect(h.put).not.toHaveBeenCalled()
})

it('applies source changes and revocation on the next request', async () => {
  const h = await harness()
  await h.request()
  const changed = await h.request('', 'GET', { ...h.env, UPTIME_SLUG: 'other' })
  expect((await changed.json()).result.uptime.state).toBe('unavailable')
  const disabled = await h.request('', 'GET', { ...h.env, STATUS_ENABLED: 'false' })
  expect((await disabled.json()).result.uptime.state).toBe('unconfigured')
  expect(h.get).toHaveBeenCalledTimes(2)
})

it.each(['?range=invalid', '?range=7d&range=24h', '?url=https://private.example.com'])('rejects %s before storage access', async query => {
  const h = await harness()
  expect((await h.request(query)).status).toBe(400)
  expect(h.get).not.toHaveBeenCalled()
})

it('returns noncacheable errors, rejects collection and keeps every HEAD response body empty', async () => {
  const h = await harness()
  expect(await (await h.request('', 'HEAD')).text()).toBe('')
  const invalid = await h.request('?range=invalid', 'HEAD')
  expect(invalid.status).toBe(400); expect(await invalid.text()).toBe('')
  const post = await h.request('', 'POST')
  expect(post.status).toBe(405); expect(post.headers.get('Allow')).toBe('GET, HEAD')
  h.get.mockRejectedValue(new Error('private bucket error'))
  const error = await h.request()
  expect(error.status).toBe(503); expect(error.headers.get('Cache-Control')).toBe('no-store')
  expect(await error.text()).not.toContain('private')
  const head = await h.request('', 'HEAD')
  expect(head.status).toBe(503); expect(await head.text()).toBe('')
  expect((await worker.fetch(new Request('https://status.example.com/api/collect-current'), h.env)).status).toBe(404)
  expect(await (await worker.fetch(new Request('https://status.example.com/api/collect-current', { method: 'HEAD' }), h.env)).text()).toBe('')
  expect(h.put).not.toHaveBeenCalled()
})

it('keeps the Worker read-only and preview storage isolated', () => {
  const config = JSON.parse(readFileSync('wrangler.jsonc', 'utf8'))
  const production = config.env.production
  expect(config.triggers.crons).toEqual([])
  expect(production.triggers.crons).toEqual([])
  expect('scheduled' in worker).toBe(false)
  expect(config.compatibility_flags ?? []).not.toContain('nodejs_compat')
  expect(config.vars.STATUS_ENABLED).toBe('false')
  expect(config.r2_buckets[0].bucket_name).not.toBe(production.r2_buckets[0].bucket_name)
  expect(production.routes).toEqual([{ pattern: 'status.yourtj.de', custom_domain: true }])
  expect(config.assets.run_worker_first).toEqual(['/api/*'])
})
