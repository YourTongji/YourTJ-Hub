import { ref } from 'vue'
import { resolveForumStickers } from '@/runtime/api'
import { stickerNamesInContent } from '@/site/utils/sticker-token'

/** Message-only resolution never adds personal assets to the official picker. */
export function useResolvedStickers() {
  const resolvedUrls = ref(new Map<string, string>())
  const checkedAt = new Map<string, number>()
  const pending = new Set<string>()

  async function ensureContentStickers(contents: readonly string[]) {
    const now = Date.now()
    const names = stickerNamesInContent(contents).filter((name) =>
      !pending.has(name) && now - (checkedAt.get(name) ?? -Infinity) >= 60_000,
    )
    for (const name of names) pending.add(name)
    try {
      for (let offset = 0; offset < names.length; offset += 200) {
        const batch = names.slice(offset, offset + 200)
        const items = await resolveForumStickers(batch)
        // A removed/disabled definition also expires an earlier image.
        const next = new Map(resolvedUrls.value)
        for (const name of batch) {
          next.delete(name)
          checkedAt.set(name, Date.now())
        }
        for (const item of items) next.set(item.name, item.url)
        resolvedUrls.value = next
      }
    } catch {
      // Keep already rendered content; failures remain eligible on the next
      // message/history change instead of caching an empty successful library.
    } finally {
      for (const name of names) pending.delete(name)
    }
  }

  return { resolvedUrls, ensureContentStickers }
}
