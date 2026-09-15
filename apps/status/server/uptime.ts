import type { StatusUptime, StatusUptimeMonitor } from '../src/types'
import type { Config } from './config'
import { origin } from './config'
import { json, object, list, count, text, timestamp, type Fetcher } from './http'
export async function fetchUptime(config: Config, fetcher: Fetcher, signal: AbortSignal, now: number): Promise<StatusUptime> {
  const base = origin(config, 'uptime'), slug = config.uptime.id
  const [pageRaw, beatsRaw] = await Promise.all([json(fetcher, `${base}/api/status-page/${slug}`, signal), json(fetcher, `${base}/api/status-page/heartbeat/${slug}`, signal)])
  const page = object(pageRaw), meta = object(page.config), beats = object(beatsRaw)
  if (meta.published !== true || meta.slug !== slug || !beats.heartbeatList || Array.isArray(beats.heartbeatList)) throw new Error('Unpublished status source')
  const heartbeats = object(beats.heartbeatList), ratios = object(beats.uptimeList), monitors: StatusUptimeMonitor[] = [], seen = new Set<number>()
  for (const group of list(page.publicGroupList)) for (const raw of list(object(group).monitorList)) {
    const item = object(raw), id = count(item.id)
    if (!id || !item.name || seen.has(id)) continue
    if (monitors.length >= 50) throw new Error('Too many public monitors')
    seen.add(id)
    const history: StatusUptimeMonitor['history'] = []
    let invalidTime = false
    for (const rawBeat of list(heartbeats[String(id)] ?? [])) {
      const beat = object(rawBeat)
      let time: number
      try { time = timestamp(beat.time); if (time > now + 60_000) throw new Error('Future check') } catch { invalidTime = true; continue }
      const status = typeof beat.status === 'number' ? ({ 0: 'down', 1: 'up', 2: 'pending', 3: 'maintenance' } as const)[beat.status as 0 | 1 | 2 | 3] ?? 'unknown' : 'unknown'
      const ping = typeof beat.ping === 'number' && Number.isFinite(beat.ping) && beat.ping >= 0 ? beat.ping : null
      history.push({ time: new Date(time).toISOString(), status, ping })
    }
    history.sort((a,b) => Date.parse(a.time) - Date.parse(b.time))
    const kept = history.slice(-100), ratio = ratios[`${id}_24`]
    monitors.push({ id, name: text(item.name), type: typeof item.type === 'string' ? item.type.slice(0, 100) : '', uptime24h: typeof ratio === 'number' && ratio >= 0 && ratio <= 1 ? ratio * 100 : null, current: invalidTime ? null : kept.at(-1) ?? null, history: kept })
  }
  return { statusPageUrl: `${base}/status/${slug}`, monitors }
}
