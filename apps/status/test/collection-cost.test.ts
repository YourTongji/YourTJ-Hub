import { readFileSync } from 'node:fs'
import { expect, it, vi } from 'vitest'
import { collect } from '../server/collect'
import { cacheKey, loadConfig } from '../server/config'
import { serveSnapshot, type SnapshotStore, type Stored } from '../server/snapshots'

const now = Date.parse('2026-09-30T12:00:00Z')
const config = loadConfig({ STATUS_ENABLED: 'true', UMAMI_URL: 'https://analytics.example.com', UMAMI_SHARE_ID: 'share', UMAMI_USERNAME: 'readonly', UMAMI_PASSWORD: 'secret' })
const fixture = JSON.parse(readFileSync('test/fixtures/status-connected.json', 'utf8')).result
function memory() {
  const values = new Map<string, Stored<unknown>>()
  const store: SnapshotStore = {
    async read<T>(key: string) { return values.has(key) ? { value: structuredClone(values.get(key)) as Stored<T>, etag: '1' } : null },
    async write(key, value) { values.set(key, structuredClone(value)); return true },
  }
  return { values, store }
}

it('does not run authenticated device reports with the more frequent history collector', async () => {
  const { values, store } = memory()
  const fetcher = vi.fn<typeof fetch>(async () => { throw new Error('offline') })
  await collect('history', store, config, fetcher, () => now)
  expect([...values.keys()].some(key => key.includes('devices-'))).toBe(false)
  expect(fetcher.mock.calls.some(([url]) => String(url).endsWith('/api/auth/login'))).toBe(false)
  expect(values.size).toBe(3)
})

it('keeps lower-frequency analytics usable between scheduled runs and expires them independently', async () => {
  const { store } = memory()
  for (const [part, data] of [['24h', fixture.traffic.data], ['devices-7d', fixture.devices.data]] as const) {
    await store.write(await cacheKey(config, 'umami', part), { attemptedAt: now, fetchedAt: new Date(now).toISOString(), failed: false, data })
  }
  const read = async (minutes: number) => (await (await serveSnapshot(new Request('https://status.example.com/api/status'), store, config, now + minutes * 60_000)).json()).result
  expect((await read(14)).traffic.state).toBe('ok')
  expect((await read(21)).traffic.state).toBe('stale')
  expect((await read(61)).traffic.data).toBeNull()
  expect((await read(65)).devices.state).toBe('ok')
  expect((await read(71)).devices.state).toBe('stale')
  expect((await read(181)).devices.data).toBeNull()
  expect((await read(0)).refreshAfter).toBe(60)
})

it('retains an hourly device snapshot when the next run fails without renewing its timestamp', async () => {
  const { store } = memory()
  await store.write(await cacheKey(config, 'umami', 'devices-7d'), { attemptedAt: now, fetchedAt: new Date(now).toISOString(), failed: false, data: fixture.devices.data })
  const later = now + 65 * 60_000
  await collect('devices', store, config, async () => { throw new Error('offline') }, () => later)
  const response = await serveSnapshot(new Request('https://status.example.com/api/status'), store, config, later)
  expect((await response.json()).result.devices).toMatchObject({ state: 'stale', fetchedAt: new Date(now).toISOString(), data: fixture.devices.data })
})
