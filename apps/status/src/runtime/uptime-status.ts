import type { StatusUptimeMonitor } from '@/types'
import { isRecentStatusTime } from './status-time'
import { CURRENT_POLICY } from './status-policy'

export function uptimeMonitorState(monitor: StatusUptimeMonitor, fresh: boolean, now: number) {
  if (!fresh || !isRecentStatusTime(monitor.current?.time, now, CURRENT_POLICY.freshFor)) return 'unknown'
  return monitor.current?.status ?? 'unknown'
}
