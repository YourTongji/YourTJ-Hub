import { computed, ref } from 'vue'

export type PostViewMode = 'flat' | 'tree'

const storageKey = 'goose:post-view-mode'

function readStoredPreferences(): Record<string, PostViewMode> | null {
  if (typeof window === 'undefined') return null

  try {
    const raw = window.localStorage.getItem(storageKey)
    if (!raw) return null

    const parsed: unknown = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null

    const normalized: Record<string, PostViewMode> = {}
    for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (value === 'flat' || value === 'tree') normalized[key] = value
    }
    return normalized
  } catch {
    // Storage 可能不可用（隐私模式/受限浏览环境），或内容损坏——静默回落默认。
    return null
  }
}

// 模块级共享：楼层流组件可能被 KeepAlive 缓存为多个实例，状态必须全局一份
// （与 home-feed-mode.ts 同一模式）。
const storedPreferences = ref<Record<string, PostViewMode> | null>(readStoredPreferences())

function defaultModeFor(contentType: number | undefined): PostViewMode {
  // 提问话题默认树状；文章/瞬间/普通/未指定默认扁平。
  return contentType === 1 ? 'tree' : 'flat'
}

export function usePostViewMode(getContentType: () => number | undefined) {
  // computed 双向联动：contentType 变化时自动回落该类型的记忆偏好或默认；
  // storedPreferences 为响应式 ref，setViewMode 后所有实例即时同步。
  const viewMode = computed<PostViewMode>(() => {
    const contentType = getContentType()
    const stored = storedPreferences.value
    if (stored && contentType !== undefined) {
      return stored[String(contentType)] ?? defaultModeFor(contentType)
    }
    return defaultModeFor(contentType)
  })

  function setViewMode(mode: PostViewMode) {
    const contentType = getContentType()
    if (contentType === undefined) return

    const next = { ...(storedPreferences.value ?? {}) }
    next[String(contentType)] = mode
    storedPreferences.value = next

    if (typeof window === 'undefined') return
    try {
      window.localStorage.setItem(storageKey, JSON.stringify(next))
    } catch {
      // 同上，静默失败。
    }
  }

  return {
    viewMode,
    setViewMode,
  }
}
