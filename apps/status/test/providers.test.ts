import { expect, it, vi } from 'vitest'
import { fetchCurrent, fetchHistory } from '../server/komari'
import { fetchTraffic } from '../server/umami'
import { fetchUptime } from '../server/uptime'
import { json, type Fetcher } from '../server/http'
import { origin, type Config } from '../server/config'
export const now = Date.parse('2026-09-14T12:00:00Z')
export const id = 'e643a364-0372-43b2-a341-b05d858866ad'
export const config: Config = { enabled: true, komari: { url: 'https://probe.example.com', id }, umami: { url: 'https://analytics.example.com', id: 'publicShare' }, uptime: { url: 'https://uptime.example.com', id: 'a' } }
const signal = () => AbortSignal.timeout(8000)
const response = (value: unknown) => Response.json(value)
function mockSource(overrides: Record<string, unknown> = {}) {
  const data: Record<string, unknown> = {
    'common:getNodes': { uuid: id, name: 'YourTJ', region: 'JP', cpu_cores: 4, mem_total: 1000, hidden: false, ipv4: 'private-host', remark: 'private note' },
    'public:getClientRecentRecords': [{ updated_at: new Date(now).toISOString(), cpu: { usage: 30 }, ram: { total: 1000, used: 500 }, disk: { total: 1000, used: 100 }, network: { up: 0, down: 10 }, uptime: 120 }],
    'common:getRecords': { records: [{ client: id, time: new Date(now - 1000).toISOString(), cpu: 20, ram: 500 }] },
    '/api/share/publicShare': { websiteId: id, token: 'private-token' },
    [`/api/websites/${id}/stats`]: { visitors: { value: 10 }, pageviews: 30, visits: 20, bounces: 5, totaltime: 200 },
    [`/api/websites/${id}/active`]: { visitors: 0 },
    [`/api/websites/${id}/pageviews`]: { pageviews: [{ x: '2026-09-14 11:00:00', y: 30 }], sessions: [{ x: '2026-09-14 11:00:00', y: 10 }] },
    '/api/status-page/a': { config: { published: true, slug: 'a' }, publicGroupList: [{ monitorList: [{ id: 1, name: 'Forum', type: 'http', url: 'secret-url' }] }] },
    '/api/status-page/heartbeat/a': { heartbeatList: { '1': [{ time: '2026-09-14 12:00:00', status: 1, ping: 0, msg: 'private error' }], '2': [{ time: '2026-09-14 12:00:00', status: 0, ping: 99 }] }, uptimeList: { '1_24': 0.9995 } },
    ...overrides,
  }
  const calls: { url: string; init?: RequestInit }[] = []
  const fetcher: Fetcher = async (input, init) => {
    const url = String(input); calls.push({ url, init })
    const rpc = new URL(url).pathname === '/api/rpc2'
    const key = rpc ? JSON.parse(String(init?.body)).method : new URL(url).pathname
    const value = data[key]
    if (value instanceof Error) throw value
    return response(rpc ? { result: value } : value)
  }
  return { fetcher, calls, data }
}
it('projects only public Komari fields, keeps zero values and checks the selected node', async () => {
  const mock = mockSource(), current = await fetchCurrent(config, mock.fetcher, signal(), now)
  expect(current.current?.networkUp).toBe(0)
  expect(JSON.stringify(current)).not.toMatch(/private|ipv4|remark/)
  mock.data['common:getNodes'] = { uuid: id, hidden: true }
  await expect(fetchCurrent(config, mock.fetcher, signal(), now)).rejects.toThrow()
})
it('rejects incomplete and future current samples instead of defaulting to zero', async () => {
  for (const changes of [{ cpu: {} }, { updated_at: new Date(now + 60001).toISOString() }, { ram: { used: 500, total: 0 } }]) {
    const mock = mockSource(); const current = mock.data['public:getClientRecentRecords'] as Record<string, unknown>[]
    mock.data['public:getClientRecentRecords'] = [{ ...current[0], ...changes }]
    await expect(fetchCurrent(config, mock.fetcher, signal(), now)).rejects.toThrow()
  }
})
it.each([false,true])('accepts Komari history array and keyed maps, retaining full span: %s', async keyed => {
  const records = Array.from({ length: 500 }, (_, i) => ({ client: id, time: new Date(now - (499-i)*60000).toISOString(), cpu: 20, ram: 500 }))
  const mock = mockSource({ 'common:getRecords': { records: keyed ? { [id]: records } : records } })
  const result = await fetchHistory(config, mock.fetcher, signal(), now, '24h')
  expect(result.history).toHaveLength(120)
  expect(result.history[0]!.time).toBe(records[0]!.time)
  expect(result.history.at(-1)!.time).toBe(records.at(-1)!.time)
  expect(result.history[0]!.memoryPercent).toBe(50)
  const query = JSON.parse(String(mock.calls.find(c => String(c.init?.body).includes('common:getRecords'))!.init!.body))
  expect(query.params).toMatchObject({ hours: 24, maxCount: 120, uuid: id })
})
it('filters wrong-node and invalid historical readings; rejects missing history', async () => {
  const mock = mockSource({ 'common:getRecords': { records: [{ client: 'other', time: new Date(now).toISOString(), cpu: 50, ram: 200 }, { client: id, time: 'bad', cpu: 20, ram: 10 }] } })
  expect((await fetchHistory(config, mock.fetcher, signal(), now, '1h')).history).toEqual([])
  mock.data['common:getRecords'] = { records: null }
  await expect(fetchHistory(config, mock.fetcher, signal(), now, '1h')).rejects.toThrow()
})
it('reads the public Umami share with context headers, supports v2/v3 counts and UTC buckets', async () => {
  const mock = mockSource(), result = await fetchTraffic(config, mock.fetcher, signal(), now, '7d')
  expect(result).toMatchObject({ visitors: 10, bounceRate: 25, averageDuration: 10, activeVisitors: 0, seriesAvailable: true })
  expect(result.series[0]!.time).toBe('2026-09-14T11:00:00.000Z')
  for (const call of mock.calls.filter(c => c.url.includes('/websites/'))) {
    expect(call.init!.headers).toMatchObject({ 'X-Umami-Share-Token': 'private-token', 'X-Umami-Share-Context': 'overview' })
    expect(call.init!.redirect).toBe('error')
  }
  expect(new URL(mock.calls.find(c => c.url.includes('/stats'))!.url).searchParams.get('unit')).toBe('day')
  expect(JSON.stringify(result)).not.toContain('private-token')
})
it('keeps Umami totals when optional chart/active calls fail; missing totals remain unavailable', async () => {
  const mock = mockSource({ [`/api/websites/${id}/active`]: new Error('offline'), [`/api/websites/${id}/pageviews`]: { pageviews: [], sessions: null } })
  expect(await fetchTraffic(config, mock.fetcher, signal(), now, '24h')).toMatchObject({ visitors: 10, activeVisitors: null, seriesAvailable: false })
  mock.data[`/api/websites/${id}/stats`] = { visitors: null, pageviews: 0, visits: 0 }
  await expect(fetchTraffic(config, mock.fetcher, signal(), now, '24h')).rejects.toThrow()
})
it('accepts long Umami share tokens without publishing them', async () => {
  const token = 'public-share-signature.'.repeat(40)
  const mock = mockSource({ '/api/share/publicShare': { websiteId: id, token } })
  const result = await fetchTraffic(config, mock.fetcher, signal(), now, '24h')
  expect(result.visitors).toBe(10)
  expect(mock.calls[1]!.init!.headers).toMatchObject({ 'X-Umami-Share-Token': token })
  expect(JSON.stringify(result)).not.toContain(token)
})
it.each(['token\r\nheader: value', 'x'.repeat(16385), null])('rejects malformed or oversized share tokens', async token => {
  const mock = mockSource({ '/api/share/publicShare': { websiteId: id, token } })
  await expect(fetchTraffic(config, mock.fetcher, signal(), now, '24h')).rejects.toThrow()
  expect(mock.calls).toHaveLength(1)
})
it('uses only published Uptime monitors and never returns target URLs or raw errors', async () => {
  const mock = mockSource(), result = await fetchUptime(config, mock.fetcher, signal(), now)
  expect(result.monitors).toHaveLength(1)
  expect(result.monitors[0]).toMatchObject({ uptime24h: 99.95, current: { status: 'up', ping: 0 } })
  expect(JSON.stringify(result)).not.toMatch(/secret-url|private error/)
  mock.data['/api/status-page/a'] = { config: { published: false, slug: 'a' }, publicGroupList: [] }
  await expect(fetchUptime(config, mock.fetcher, signal(), now)).rejects.toThrow()
})
it.each([0,1,2,3,4,null])('preserves Uptime status %s', async status => {
  const mock = mockSource({ '/api/status-page/heartbeat/a': { heartbeatList: { '1': [{ time: '2026-09-14 12:00:00', status }] } } })
  expect((await fetchUptime(config, mock.fetcher, signal(), now)).monitors[0]!.current!.status).toBe(({ 0:'down',1:'up',2:'pending',3:'maintenance' } as Record<string,string>)[String(status)] ?? 'unknown')
})
it.each(['invalid', '2026-09-14T12:10:00Z'])('does not promote an old green check after invalid latest time %s', async time => {
  const mock = mockSource({ '/api/status-page/heartbeat/a': { heartbeatList: { '1': [{ time: '2026-09-14 11:59:00', status: 1 }, { time, status: 0 }] } } })
  const monitor = (await fetchUptime(config, mock.fetcher, signal(), now)).monitors[0]!
  expect(monitor.current).toBeNull(); expect(monitor.history).toHaveLength(1)
})
it('limits source response size and rejects redirects/status failures', async () => {
  const big = vi.fn(async () => new Response('x'.repeat(2*1024*1024 + 1))) as Fetcher
  await expect(json(big,'https://example.com',signal())).rejects.toThrow('too large')
  const redirect = vi.fn(async () => new Response(null, { status: 302 })) as Fetcher
  await expect(json(redirect,'https://example.com',signal())).rejects.toThrow()
})
it.each(['http://example.com','https://user:pass@example.com','https://example.com/path','https://example.com/?x=1'])('rejects unsafe configured origin %s', url => {
  expect(() => origin({ ...config, uptime: { url, id:'a' } },'uptime')).toThrow()
})
