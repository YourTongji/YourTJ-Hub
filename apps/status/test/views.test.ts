import { expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { cacheKey, loadConfig } from '../server/config'
import { readSnapshot, type SnapshotStore, type Stored } from '../server/snapshots'
import { publishViews, viewStore } from '../server/views'

const fixture = JSON.parse(readFileSync('test/fixtures/status-connected.json', 'utf8')).result
const now = Date.parse('2026-09-14T12:00:00Z')
const config = loadConfig({ STATUS_ENABLED: 'true', UPTIME_URL: 'https://uptime.example.com', UPTIME_SLUG: 'a', KOMARI_URL: 'https://probe.example.com', KOMARI_NODE_ID: 'node', UMAMI_URL: 'https://analytics.example.com', UMAMI_SHARE_ID: 'share', UMAMI_DEVICE_REVISION: 'v1' })
function harness() {
  const records = new Map<string, { value: Stored<unknown>; etag: string }>()
  const read = vi.fn(async (key: string) => structuredClone(records.get(key) ?? null))
  let version = 0
  const write = vi.fn(async (key: string, value: Stored<unknown>, etag?: string) => {
    if (records.get(key)?.etag !== etag) return false
    records.set(key, { value: structuredClone(value), etag: `${++version}` }); return true
  })
  const store: SnapshotStore = { async read<T>(key: string) { return await read(key) as { value: Stored<T>; etag: string } | null }, write }
  const seed = (key: string, data: unknown) => records.set(key, { value: { attemptedAt: now, fetchedAt: new Date(now).toISOString(), failed: false, data }, etag: 'seed' })
  seed(cacheKey(config, 'komari', 'current'), fixture.server.data)
  seed(cacheKey(config, 'uptime', 'current'), fixture.uptime.data)
  for (const range of ['1h', '6h', '24h', '7d']) seed(cacheKey(config, 'komari', `history-${range}`), { history: fixture.server.data.history, historyAvailable: true })
  for (const range of ['24h', '7d', '30d']) {
    seed(cacheKey(config, 'umami', range), fixture.traffic.data)
    seed(cacheKey(config, 'umami', `devices-${range}`), fixture.devices.data)
  }
  return { store, records, read, write }
}

it('serves every range with identical source semantics using only two durable reads', async () => {
  const h = harness()
  await publishViews(h.store, config, () => now + 1000)
  for (const range of ['24h', '7d', '30d'] as const) for (const serverRange of ['1h', '6h', '24h', '7d'] as const) for (const deviceRange of ['24h', '7d', '30d'] as const) {
    const expected = await readSnapshot(h.store, config, range, serverRange, now, deviceRange)
    h.read.mockClear()
    expect(await readSnapshot(viewStore(h.store, range, serverRange), config, range, serverRange, now, deviceRange)).toEqual(expected)
    expect(h.read).toHaveBeenCalledTimes(2)
  }
})

it('preserves failed-source age, independent retention and source revocation inside a view', async () => {
  const h = harness()
  h.records.get(cacheKey(config, 'komari', 'current'))!.value.failed = true
  await publishViews(h.store, config, () => now + 1000)
  for (const age of [0, 21 * 60_000, 61 * 60_000, 181 * 60_000]) {
    expect(await readSnapshot(viewStore(h.store, '24h', '1h'), config, '24h', '1h', now + age)).toEqual(await readSnapshot(h.store, config, '24h', '1h', now + age))
  }
  const changed = { ...config, komari: { ...config.komari, id: 'new-node' } }
  const response = await readSnapshot(viewStore(h.store, '24h', '1h'), changed, '24h', '1h', now)
  expect(response.server).toEqual({ state: 'unavailable', data: null })
  expect(response.traffic.state).toBe('ok')
  const disabled = { ...config, enabled: false }
  await publishViews(h.store, disabled, () => now + 2000)
  expect((await readSnapshot(viewStore(h.store, '24h', '1h'), config, '24h', '1h', now)).server.state).toBe('unavailable')
})

it('handles missing views and surfaces storage failures without exposing raw records', async () => {
  const h = harness()
  expect((await readSnapshot(viewStore(h.store, '24h', '1h'), config, '24h', '1h', now)).server.state).toBe('unavailable')
  await expect(viewStore(h.store, '24h', '1h').write('key', { attemptedAt: now, data: null, failed: true })).rejects.toThrow('read-only')
  h.read.mockRejectedValueOnce(new Error('storage failure'))
  await expect(readSnapshot(viewStore(h.store, '24h', '1h'), config, '24h', '1h', now)).rejects.toThrow('storage failure')
  h.write.mockRejectedValueOnce(new Error('storage failure'))
  await expect(publishViews(h.store, config, () => now)).rejects.toThrow('Public views could not be saved')
  expect(h.write).toHaveBeenCalledTimes(12)
})
