import type { components } from './generated/openapi'
export type StatusSnapshot = components['schemas']['StatusSnapshot']
export type StatusRange = StatusSnapshot['range']
export type StatusServerRange = StatusSnapshot['serverRange']
export type StatusServer = NonNullable<StatusSnapshot['server']['data']>
export type StatusTraffic = NonNullable<StatusSnapshot['traffic']['data']>
export type StatusUptime = NonNullable<StatusSnapshot['uptime']['data']>
export type StatusUptimeMonitor = StatusUptime['monitors'][number]
export type Source<T> = { state: 'ok' | 'stale' | 'unavailable' | 'unconfigured'; fetchedAt?: string; data: T | null }
