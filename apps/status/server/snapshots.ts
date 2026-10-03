import type { Source, StatusSnapshot, StatusRange, StatusServerRange, StatusServer, StatusTraffic, StatusUptime, StatusDevices } from '../src/types'
import { cacheKey, configured, devicesConfigured, type Config, type Provider } from './config'
import type { ServerHistory } from './komari'
import { CURRENT_POLICY, HISTORY_POLICY, DEVICE_POLICY, POLL_SECONDS } from '../src/runtime/status-policy'
export type Stored<T> = { attemptedAt: number; fetchedAt?: string; failed: boolean; data: T | null }
export interface SnapshotStore {
  read<T>(key: string): Promise<{ value: Stored<T>; etag: string } | null>
  write<T>(key: string, value: Stored<T>, etag?: string): Promise<boolean>
}
export function source<T>(value: Stored<T> | undefined, now: number, freshFor: number, retainFor: number = CURRENT_POLICY.retainFor): Source<T> {
  const age = now - Date.parse(value?.fetchedAt ?? '')
  if ((!value || value.data === null) || !Number.isFinite(age) || age < -60_000 || age > retainFor) return { state: 'unavailable', data: null }
  return { state: value.failed || age > freshFor ? 'stale' : 'ok', fetchedAt: value.fetchedAt, data: value.data }
}
export async function collectOne<T>(store: SnapshotStore, key: string, fetchValue: () => Promise<T>, now: () => number = Date.now, retainFor: number = CURRENT_POLICY.retainFor) {
  const started = now()
  let data: T | null = null, fetchedAt: string | undefined, failed = false
  try { data = await fetchValue(); fetchedAt = new Date(now()).toISOString() } catch { failed = true }
  // Conditional writes prevent a slower, older run overwriting a newer sample.
  for (let retry = 0; retry < 3; retry++) {
    const old = await store.read<T>(key)
    if (old && old.value.attemptedAt >= started) return
    const previous = source(old?.value, now(), retainFor, retainFor)
    const value: Stored<T> = { attemptedAt: started, failed, data: failed ? previous.data : data, fetchedAt: failed ? previous.fetchedAt : fetchedAt }
    if (await store.write(key, value, old?.etag)) return
  }
  throw new Error('Snapshot changed concurrently')
}
export async function readSnapshot(store: SnapshotStore, config: Config, range: StatusRange, serverRange: StatusServerRange, now: number, deviceRange: StatusRange = '7d'): Promise<StatusSnapshot> {
  async function read<T>(provider: Provider, part: string, policy: { freshFor: number; retainFor: number }): Promise<Source<T>> {
    if (!configured(config, provider)) return { state: 'unconfigured', data: null }
    return source((await store.read<T>(await cacheKey(config, provider, part)))?.value, now, policy.freshFor, policy.retainFor)
  }
  const [server, history, traffic, uptime, devices] = await Promise.all([
    read<StatusServer>('komari', 'current', CURRENT_POLICY), read<ServerHistory>('komari', `history-${serverRange}`, HISTORY_POLICY),
    read<StatusTraffic>('umami', range, HISTORY_POLICY), read<StatusUptime>('uptime', 'current', CURRENT_POLICY),
    devicesConfigured(config) ? read<StatusDevices>('umami', `devices-${deviceRange}`, DEVICE_POLICY) : Promise.resolve<Source<StatusDevices>>({ state: 'unconfigured', data: null }),
  ])
  // Keep the history envelope even when current readings are absent or expired.
  // The source state/timestamp still describe current readings, never history.
  if (server.data || history.data) server.data = {
    ...(server.data ?? { name: '', region: '', cpuCores: null, current: null }),
    history: history.data?.history ?? [], historyAvailable: history.data?.historyAvailable ?? false,
    historyFetchedAt: history.fetchedAt, historyStale: history.state === 'stale',
  }
  return { range, serverRange, deviceRange, refreshAfter: POLL_SECONDS, server, traffic, uptime, devices }
}
export function snapshotQuery(request: Request) {
  const query = new URL(request.url).searchParams
  const range = query.get('range') ?? '24h', serverRange = query.get('serverRange') ?? '1h', deviceRange = query.get('deviceRange') ?? '7d'
  if ([...query.keys()].some(k => !['range','serverRange','deviceRange'].includes(k) || query.getAll(k).length !== 1) || !['24h','7d','30d'].includes(range) || !['1h','6h','24h','7d'].includes(serverRange) || !['24h','7d','30d'].includes(deviceRange)) return null
  return { range: range as StatusRange, serverRange: serverRange as StatusServerRange, deviceRange: deviceRange as StatusRange }
}
export async function serveSnapshot(request: Request, store: SnapshotStore, config: Config, now = Date.now()): Promise<Response> {
  const headers = { 'Cache-Control': 'no-store', 'Content-Type': 'application/json', 'X-Content-Type-Options': 'nosniff' }
  if (!['GET','HEAD'].includes(request.method)) return new Response(null, { status: 405, headers: { ...headers, Allow: 'GET, HEAD' } })
  const query = snapshotQuery(request)
  if (!query) return Response.json({ error: 'Invalid range' }, { status: 400, headers })
  try {
    const result = await readSnapshot(store, config, query.range, query.serverRange, now, query.deviceRange)
    return new Response(request.method === 'HEAD' ? null : JSON.stringify({ code: 0, result }), { headers })
  } catch { return Response.json({ error: 'Snapshot storage unavailable' }, { status: 503, headers }) }
}
