import { expect, it, vi } from 'vitest'
import { fetchDevices } from '../server/devices'
import { cacheKey, loadConfig } from '../server/config'
import { collect } from '../server/collect'
import { serveSnapshot, type SnapshotStore, type Stored } from '../server/snapshots'
import { deviceGraph } from '../src/runtime/device-sankey'

const now = Date.parse('2026-09-28T12:00:00Z')
const config = loadConfig({ STATUS_ENABLED: 'true', UMAMI_URL: 'https://analytics.example.com', UMAMI_SHARE_ID: 'share', UMAMI_USERNAME: 'readonly', UMAMI_PASSWORD: 'private-password' })
const websiteId = 'e643a364-0372-43b2-a341-b05d858866ad'
const rows = [
  { device: 'mobile', os: 'iOS', browser: 'ios', visitors: 5, path: '/private', sessionId: 'private-session', views: '20' },
  { device: 'laptop', os: 'Windows 10', browser: 'edge-chromium', visitors: 3 },
  { device: 'mobile', os: null, browser: 'private-label', visitors: 2 },
]
function upstream(report: unknown = rows, total: unknown = 10) {
  return vi.fn<typeof fetch>(async (input, init) => {
    const url = new URL(String(input))
    if (url.pathname === '/api/share/share') return Response.json({ websiteId, token: 'private-share' })
    if (url.pathname === '/api/auth/login') return Response.json({ token: 'private-auth', user: { username: 'private-user' } })
    expect(init?.headers).toMatchObject({ Authorization: 'Bearer private-auth' })
    if (url.pathname === '/api/reports/breakdown') return Response.json(report)
    if (url.pathname.endsWith('/stats')) return Response.json({ visitors: total })
    throw new Error('Unexpected upstream call')
  })
}
const fetchReport = (fetcher: typeof fetch) => fetchDevices(config, fetcher, AbortSignal.timeout(8000), now, '7d')

it('reads a joint report and only exposes coarse allowlisted visitor aggregates', async () => {
  const fetcher = upstream(), data = await fetchReport(fetcher)
  expect(data).toMatchObject({ visitors: 10, totalVisitors: 10, complete: true, startAt: '2026-09-21T12:00:00.000Z' })
  expect(data.rows).toEqual([{ device: 'mobile', os: 'ios', browser: 'safari', visitors: 5 }, { device: 'laptop', os: 'windows', browser: 'edge', visitors: 3 }, { device: 'mobile', os: 'unknown', browser: 'other', visitors: 2 }])
  expect(JSON.stringify(data)).not.toMatch(/private|session|token|password|path/)
  const report = fetcher.mock.calls.find(([url]) => String(url).endsWith('/breakdown'))!
  expect(JSON.parse(String(report[1]?.body))).toMatchObject({ filters: {}, parameters: { fields: ['device', 'os', 'browser'], startDate: data.startAt, endDate: data.endAt } })
  expect(fetcher.mock.calls.every(([, init]) => init?.redirect === 'error')).toBe(true)
})

it('coalesces aliases and preserves unknown types without dropping visitors', async () => {
  const data = await fetchReport(upstream([...rows, { device: 'mobile', os: 'iOS', browser: 'safari', visitors: 1 }], { value: 11 }))
  expect(data.rows[0].visitors).toBe(6)
  expect(data.rows).toHaveLength(3)
})

it('reports empty and incomplete windows without inventing missing visitors', async () => {
  expect(await fetchReport(upstream([], 0))).toMatchObject({ visitors: 0, totalVisitors: 0, complete: true, rows: [] })
  expect(await fetchReport(upstream(rows, 12))).toMatchObject({ visitors: 10, totalVisitors: 12, complete: false })
  const capped = Array.from({ length: 500 }, () => ({ ...rows[0], visitors: 1 }))
  expect(await fetchReport(upstream(capped, 500))).toMatchObject({ visitors: 500, complete: false })
})

