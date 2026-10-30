import { ref } from 'vue'
import type { StickerItem, StickerMarketPack, StickerMarketTopic } from '@gooseforum/client'
import {
  deleteMyStickerPack,
  getMyStickerPacks,
  getStickerMarket,
  getStickerMarketPack,
  saveMyStickerPack,
} from '@/runtime/api'

/**
 * Sticker market state for the editor picker (issue #1429). Like useStickerLibrary,
 * the cache is module-level: every editor on the page shares one copy of the
 * subscribed packs and of each pack's stickers, and concurrent requests for the
 * same pack share one promise. Subscriptions update optimistically and roll back
 * when the server rejects them.
 */

const MARKET_PAGE_SIZE = 24

const myPacks = ref<StickerMarketPack[]>([])
const myPacksReady = ref(false)
let myPacksRequest: Promise<void> | null = null

const packStickers = new Map<number, StickerItem[]>()
const packRequests = new Map<number, Promise<StickerItem[]>>()

const topics = ref<StickerMarketTopic[]>([])
const marketPacks = ref<StickerMarketPack[]>([])
const marketTopic = ref('')
const marketTotal = ref(0)
let marketPage = 0
let marketRequest: Promise<void> | null = null
let failedMarketRequest: { topic: string, page: number, more: boolean } | null = null

export function useStickerMarket() {
  const marketLoading = ref(false)
  const marketFailed = ref(false)
  const actionError = ref('')

  /** Load subscriptions once; signed-out viewers simply see no packs. */
  function ensureMyPacks() {
    if (myPacksReady.value) return Promise.resolve()
    myPacksRequest ??= getMyStickerPacks()
      .then((packs) => {
        myPacks.value = packs
        myPacksReady.value = true
      })
      .catch(() => undefined)
      .finally(() => {
        myPacksRequest = null
      })
    return myPacksRequest
  }

  function loadPackStickers(id: number): Promise<StickerItem[]> {
    const cached = packStickers.get(id)
    if (cached) return Promise.resolve(cached)
    let request = packRequests.get(id)
    if (!request) {
      request = getStickerMarketPack(id)
        .then((detail) => {
          packStickers.set(id, detail.stickers)
          return detail.stickers
        })
        .finally(() => packRequests.delete(id))
      packRequests.set(id, request)
    }
    return request
  }

  async function loadMarket(topic = marketTopic.value, more = false) {
    if (marketRequest) return marketRequest
    const retry = failedMarketRequest?.topic === topic ? failedMarketRequest : null
    if (!retry && !more && topic === marketTopic.value && marketPage > 0) return
    const page = retry?.page ?? (more ? marketPage + 1 : 1)
    more = retry?.more ?? more
    marketLoading.value = true
    marketFailed.value = false
    marketRequest = getStickerMarket({ topic, page, pageSize: MARKET_PAGE_SIZE })
      .then((result) => {
        topics.value = result.topics
        marketPacks.value = more ? [...marketPacks.value, ...result.packs] : result.packs
        marketTotal.value = result.total
        marketTopic.value = topic
        marketPage = page
        failedMarketRequest = null
      })
      .catch(() => {
        failedMarketRequest = { topic, page, more }
        marketFailed.value = true
      })
      .finally(() => {
        marketLoading.value = false
        marketRequest = null
      })
    return marketRequest
  }

  function isSubscribed(id: number) {
    return myPacks.value.some((pack) => pack.id === id)
  }

  async function subscribe(pack: StickerMarketPack) {
    if (isSubscribed(pack.id)) return true
    const previous = myPacks.value
    myPacks.value = [...previous, pack]
    actionError.value = ''
    try {
      await saveMyStickerPack(pack.id)
      return true
    } catch (err) {
      myPacks.value = myPacks.value.filter((item) => item.id !== pack.id)
      actionError.value = err instanceof Error ? err.message : ''
      return false
    }
  }

  async function unsubscribe(pack: StickerMarketPack) {
    const previous = myPacks.value
    myPacks.value = previous.filter((item) => item.id !== pack.id)
    actionError.value = ''
    try {
      await deleteMyStickerPack(pack.id)
      return true
    } catch (err) {
      if (previous.some((item) => item.id === pack.id) && !isSubscribed(pack.id)) {
        const restored = [...myPacks.value]
        restored.splice(Math.min(previous.findIndex((item) => item.id === pack.id), restored.length), 0, pack)
        myPacks.value = restored
      }
      actionError.value = err instanceof Error ? err.message : ''
      return false
    }
  }

  return {
    myPacks,
    myPacksReady,
    ensureMyPacks,
    loadPackStickers,
    topics,
    marketPacks,
    marketTopic,
    marketTotal,
    marketLoading,
    marketFailed,
    loadMarket,
    isSubscribed,
    subscribe,
    unsubscribe,
    actionError,
  }
}
