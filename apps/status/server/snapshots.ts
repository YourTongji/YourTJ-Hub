import type { Source, StatusSnapshot, StatusRange, StatusServerRange, StatusServer, StatusTraffic, StatusUptime } from '../src/types'
import { cacheKey, configured, type Config, type Provider } from './config'
import type { ServerHistory } from './komari'
export const RETENTION = 900_000
export const CURRENT_FRESHNESS = 150_000
export const HISTORY_FRESHNESS = 600_000
export type Stored<T> = { attemptedAt: number; fetchedAt?: string; failed: boolean; data: T | null }
export interface SnapshotStore {
  read<T>(key: string): Promise<{ value: Stored<T>; etag: string } | null>
  write<T>(key: string, value: Stored<T>, etag?: string): Promise<boolean>
}
export function source<T>(value: Stored<T> | undefined, now: number, freshFor: number): Source<T> {
  const age = now - Date.parse(value?.fetchedAt ?? '')
  if ((!value || value.data === null) || !Number.isFinite(age) || age < -60_000 || age > RETENTION) return { state: 'unavailable', data: null }
  return { state: value.failed || age > freshFor ? 'stale' : 'ok', fetchedAt: value.fetchedAt, data: value.data }
}
export async function collectOne<T>(store: SnapshotStore, key: string, fetchValue: () => Promise<T>, now: () => number = Date.now) {
  const started = now()
  let data: T | null = null, fetchedAt: string | undefined, failed = false
  try { data = await fetchValue(); fetchedAt = new Date(now()).toISOString() } catch { failed = true }
  // Conditional writes prevent a slower, older run overwriting a newer sample.
  for (let retry = 0; retry < 3; retry++) {
    const old = await store.read<T>(key)
    if (old && old.value.attemptedAt >= started) return
    const previous = source(old?.value, now(), RETENTION)
    const value: Stored<T> = { attemptedAt: started, failed, data: failed ? previous.data : data, fetchedAt: failed ? previous.fetchedAt : fetchedAt }
    if (await store.write(key, value, old?.etag)) return
  }
  throw new Error('Snapshot changed concurrently')
}
export async function readSnapshot(store: SnapshotStore, config: Config, range: StatusRange, serverRange: StatusServerRange, now: number): Promise<StatusSnapshot> {
  async function read<T>(provider: Provider, part: string, freshness: number): Promise<Source<T>> {
    if (!configured(config, provider)) return { state: 'unconfigured', data: null }
    return source((await store.read<T>(cacheKey(config, provider, part)))?.value, now, freshness)
  }
  const [server, history, traffic, uptime] = await Promise.all([
    read<StatusServer>('komari', 'current', CURRENT_FRESHNESS), read<ServerHistory>('komari', `history-${serverRange}`, HISTORY_FRESHNESS),
    read<StatusTraffic>('umami', range, HISTORY_FRESHNESS), read<StatusUptime>('uptime', 'current', CURRENT_FRESHNESS),
  ])
  if (server.data) server.data = { ...server.data, history: history.data?.history ?? [], historyAvailable: history.data?.historyAvailable ?? false, ...(history.fetchedAt ? { historyFetchedAt: history.fetchedAt } : {}), historyStale: history.state === 'stale' }
  return { range, serverRange, refreshAfter: 30, server, traffic, uptime }
}
export async function serveSnapshot(request: Request, store: SnapshotStore, config: Config, now = Date.now()): Promise<Response> {
  const headers = { 'Cache-Control': 'no-store', 'Content-Type': 'application/json', 'X-Content-Type-Options': 'nosniff' }
  if (!['GET','HEAD'].includes(request.method)) return new Response(null, { status: 405, headers: { ...headers, Allow: 'GET, HEAD' } })
  const query = new URL(request.url).searchParams
  const range = query.get('range') ?? '24h', serverRange = query.get('serverRange') ?? '1h'
  if ([...query.keys()].some(k => !['range','serverRange'].includes(k) || query.getAll(k).length !== 1) || !['24h','7d','30d'].includes(range) || !['1h','6h','24h','7d'].includes(serverRange)) return Response.json({ error: 'Invalid range' }, { status: 400, headers })
  try {
    const result = await readSnapshot(store, config, range as StatusRange, serverRange as StatusServerRange, now)
    // Cache variants include only the two permitted query keys. Responses never set cookies.
    return new Response(request.method === 'HEAD' ? null : JSON.stringify({ code: 0, result }), { headers: { ...headers, 'Netlify-CDN-Cache-Control': 'public, durable, max-age=15', 'Netlify-Vary': 'query=range|serverRange' } })
  } catch { return Response.json({ error: 'Snapshot storage unavailable' }, { status: 503, headers }) }
}
