import { ref } from 'vue'

export type AvatarClickBehavior = 'card' | 'profile'

const avatarClickBehaviorStorageKey = 'goose:avatar-click-behavior'

function readAvatarClickBehavior(): AvatarClickBehavior {
  if (typeof window === 'undefined') return 'card'

  try {
    const stored = window.localStorage.getItem(avatarClickBehaviorStorageKey)
    if (stored === 'card' || stored === 'profile') return stored
  } catch {
    // Storage may be unavailable in private or restricted browsing contexts.
  }

  // 无值、非法值（含未登录访客）回退默认：点击头像弹用户卡片（issue #1134）。
  return 'card'
}

// 头像点击行为是跨页面的阅读偏好：SPA 导航复用同一个模块实例，
// 登录/重置密码/校园地图这类独立页往返会重建组件，因此状态放在模块级别
// （与 shell-sidebar / home-feed-mode 同一模式）。
const sharedAvatarClickBehavior = ref<AvatarClickBehavior>(readAvatarClickBehavior())

// 偏好是设备级的：同一浏览器的其他标签页改动后经 storage 事件同步到本标签页，
// 避免旧标签页在刷新前一直沿用切换前的点击行为（key 为 null 表示 storage.clear()）。
if (typeof window !== 'undefined') {
  window.addEventListener('storage', (event) => {
    if (event.key !== null && event.key !== avatarClickBehaviorStorageKey) return
    sharedAvatarClickBehavior.value = readAvatarClickBehavior()
  })
}

function setAvatarClickBehavior(behavior: AvatarClickBehavior) {
  sharedAvatarClickBehavior.value = behavior
  if (typeof window === 'undefined') return

  try {
    window.localStorage.setItem(avatarClickBehaviorStorageKey, behavior)
  } catch {
    // Storage may be unavailable in private or restricted browsing contexts.
  }
}

/**
 * 设置页「通用设置 → 恢复默认」：回到弹出用户卡片并清除持久化。
 * 该按钮位于通用设置卡片末尾，确认文案在四种语言中都写明「通用设置」，
 * 因此清除本偏好属于按钮声明的范围（决策 0085）。
 */
function restoreAvatarClickBehavior() {
  sharedAvatarClickBehavior.value = 'card'
  if (typeof window === 'undefined') return

  try {
    window.localStorage.removeItem(avatarClickBehaviorStorageKey)
  } catch {
    // Storage may be unavailable in private or restricted browsing contexts.
  }
}

export function useAvatarClickBehavior() {
  return {
    avatarClickBehavior: sharedAvatarClickBehavior,
    setAvatarClickBehavior,
    restoreAvatarClickBehavior,
  }
}
