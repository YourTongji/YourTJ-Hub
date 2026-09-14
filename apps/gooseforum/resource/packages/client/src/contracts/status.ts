import type { components } from '../gen/openapi.js'

export type StatusSnapshot = components['schemas']['StatusSnapshot']
export type StatusRange = StatusSnapshot['range']
export type StatusServerRange = StatusSnapshot['serverRange']
export type StatusServer = NonNullable<StatusSnapshot['server']['data']>
export type StatusTraffic = NonNullable<StatusSnapshot['traffic']['data']>
export type StatusUptimeMonitor = NonNullable<StatusSnapshot['uptime']['data']>['monitors'][number]
