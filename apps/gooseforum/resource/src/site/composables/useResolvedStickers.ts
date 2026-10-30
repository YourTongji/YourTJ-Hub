import { ref } from 'vue'
import { resolveForumStickers } from '@/runtime/api'
import { stickerNamesInContent } from '@/site/utils/sticker-token'

/**
 * 模块级共享解析状态：聊天、转发卡片与各个 TopicList 实例读写同一份缓存，
 * 同页多个列表（或聊天与列表）在 60s TTL 内只解析一次同名贴纸（issue #1304
 * 已知缺口：实例级缓存会让同页两个列表各发一次请求）。
 */
const sharedResolvedUrls = ref(new Map<string, string>())
const sharedCheckedAt = new Map<string, number>()
const sharedPending = new Set<string>()

/**
 * 清空共享缓存：模块级状态跨组件实例存活，vitest 同文件内多个用例之间
 * 需要显式重置（生产环境不应调用）。
 */
export function resetResolvedStickerCache() {
  sharedResolvedUrls.value = new Map<string, string>()
  sharedCheckedAt.clear()
  sharedPending.clear()
}

/**
 * 选择面板与贴纸库里已拿到的启用贴纸直接记入共享缓存：编辑器插入后立刻显示为
 * 图片，不必等一次 resolve 往返。只应传入可用（启用且已通过审核）的贴纸。
 */
export function rememberStickerUrls(items: readonly { name: string, url: string }[]) {
  const fresh = items.filter((item) => item.url && sharedResolvedUrls.value.get(item.name) !== item.url)
  const now = Date.now()
  for (const item of items) {
    if (item.url) sharedCheckedAt.set(item.name, now)
  }
  if (fresh.length === 0) return
  const next = new Map(sharedResolvedUrls.value)
  for (const item of fresh) next.set(item.name, item.url)
  sharedResolvedUrls.value = next
}

/** Message-only resolution never adds personal assets to the official picker. */
export function useResolvedStickers() {
  async function ensureContentStickers(contents: readonly string[]) {
    const now = Date.now()
    const names = stickerNamesInContent(contents).filter((name) =>
      !sharedPending.has(name) && now - (sharedCheckedAt.get(name) ?? -Infinity) >= 60_000,
    )
    for (const name of names) sharedPending.add(name)
    try {
      for (let offset = 0; offset < names.length; offset += 200) {
        const batch = names.slice(offset, offset + 200)
        const items = await resolveForumStickers(batch)
        // A removed/disabled definition also expires an earlier image.
        const next = new Map(sharedResolvedUrls.value)
        for (const name of batch) {
          next.delete(name)
          sharedCheckedAt.set(name, Date.now())
        }
        for (const item of items) next.set(item.name, item.url)
        sharedResolvedUrls.value = next
      }
    } catch {
      // Keep already rendered content; failures remain eligible on the next
      // message/history change instead of caching an empty successful library.
    } finally {
      for (const name of names) sharedPending.delete(name)
    }
  }

  return { resolvedUrls: sharedResolvedUrls, ensureContentStickers }
}
