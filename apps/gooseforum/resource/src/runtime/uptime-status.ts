import type { StatusUptimeMonitor } from '@gooseforum/client'

export function uptimeMonitorState(monitor: StatusUptimeMonitor, fresh: boolean, now: number) {
  const checked = Date.parse(monitor.current?.time ?? '')
  if (!fresh || !Number.isFinite(checked) || now - checked > 300_000 || checked - now > 60_000) return 'unknown'
  return monitor.current?.status ?? 'unknown'
}
