import { ref } from 'vue'

export type PostViewMode = 'flat' | 'tree'

const storageKey = 'goose:post-view-mode'

// 全局默认扁平（issue #580：视图模式收敛为浏览器级全局偏好，不再按内容类型分层默认）。
const defaultMode: PostViewMode = 'flat'

function normalizeMode(value: unknown): PostViewMode | null {
  return value === 'flat' || value === 'tree' ? value : null
}

function readStoredMode(): PostViewMode {
  if (typeof window === 'undefined') return defaultMode

  try {
    const raw = window.localStorage.getItem(storageKey)
    if (!raw) return defaultMode

    // 新格式：全局单值（'flat' | 'tree' 原样字符串）。
    const direct = normalizeMode(raw)
    if (direct) return direct

    // 旧格式（issue #580 前）：按内容类型记忆的 map（如 {"1":"flat","3":"tree"}）。
    // 各类型选择一致时继承为全局偏好，冲突/为空/损坏时回落默认，避免静默丢弃用户偏好。
    const parsed: unknown = JSON.parse(raw)
    const parsedMode = normalizeMode(parsed)
    if (parsedMode) return parsedMode

    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      const modes = new Set<PostViewMode>()
      for (const value of Object.values(parsed as Record<string, unknown>)) {
        const mode = normalizeMode(value)
        if (mode) modes.add(mode)
      }
      if (modes.size === 1) {
        for (const mode of modes) return mode
      }
    }
    return defaultMode
  } catch {
    // Storage 可能不可用（隐私模式/受限浏览环境），或内容损坏——静默回落默认。
    return defaultMode
  }
}

// 模块级共享：楼层流组件可能被 KeepAlive 缓存为多个实例，状态必须全局一份
// （与 home-feed-mode.ts 同一模式）。全局单值意味着任一话题切换后所有话题即时跟随，
// Wiki 评论流（未传 contentType）的胶囊也因此恢复生效。
const sharedViewMode = ref<PostViewMode>(readStoredMode())

function setViewMode(mode: PostViewMode) {
  sharedViewMode.value = mode

  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(storageKey, mode)
  } catch {
    // 同上，静默失败。
  }
}

export function usePostViewMode() {
  return {
    viewMode: sharedViewMode,
    setViewMode,
  }
}