it.each([-1, 1.5, '5', null])('rejects invalid visitor count %s', async visitors => {
  await expect(fetchReport(upstream([{ ...rows[0], visitors }]))).rejects.toThrow()
})
it.each(['__proto__', 'constructor', 'toString'])('treats provider label %s as an unknown family, never an inherited property', async value => {
  const data = await fetchReport(upstream([{ device: value, os: value, browser: value, visitors: 10 }]))
  expect(data.rows).toEqual([{ device: 'other', os: 'other', browser: 'other', visitors: 10 }])
})
it('rejects inconsistent counts and authentication failures', async () => {
  await expect(fetchReport(upstream(rows, 9))).rejects.toThrow('Inconsistent')
  await expect(fetchReport(async () => Response.json({}, { status: 401 }))).rejects.toThrow()
  await expect(fetchReport(async input => Response.json(String(input).endsWith('/login') ? { token: 'bad\r\nheader' } : { websiteId }))).rejects.toThrow('access unavailable')
})

it('conserves all visitors through both chart stages, including folded categories', async () => {
  const data = await fetchReport(upstream())
  for (const dimensions of [['device', 'os', 'browser'], ['device', 'os'], ['os', 'browser']] as const) {
    const graph = deviceGraph(data, 800, [...dimensions])
    for (let depth = 0; depth < dimensions.length; depth++) expect(graph.nodes.filter(n => n.depth === depth).reduce((n, d) => n + d.value!, 0)).toBe(10)
    for (const node of graph.nodes.filter(n => n.depth === 1 && n.sourceLinks?.length)) {
      expect(node.sourceLinks!.reduce((n, l) => n + l.value, 0)).toBe(node.targetLinks!.reduce((n, l) => n + l.value, 0))
    }
    expect(graph.nodes.every(n => Number.isFinite(n.y0) && Number.isFinite(n.y1))).toBe(true)
  }
})

it('isolates device freshness, credentials, and range from other snapshots', async () => {
  const values = new Map<string, Stored<unknown>>()
  const store: SnapshotStore = { read: async <T>(key: string) => values.has(key) ? { value: structuredClone(values.get(key)) as Stored<T>, etag: '1' } : null, write: async (key, value) => { values.set(key, structuredClone(value)); return true } }
  const key = cacheKey(config, 'umami', 'devices-7d')
  values.set(key, { attemptedAt: now, fetchedAt: new Date(now).toISOString(), failed: false, data: await fetchReport(upstream()) })
  const request = (query = '') => new Request(`https://status.example.com/api/status${query}`)
  const read = async (cfg = config, time = now, query = '') => (await (await serveSnapshot(request(query), store, cfg, time)).json()).result
  expect((await read()).devices.state).toBe('ok')
  expect((await read(config, now + 660000)).devices.state).toBe('stale')
  expect((await read(config, now + 900001)).devices.data).toBeNull()
  expect((await read(config, now, '?deviceRange=30d')).devices.state).toBe('unavailable')
  const revoked = { ...config, umami: { ...config.umami, username: undefined, password: undefined } }
  expect((await read(revoked)).devices.state).toBe('unconfigured')
  expect(cacheKey(revoked, 'umami', '7d')).toBe(cacheKey(config, 'umami', '7d'))
  expect((await read({ ...config, umami: { ...config.umami, password: 'changed' } })).devices.state).toBe('unavailable')
  const offline = vi.fn(async () => { throw new Error('private upstream failure') })
  await collect('history', store, config, offline, () => now + 1000)
  const stale = (await read(config, now + 1000)).devices
  expect(stale).toMatchObject({ state: 'stale', fetchedAt: new Date(now).toISOString() })
  expect(stale.data).not.toBeNull()
  expect(JSON.stringify([...values])).not.toContain('private')
  for (const query of ['?deviceRange=1h', '?deviceRange=7d&deviceRange=30d']) expect((await serveSnapshot(request(query), store, config, now)).status).toBe(400)
})


it('recognizes only the explicit native client marker and retains small App flows', async () => {
  const data = await fetchReport(upstream([
    { device: 'mobile', os: 'Android OS', browser: 'yourtj-app', visitors: 1 },
    { device: 'mobile', os: 'Android OS', browser: 'chromium-webview', visitors: 99 },
    { device: 'mobile', os: 'iOS', browser: 'safari', visitors: 900 },
  ], 1000))
  expect(data.rows.find(row => row.browser === 'yourtj-app')).toMatchObject({ visitors: 1, os: 'android' })
  expect(data.rows.find(row => row.browser === 'webview')?.visitors).toBe(99)
  const graph = deviceGraph(data, 800, ['device', 'os', 'browser'])
  expect(graph.nodes.find(node => node.id === 'browser:yourtj-app')?.value).toBe(1)
})
