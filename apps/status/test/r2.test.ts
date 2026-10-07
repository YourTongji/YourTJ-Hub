import { afterAll, beforeAll, expect, it } from 'vitest'
import { Miniflare, convertV4MiniflareOptions } from 'miniflare'
import type { R2Bucket } from '@cloudflare/workers-types'
import { r2Store } from '../server/r2'
import { collectOne, source, type Stored } from '../server/snapshots'

const mf = new Miniflare(convertV4MiniflareOptions({ modules: true, script: 'export default { fetch() { return new Response("ok") } }', r2Buckets: ['PRODUCTION', 'PREVIEW'] }))
let bucket: R2Bucket
beforeAll(async () => { bucket = await mf.getR2Bucket('PRODUCTION') as unknown as R2Bucket })
afterAll(() => mf.dispose())
const value = (n: number): Stored<number> => ({ attemptedAt: n, fetchedAt: new Date(n).toISOString(), failed: false, data: n })

it('uses real R2 conditional writes for initial creation and compare-and-swap updates', async () => {
  const store = r2Store(bucket)
  expect(await store.read('cas')).toBeNull()
  const created = await Promise.all([store.write('cas', value(1)), store.write('cas', value(2))])
  expect(created.filter(Boolean)).toHaveLength(1)
  const old = (await store.read<number>('cas'))!
  const updated = await Promise.all([store.write('cas', value(3), old.etag), store.write('cas', value(4), old.etag)])
  expect(updated.filter(Boolean)).toHaveLength(1)
  expect(await store.write('cas', value(5), old.etag)).toBe(false)
  expect(await store.write('cas', value(5))).toBe(false)
  expect([3, 4]).toContain((await store.read<number>('cas'))!.value.data)
})

it('keeps production snapshots across adapter instances and isolates preview buckets', async () => {
  await r2Store(bucket).write('isolation', value(1))
  expect((await r2Store(bucket).read('isolation'))?.value).toEqual(value(1))
  const preview = r2Store(await mf.getR2Bucket('PREVIEW') as unknown as R2Bucket)
  expect(await preview.read('isolation')).toBeNull()
  await preview.write('isolation', value(2))
  expect((await r2Store(bucket).read('isolation'))?.value).toEqual(value(1))
})

it('preserves new data against late collectors and preserves success age on provider failure', async () => {
  const store = r2Store(bucket)
  let resolve!: (n: number) => void
  const slow = collectOne(store, 'late', () => new Promise<number>(r => { resolve = r }), () => 1_000)
  await collectOne(store, 'late', async () => 2, () => 2_000)
  resolve(1)
  await slow
  await collectOne(store, 'late', async () => { throw new Error('private upstream error') }, () => 3_000)
  const saved = (await store.read<number>('late'))!.value
  expect(source(saved, 3_000, 150_000)).toEqual({ state: 'stale', fetchedAt: new Date(2_000).toISOString(), data: 2 })
  expect(source(saved, 3_602_001, 150_000)).toEqual({ state: 'unavailable', data: null })
})
