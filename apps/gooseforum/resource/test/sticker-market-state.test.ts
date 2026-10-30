import { beforeEach, expect, test, vi } from 'vitest'
import type { StickerMarketPack } from '@gooseforum/client'

const api = vi.hoisted(() => ({
  getMyStickerPacks: vi.fn(),
  getStickerMarket: vi.fn(),
  getStickerMarketPack: vi.fn(),
  saveMyStickerPack: vi.fn(),
  deleteMyStickerPack: vi.fn(),
}))
vi.mock('@/runtime/api', () => api)

beforeEach(() => {
  vi.resetModules()
  vi.resetAllMocks()
})

const pack = (id: number) => ({ id, name: `Pack ${id}` }) as StickerMarketPack

function deferred() {
  let resolve!: () => void
  let reject!: (error: Error) => void
  const promise = new Promise<void>((yes, no) => { resolve = yes; reject = no })
  return { promise, resolve, reject }
}

for (const adding of [true, false]) {
  test(`failed ${adding ? 'add' : 'remove'} preserves another successful operation`, async () => {
    const { useStickerMarket } = await import('../src/site/composables/useStickerMarket')
    const state = useStickerMarket()
    const a = deferred()
    if (!adding) state.myPacks.value = [pack(1), pack(2)]
    const request = adding ? api.saveMyStickerPack : api.deleteMyStickerPack
    request.mockReturnValueOnce(a.promise).mockResolvedValueOnce(undefined)
    const action = adding ? state.subscribe : state.unsubscribe

    const pending = action(pack(1))
    await action(pack(2))
    a.reject(new Error('offline'))
    await pending

    expect(state.myPacks.value.map(p => p.id)).toEqual(adding ? [2] : [1])
  })
}

test('retry requests the failed second page and keeps the first page', async () => {
  const { useStickerMarket } = await import('../src/site/composables/useStickerMarket')
  const state = useStickerMarket()
  api.getStickerMarket
    .mockResolvedValueOnce({ topics: [], packs: [pack(1)], total: 25 })
    .mockRejectedValueOnce(new Error('offline'))
    .mockResolvedValueOnce({ topics: [], packs: [pack(2)], total: 25 })

  await state.loadMarket('anime')
  await state.loadMarket('anime', true)
  expect(state.marketFailed.value).toBe(true)
  await state.loadMarket('anime')

  expect(api.getStickerMarket).toHaveBeenCalledTimes(3)
  expect(api.getStickerMarket).toHaveBeenLastCalledWith({ topic: 'anime', page: 2, pageSize: 24 })
  expect(state.marketPacks.value.map(p => p.id)).toEqual([1, 2])
  expect(state.marketFailed.value).toBe(false)
})
