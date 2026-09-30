import type { StatusRange, StatusServerRange } from '../src/types'
import { cacheKey, configured, devicesConfigured, type Config } from './config'
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
  if (kind === 'current') {
    if (configured(config, 'uptime')) jobs.push(collectOne(store, cacheKey(config, 'uptime', 'current'), () => fetchUptime(config, fetcher, timeout(), now()), now))
    if (configured(config, 'komari')) jobs.push(collectOne(store, cacheKey(config, 'komari', 'current'), () => fetchCurrent(config, fetcher, timeout(), now()), now))
  } else if (kind === 'history') {
    if (configured(config, 'komari')) for (const range of Object.keys(serverHours) as StatusServerRange[]) jobs.push(collectOne(store, cacheKey(config, 'komari', `history-${range}`), () => fetchHistory(config, fetcher, timeout(), now(), range), now, HISTORY_POLICY.retainFor))
    if (configured(config, 'umami')) for (const range of Object.keys(trafficHours) as StatusRange[]) jobs.push(collectOne(store, cacheKey(config, 'umami', range), () => fetchTraffic(config, fetcher, timeout(), now(), range), now, HISTORY_POLICY.retainFor))
  } else if (devicesConfigured(config)) {
    // Share one short-lived login within this collection only; never persist the token.
    // Joint 30-day reports include login and reconciliation and need a longer bounded budget.
    const signal = AbortSignal.timeout(15_000), access = deviceAccess(config, fetcher, signal)
    for (const range of Object.keys(trafficHours) as StatusRange[]) jobs.push(collectOne(store, cacheKey(config, 'umami', `devices-${range}`), () => fetchDevices(config, fetcher, signal, now(), range, access), now, DEVICE_POLICY.retainFor))
  }
  // One failed store write must not cancel successful writes from other sources.
  const results = await Promise.allSettled(jobs)
  if (results.some(r => r.status === 'rejected')) throw new Error('Some snapshots could not be saved')
}
