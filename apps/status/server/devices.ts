import type { StatusDevices, StatusRange } from '../src/types'
import { devicesConfigured, origin, type Config } from './config'
import { count, json, list, object, uuid, type Fetcher } from './http'
import { trafficHours } from './umami'

type DeviceRow = StatusDevices['rows'][number]
// Only these coarse categories cross the public boundary. Never forward arbitrary provider labels.
const devices = new Set(['mobile', 'laptop', 'desktop', 'tablet'])
const systems = new Map<string, DeviceRow['os']>([['Mac OS', 'macos'], ['iOS', 'ios'], ['Android OS', 'android'], ['Linux', 'linux'], ['Chrome OS', 'chromeos']])
const browsers = new Map<string, DeviceRow['browser']>([
  ['yourtj-app', 'yourtj-app'], ['chrome', 'chrome'], ['crios', 'chrome'], ['edge-chromium', 'edge'], ['edge', 'edge'],
  ['safari', 'safari'], ['ios', 'safari'], ['firefox', 'firefox'], ['fxios', 'firefox'],
  ['chromium-webview', 'webview'], ['ios-webview', 'webview'], ['opera', 'opera'], ['opera-mini', 'opera'], ['samsung', 'samsung'],
])
function device(value: unknown): DeviceRow['device'] {
  return value == null || value === '' ? 'unknown' : typeof value === 'string' && devices.has(value) ? value as DeviceRow['device'] : 'other'
}
function os(value: unknown): DeviceRow['os'] {
  if (value == null || value === '') return 'unknown'
  if (typeof value !== 'string') return 'other'
  if (/^Windows(?: |$)/i.test(value)) return 'windows'
  return systems.get(value) ?? 'other'
}
function browser(value: unknown): DeviceRow['browser'] {
  if (value == null || value === '') return 'unknown'
  return typeof value === 'string' ? browsers.get(value) ?? 'other' : 'other'
}

export async function deviceAccess(config: Config, fetcher: Fetcher, signal: AbortSignal) {
  if (!devicesConfigured(config) || !config.umami.username || !config.umami.password) throw new Error('Device reports not configured')
  const base = origin(config, 'umami')
  const [share, auth] = await Promise.all([
    json(fetcher, `${base}/api/share/${config.umami.id}`, signal).then(object),
    json(fetcher, `${base}/api/auth/login`, signal, { method: 'POST', body: JSON.stringify({ username: config.umami.username, password: config.umami.password }) }).then(object),
  ])
  if (!uuid(share.websiteId) || typeof auth.token !== 'string' || auth.token.length > 16384 || !/^[!-~]+$/.test(auth.token)) throw new Error('Device report access unavailable')
  return { base, websiteId: share.websiteId, headers: { Authorization: `Bearer ${auth.token}` } }
}

export async function fetchDevices(config: Config, fetcher: Fetcher, signal: AbortSignal, now: number, range: StatusRange, access = deviceAccess(config, fetcher, signal)): Promise<StatusDevices> {
  const { base, websiteId, headers } = await access
  const start = now - trafficHours[range] * 3_600_000
  const startAt = new Date(start).toISOString(), endAt = new Date(now).toISOString()
  const query = new URLSearchParams({ startAt: String(start), endAt: String(now) })
  // This instance uses Umami's report API. A single joint query preserves relationships;
  // independent metrics must never be joined into invented flows.
  const [report, stats] = await Promise.all([
    json(fetcher, `${base}/api/reports/breakdown`, signal, { method: 'POST', headers, body: JSON.stringify({ websiteId, type: 'breakdown', filters: {}, parameters: { startDate: startAt, endDate: endAt, fields: ['device', 'os', 'browser'] } }) }),
    json(fetcher, `${base}/api/websites/${websiteId}/stats?${query}`, signal, { headers }),
  ])
  const raw = list(report)
  if (raw.length > 500) throw new Error('Unexpected device report size')
  const total = object(stats).visitors
  const totalVisitors = count(typeof total === 'object' && total !== null ? object(total).value : total)
  const groups = new Map<string, DeviceRow>()
  let visitors = 0
  for (const item of raw) {
    const row = object(item), n = count(row.visitors)
    if (!n) continue
    const next: DeviceRow = { device: device(row.device), os: os(row.os), browser: browser(row.browser), visitors: n }
    const key = `${next.device}/${next.os}/${next.browser}`
    next.visitors += groups.get(key)?.visitors ?? 0
    groups.set(key, next)
    visitors += n
    if (!Number.isSafeInteger(visitors) || visitors > totalVisitors) throw new Error('Inconsistent device visitor totals')
  }
  return { startAt, endAt, visitors, totalVisitors, complete: raw.length < 500 && visitors === totalVisitors, rows: [...groups.values()].sort((a, b) => b.visitors - a.visitors) }
}
