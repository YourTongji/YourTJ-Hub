import type { StatusServer, StatusServerRange } from '../src/types'
import type { Config } from './config'
import { origin } from './config'
import { json, object, list, count, text, percentage, timestamp, type Fetcher } from './http'
export type ServerHistory = Pick<StatusServer, 'history' | 'historyAvailable'>
export const serverHours = { '1h': 1, '6h': 6, '24h': 24, '7d': 168 } as const
async function rpc(config: Config, fetcher: Fetcher, signal: AbortSignal, method: string, params: Record<string, unknown>) {
  const envelope = object(await json(fetcher, `${origin(config, 'komari')}/api/rpc2`, signal, { method: 'POST', body: JSON.stringify({ jsonrpc: '2.0', id: 1, method, params: { uuid: config.komari.id, ...params } }) }))
  if (envelope.error != null || envelope.result == null) throw new Error('Probe unavailable')
  return envelope.result
}
async function node(config: Config, fetcher: Fetcher, signal: AbortSignal) {
  const value = object(await rpc(config, fetcher, signal, 'common:getNodes', {}))
  if (value.uuid !== config.komari.id || value.hidden === true) throw new Error('Probe not public')
  return value
}
export async function fetchCurrent(config: Config, fetcher: Fetcher, signal: AbortSignal, now: number): Promise<StatusServer> {
  const [info, raw] = await Promise.all([node(config, fetcher, signal), rpc(config, fetcher, signal, 'public:getClientRecentRecords', {})])
  const recent = list(raw).map(r => ({ value: object(r), time: timestamp(object(r).updated_at) })).sort((a,b) => a.time - b.time)
  const latest = recent.at(-1)
  let current: StatusServer['current'] = null
  if (latest) {
    const r = latest.value, ram = object(r.ram), disk = object(r.disk), network = object(r.network)
    const memoryTotal = count(ram.total), diskTotal = count(disk.total)
    if (!memoryTotal || !diskTotal || latest.time > now + 60_000) throw new Error('Invalid probe sample')
    current = { observedAt: new Date(latest.time).toISOString(), cpu: percentage(object(r.cpu).usage), memoryUsed: count(ram.used), memoryTotal, diskUsed: count(disk.used), diskTotal, networkUp: count(network.up), networkDown: count(network.down), uptime: count(r.uptime) }
  }
  return { name: text(info.name), region: typeof info.region === 'string' ? info.region.slice(0,100) : '', cpuCores: count(info.cpu_cores), current, history: [], historyAvailable: false }
}
export async function fetchHistory(config: Config, fetcher: Fetcher, signal: AbortSignal, now: number, range: StatusServerRange): Promise<ServerHistory> {
  const [info, result] = await Promise.all([node(config, fetcher, signal), rpc(config, fetcher, signal, 'common:getRecords', { type: 'load', hours: serverHours[range], maxCount: 120 })])
  const raw = object(result).records
  if (raw == null) throw new Error('Missing history')
  const records = Array.isArray(raw) ? raw : list(object(raw)[config.komari.id] ?? [])
  const points: StatusServer['history'] = []
  for (const rawRecord of records) {
    const r = object(rawRecord)
    try {
      const time = timestamp(r.time), cpu = percentage(r.cpu), total = count(r.ram_total || info.mem_total), used = count(r.ram)
      if (!total || (r.client && r.client !== config.komari.id) || time < now - serverHours[range] * 3_600_000 || time > now + 60_000) continue
      points.push({ time: new Date(time).toISOString(), cpu, memoryPercent: Math.min(100, used / total * 100) })
    } catch { /* Omit malformed samples without fabricating zero readings. */ }
  }
  points.sort((a,b) => Date.parse(a.time) - Date.parse(b.time))
  const history = points.length <= 120 ? points : Array.from({ length: 120 }, (_, i) => points[Math.floor(i * (points.length - 1) / 119)]!)
  return { history, historyAvailable: true }
}
