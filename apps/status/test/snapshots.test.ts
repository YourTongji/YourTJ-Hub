import { expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { collectOne, source, serveSnapshot, type SnapshotStore, type Stored } from '../server/snapshots'
import { cacheKey, type Config } from '../server/config'
import { collect } from '../server/collect'
import type { StatusSnapshot } from '../src/types'
const now = Date.parse('2026-09-14T12:00:00Z')
const config: Config = { enabled: true, komari: { url:'https://probe.example.com',id:'e643a364-0372-43b2-a341-b05d858866ad' },umami:{url:'https://analytics.example.com',id:'share'},uptime:{url:'https://uptime.example.com',id:'a'} }
const fixture = JSON.parse(readFileSync('test/fixtures/status-connected.json','utf8')).result as StatusSnapshot
function memoryStore() {
  const data = new Map<string,{ value: Stored<unknown>; etag: string }>(); let version=0
  const store: SnapshotStore = {
    async read<T>(key:string) { return structuredClone(data.get(key) ?? null) as { value: Stored<T>; etag:string } | null },
    async write<T>(key:string,value:Stored<T>,etag?:string) { if (data.get(key)?.etag !== etag) return false; data.set(key,{value:structuredClone(value),etag:String(++version)}); return true },
  }
  return { store,data }
}
it('retains last successful data on failure without freshening its timestamp, then expires it', async () => {
  const {store}=memoryStore(); let clock=now
  await collectOne(store,'key',async()=>({cpu:0}),()=>clock)
  clock+=60_000
  await collectOne(store,'key',async()=>{throw new Error('private provider failure')},()=>clock)
  const cached=(await store.read<{cpu:number}>('key'))!.value
  expect(source(cached,clock,150000)).toEqual({state:'stale',data:{cpu:0},fetchedAt:new Date(now).toISOString()})
  expect(source(cached,now+3_600_001,150000)).toEqual({state:'unavailable',data:null})
  expect(JSON.stringify(cached)).not.toContain('private')
})
it('a slow older run cannot overwrite the newer successful snapshot', async () => {
  const {store}=memoryStore(); let resolve!:(value:number)=>void
  const older=collectOne(store,'key',()=>new Promise<number>(r=>{resolve=r}),()=>now)
  await collectOne(store,'key',async()=>2,()=>now+60000)
  resolve(1); await older
  expect((await store.read<number>('key'))!.value.data).toBe(2)
})
it('separates live metrics from history freshness and never invokes upstreams on reads', async () => {
  const {store}=memoryStore()
  for (const [provider,part,data,age] of [ ['komari','current',fixture.server.data,0],['komari','history-24h',{history:fixture.server.data!.history,historyAvailable:true},1260000],['umami','7d',fixture.traffic.data,240000],['uptime','current',fixture.uptime.data,0] ] as const) {
    await store.write(cacheKey(config,provider,part),{attemptedAt:now-age,fetchedAt:new Date(now-age).toISOString(),data,failed:false})
  }
  const fetcher=vi.spyOn(globalThis,'fetch').mockRejectedValue(new Error('must not fetch'))
  try {
    const response=await serveSnapshot(new Request('https://status.example.com/api/status?range=7d&serverRange=24h'),store,config,now)
    const result=(await response.json()).result as StatusSnapshot
    expect(result.server.state).toBe('ok');expect(result.server.data!.historyStale).toBe(true)
    expect(result.traffic.state).toBe('ok');expect(result.uptime.state).toBe('ok')
    expect(fetcher).not.toHaveBeenCalled()
    expect(response.headers.get('Cache-Control')).toBe('no-store')
  } finally { fetcher.mockRestore() }
})
it.each([false, true])('serves retained resource history without current readings (previous current: %s)', async hasCurrent => {
  const { store } = memoryStore()
  if (hasCurrent) await store.write(cacheKey(config, 'komari', 'current'), { attemptedAt: now - 45 * 60_000, fetchedAt: new Date(now - 45 * 60_000).toISOString(), data: fixture.server.data, failed: false })
  const history = { history: fixture.server.data!.history, historyAvailable: true }
  await store.write(cacheKey(config, 'komari', 'history-1h'), { attemptedAt: now, fetchedAt: new Date(now).toISOString(), data: history, failed: false })
  const read = async (age: number) => (await (await serveSnapshot(new Request('https://status.example.com/api/status'), store, config, now + age)).json()).result as StatusSnapshot
  for (const age of [16 * 60_000, 21 * 60_000, 60 * 60_000]) {
    const result = await read(age)
    expect(result.server).toMatchObject({ state: 'unavailable', data: { current: null, cpuCores: null, ...history, historyFetchedAt: new Date(now).toISOString(), historyStale: age > 20 * 60_000 } })
    expect(result.server.fetchedAt).toBeUndefined()
  }
  expect((await read(60 * 60_000 + 1)).server).toEqual({ state: 'unavailable', data: null })
})
it('does not read previous source data when configuration is changed or disabled', async () => {
  const {store}=memoryStore()
  await store.write(cacheKey(config,'uptime','current'),{attemptedAt:now,fetchedAt:new Date(now).toISOString(),data:fixture.uptime.data,failed:false})
  const changed={...config,uptime:{...config.uptime,id:'new'}}
  const request=()=>new Request('https://status.example.com/api/status')
  expect((await (await serveSnapshot(request(),store,changed,now)).json()).result.uptime.state).toBe('unavailable')
  expect((await (await serveSnapshot(request(),store,{...config,enabled:false},now)).json()).result.uptime.state).toBe('unconfigured')
})
it.each(['range=forever','serverRange=5m','range=7d&range=24h','url=https://private.example.com'])('rejects invalid queries %s before accessing storage', async query => {
  const read=vi.fn();const store={read,write:vi.fn()} as SnapshotStore
  expect((await serveSnapshot(new Request(`https://status.example.com/api/status?${query}`),store,config,now)).status).toBe(400)
  expect(read).not.toHaveBeenCalled()
})
it('returns a noncacheable 503 on storage failure, with no provider or internal error details', async () => {
  const store={read:async()=>{throw new Error('private')},write:vi.fn()} as SnapshotStore
  const response=await serveSnapshot(new Request('https://status.example.com/api/status'),store,config,now)
  expect(response.status).toBe(503);expect(response.headers.get('Cache-Control')).toBe('no-store')
  expect(await response.text()).not.toContain('private')
})
it('scheduled collection isolates upstream failures and stays idle when disabled', async () => {
  const {store,data}=memoryStore(),fetcher=vi.fn(async()=>{throw new Error('offline')})
  await collect('current',store,config,fetcher,()=>now)
  expect(data.size).toBe(2)
  expect([...data.values()].every(v=>v.value.failed && v.value.data===null)).toBe(true)
  fetcher.mockClear();await collect('current',store,{...config,enabled:false},fetcher,()=>now)
  expect(fetcher).not.toHaveBeenCalled()
})
