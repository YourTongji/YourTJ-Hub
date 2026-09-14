import type { StatusUptimeMonitor } from '@gooseforum/client'
import { isRecentStatusTime } from './status-time'

export function uptimeMonitorState(monitor: StatusUptimeMonitor, fresh: boolean, now: number) {
  if (!fresh || !isRecentStatusTime(monitor.current?.time, now, 300_000)) return 'unknown'
  return monitor.current?.status ?? 'unknown'
}
