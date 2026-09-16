import { ref } from 'vue'
import { getForumStickers } from '@/runtime/api'
import type { StickerItem } from '@gooseforum/client'

/**
 * 启用表情包库（MADR 0030）：编辑器选择面板数据源。
 * 模块级缓存全站共享一份（发布页 / 回复面板 / 快捷发布弹窗三个编辑器实例
 * 只发一次请求）；拉取失败后下次打开面板自动重试。
 */

const stickers = ref<StickerItem[]>([])
let loaded = false

export function useStickerLibrary() {
  const loading = ref(false)
  const failed = ref(false)

  async function ensureStickers() {
    if (loaded || loading.value) return
    loading.value = true
    failed.value = false
    try {
      stickers.value = await getForumStickers()
      loaded = true
    } catch {
      failed.value = true
    } finally {
      loading.value = false
    }
  }

  return {
    stickers,
    /** 首次打开面板时拉取；已加载则直接复用缓存 */
    ensureStickers,
    loading,
    failed,
  }
}
