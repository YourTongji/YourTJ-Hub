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

// 首页的最新、热门、流行页面会被 KeepAlive 缓存为多个组件实例，
// 因此视图模式必须放在模块级别，不能放在 HomePage 的 setup 实例中。
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
