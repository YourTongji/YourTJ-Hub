import { expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { collectOne, source, serveSnapshot, type SnapshotStore, type Stored } from '../server/snapshots'
import { cacheKey, type Config } from '../server/config'
import { collect } from '../server/collect'
import { storeName } from '../server/blobs'
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
  expect(source(cached,now+900001,150000)).toEqual({state:'unavailable',data:null})
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
  for (const [provider,part,data,age] of [ ['komari','current',fixture.server.data,0],['komari','history-24h',{history:fixture.server.data!.history,historyAvailable:true},660000],['umami','7d',fixture.traffic.data,240000],['uptime','current',fixture.uptime.data,0] ] as const) {
    await store.write(cacheKey(config,provider,part),{attemptedAt:now-age,fetchedAt:new Date(now-age).toISOString(),data,failed:false})
  }
  const fetcher=vi.spyOn(globalThis,'fetch').mockRejectedValue(new Error('must not fetch'))
  try {
    const response=await serveSnapshot(new Request('https://status.example.com/api/status?range=7d&serverRange=24h'),store,config,now)
    const result=(await response.json()).result as StatusSnapshot
    expect(result.server.state).toBe('ok');expect(result.server.data!.historyStale).toBe(true)
    expect(result.traffic.state).toBe('ok');expect(result.uptime.state).toBe('ok')
    expect(fetcher).not.toHaveBeenCalled()
    expect(response.headers.get('Netlify-CDN-Cache-Control')).toContain('max-age=15')
    expect(response.headers.get('Netlify-Vary')).toBe('query=range|serverRange')
  } finally { fetcher.mockRestore() }
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
  expect(response.status).toBe(503);expect(response.headers.get('Netlify-CDN-Cache-Control')).toBeNull()
  expect(await response.text()).not.toContain('private')
})
it('reads scheduled production snapshots even when the invocation published flags differ', async () => {
  const stores = new Map<string, SnapshotStore>()
  function deployedStore(published: boolean, id = 'production-deploy') {
    const name = storeName({ context: 'production', published, id })
    if (!stores.has(name)) stores.set(name, memoryStore().store)
    return stores.get(name)!
  }
  await collectOne(deployedStore(false), cacheKey(config, 'uptime', 'current'), async () => fixture.uptime.data, () => now)
  for (const id of ['production-deploy', 'next-production-deploy']) {
    const response = await serveSnapshot(new Request('https://status.example.com/api/status'), deployedStore(true, id), config, now)
    expect((await response.json()).result.uptime).toMatchObject({ state: 'ok', data: fixture.uptime.data })
  }
})
it.each(['deploy-preview', 'branch-deploy'])('keeps %s snapshots isolated even when published', context => {
  for (const published of [false, true]) {
    const first = storeName({ context, published, id: 'one' })
    expect(first).not.toBe('status-v1')
    expect(first).not.toBe(storeName({ context, published, id: 'two' }))
  }
  expect(storeName({ context: 'dev', published: false, id: '' })).toBe('status-local-v1')
})
it('scheduled collection isolates upstream failures and stays idle when disabled', async () => {
  const {store,data}=memoryStore(),fetcher=vi.fn(async()=>{throw new Error('offline')})
  await collect('current',store,config,fetcher,()=>now)
  expect(data.size).toBe(2)
  expect([...data.values()].every(v=>v.value.failed && v.value.data===null)).toBe(true)
  fetcher.mockClear();await collect('current',store,{...config,enabled:false},fetcher,()=>now)
  expect(fetcher).not.toHaveBeenCalled()
})

it('uses matching list versions when the local Blobs emulator omits GET ETags', async () => {
  const { adaptStore } = await import('../server/blobs')
  const current={data:{attemptedAt:now,data:{cpu:1},failed:false},metadata:{}}
  const raw={getWithMetadata:vi.fn().mockResolvedValue(current),list:vi.fn().mockResolvedValue({blobs:[{key:'key',etag:'v1'}]}),setJSON:vi.fn().mockResolvedValue({modified:true})}
  const store=adaptStore(raw as unknown as Parameters<typeof adaptStore>[0])
  expect(await store.read('key')).toEqual({value:current.data,etag:'v1'})
  await store.write('key',current.data,'v1')
  expect(raw.setJSON).toHaveBeenLastCalledWith('key',current.data,{onlyIfMatch:'v1'})
  raw.list.mockResolvedValueOnce({blobs:[{key:'key',etag:'v1'}]}).mockResolvedValue({blobs:[{key:'key',etag:'v2'}]})
  raw.getWithMetadata.mockResolvedValueOnce(current).mockResolvedValueOnce(current).mockResolvedValue({...current,data:{...current.data,data:{cpu:2}}})
  expect(await store.read('key')).toMatchObject({value:{data:{cpu:2}},etag:'v2'})
})
