import type { StatusTraffic, StatusRange } from '../src/types'
import type { Config } from './config'
import { origin } from './config'
import { json, object, list, count, uuid, timestamp, type Fetcher } from './http'
const metric = (v: unknown) => count(typeof v === 'object' && v !== null ? object(v).value : v)
function optionalMetric(v: unknown): number | null { try { return metric(v) } catch { return null } }
export const trafficHours = { '24h': 24, '7d': 168, '30d': 720 } as const
export async function fetchTraffic(config: Config, fetcher: Fetcher, signal: AbortSignal, now: number, range: StatusRange): Promise<StatusTraffic> {
  const base = origin(config, 'umami'), share = object(await json(fetcher, `${base}/api/share/${config.umami.id}`, signal))
  if (!uuid(share.websiteId)) throw new Error('Invalid public share')
  // Signed share tokens can exceed the display-name limit; accept bounded header-safe text.
  if (typeof share.token !== 'string' || share.token.length > 16384 || !/^[!-~]+$/.test(share.token)) throw new Error('Invalid share token')
  const headers = { 'X-Umami-Share-Token': share.token, 'X-Umami-Share-Context': 'overview' }
  const start = now - trafficHours[range] * 3_600_000
  const query = new URLSearchParams({ startAt: String(start), endAt: String(now), timezone: 'UTC', unit: range === '24h' ? 'hour' : 'day' })
  const endpoint = `${base}/api/websites/${share.websiteId}`
  const [statsResult, activeResult, seriesResult] = await Promise.allSettled([
    json(fetcher, `${endpoint}/stats?${query}`, signal, { headers }), json(fetcher, `${endpoint}/active`, signal, { headers }), json(fetcher, `${endpoint}/pageviews?${query}`, signal, { headers }),
  ])
  if (statsResult.status !== 'fulfilled') throw new Error('Traffic unavailable')
  const stats = object(statsResult.value), visitors = metric(stats.visitors), pageviews = metric(stats.pageviews), visits = metric(stats.visits)
  const bounces = optionalMetric(stats.bounces), totalTime = optionalMetric(stats.totaltime)
  const result: StatusTraffic = { startAt: new Date(start).toISOString(), endAt: new Date(now).toISOString(), visitors, pageviews, visits, bounceRate: visits && bounces !== null ? Math.min(100,bounces / visits * 100) : null, averageDuration: visits && totalTime !== null ? totalTime / visits : null, activeVisitors: activeResult.status === 'fulfilled' ? optionalMetric(object(activeResult.value).visitors) : null, series: [], seriesAvailable: false }
  if (seriesResult.status === 'fulfilled') {
    try {
      const series = object(seriesResult.value), points = new Map<string, StatusTraffic['series'][number]>()
      for (const [key, field] of [['pageviews','pageviews'], ['sessions','visitors']] as const) for (const bucket of list(series[key])) {
        const b = object(bucket), time = new Date(timestamp(b.x)).toISOString(), point = points.get(time) ?? { time, pageviews: 0, visitors: 0 }
        point[field] = count(b.y); points.set(time, point)
      }
      if (points.size > 750) throw new Error('Too many traffic buckets')
      result.series = [...points.values()].sort((a,b) => a.time.localeCompare(b.time)); result.seriesAvailable = true
    } catch { /* Valid totals remain available when the chart is missing. */ }
  }
  return result
}
