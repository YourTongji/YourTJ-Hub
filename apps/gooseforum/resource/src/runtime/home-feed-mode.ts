import { ref } from 'vue'

export type HomeFeedMode = 'table' | 'card'

const feedStorageKey = 'goose:home-feed-mode'

function readFeedMode(): HomeFeedMode {
  if (typeof window === 'undefined') return 'table'

  try {
    const stored = window.localStorage.getItem(feedStorageKey)
    if (stored === 'card' || stored === 'table') return stored
  } catch {
    // Storage may be unavailable in private or restricted browsing contexts.
  }

  // 未手动选择时按设备默认：移动端卡片、桌面端列表（与 Tailwind lg 断点一致）。
  return typeof window.matchMedia === 'function' && window.matchMedia('(min-width: 1024px)').matches
    ? 'table'
    : 'card'
}

// 首页组件在 KeepAlive 恢复或重新创建时共享视图模式。
const sharedFeedMode = ref<HomeFeedMode>(readFeedMode())

function setFeedMode(mode: HomeFeedMode) {
  sharedFeedMode.value = mode
  if (typeof window === 'undefined') return

  try {
    window.localStorage.setItem(feedStorageKey, mode)
  } catch {
    // Storage may be unavailable in private or restricted browsing contexts.
  }
}

export function useHomeFeedMode() {
  return {
    feedMode: sharedFeedMode,
    setFeedMode,
  }
}
