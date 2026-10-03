import type { StatusRange, StatusServerRange } from '../src/types'
import { cacheKey, configured, devicesConfigured, type Config, type Provider } from './config'
import { deviceAccess, fetchDevices } from './devices'
import { fetchCurrent, fetchHistory, serverHours } from './komari'
import { fetchTraffic, trafficHours } from './umami'
import { fetchUptime } from './uptime'
import { collectOne, type SnapshotStore } from './snapshots'
import type { Fetcher } from './http'
import { HISTORY_POLICY, DEVICE_POLICY } from '../src/runtime/status-policy'
export async function collect(kind: 'current' | 'history' | 'devices', store: SnapshotStore, config: Config, fetcher: Fetcher = fetch, now: () => number = Date.now) {
  const jobs: Promise<void>[] = []
  const timeout = () => AbortSignal.timeout(8_000)
  const save = async (provider: Provider, part: string, fetchValue: () => Promise<unknown>, retainFor?: number) => collectOne(store, await cacheKey(config, provider, part), fetchValue, now, retainFor)
  if (kind === 'current') {
    if (configured(config, 'uptime')) jobs.push(save('uptime', 'current', () => fetchUptime(config, fetcher, timeout(), now())))
    if (configured(config, 'komari')) jobs.push(save('komari', 'current', () => fetchCurrent(config, fetcher, timeout(), now())))
  } else if (kind === 'history') {
    if (configured(config, 'komari')) for (const range of Object.keys(serverHours) as StatusServerRange[]) jobs.push(save('komari', `history-${range}`, () => fetchHistory(config, fetcher, timeout(), now(), range), HISTORY_POLICY.retainFor))
    if (configured(config, 'umami')) for (const range of Object.keys(trafficHours) as StatusRange[]) jobs.push(save('umami', range, () => fetchTraffic(config, fetcher, timeout(), now(), range), HISTORY_POLICY.retainFor))
  } else if (devicesConfigured(config)) {
    // Share one short-lived login within this collection only; never persist the token.
    // Joint 30-day reports include login and reconciliation and need a longer bounded budget.
    const ranges = Object.keys(trafficHours) as StatusRange[]
    const keys = await Promise.all(ranges.map(range => cacheKey(config, 'umami', `devices-${range}`)))
    const signal = AbortSignal.timeout(15_000), access = deviceAccess(config, fetcher, signal)
    for (const [index, range] of ranges.entries()) jobs.push(collectOne(store, keys[index]!, () => fetchDevices(config, fetcher, signal, now(), range, access), now, DEVICE_POLICY.retainFor))
  }
  // One failed store write must not cancel successful writes from other sources.
  const results = await Promise.allSettled(jobs)
  if (results.some(r => r.status === 'rejected')) throw new Error('Some snapshots could not be saved')
}
