import { ref } from 'vue'
import { getForumStickerPacks, getForumStickers } from '@/runtime/api'
import type { StickerItem, StickerOfficialPack } from '@gooseforum/client'
import { rememberStickerUrls } from '@/site/composables/useResolvedStickers'

/**
 * 启用表情包库（MADR 0030）：编辑器选择面板数据源。
 * 模块级缓存全站共享一份（发布页 / 回复面板 / 快捷发布弹窗三个编辑器实例
 * 只发一次请求）；拉取失败后下次打开面板自动重试。
 * 官方表情包分组（名称、封面、顺序）随贴纸一起拉取；它只用于分组展示，
 * 失败或旧服务端不支持时按贴纸的 pack 键回退，不影响贴纸本身。
 */

const stickers = ref<StickerItem[]>([])
const officialPacks = ref<StickerOfficialPack[]>([])
let loaded = false
let packsLoaded = false

export function useStickerLibrary() {
  const loading = ref(false)
  const failed = ref(false)

  async function ensureStickers() {
    if (!packsLoaded) {
      packsLoaded = true
      // Deferred so a pack failure of any kind can never block the stickers.
      Promise.resolve()
        .then(() => getForumStickerPacks())
        .then((packs) => {
          officialPacks.value = packs
        })
        .catch(() => {
          packsLoaded = false
        })
    }
    if (loaded || loading.value) return
    loading.value = true
    failed.value = false
    try {
      stickers.value = await getForumStickers()
      loaded = true
      rememberStickerUrls(stickers.value)
    } catch {
      failed.value = true
    } finally {
      loading.value = false
    }
  }

  return {
    stickers,
    officialPacks,
    /** 首次打开面板时拉取；已加载则直接复用缓存 */
    ensureStickers,
    loading,
    failed,
  }
}
