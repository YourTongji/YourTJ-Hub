// 课程目录返回态：从目录页（/courses，含搜索/筛选 query）进入课程详情时，
// 记录当时的目录 URL；详情页「返回课程目录」据此恢复原查询状态。
// 记录保留在会话内（sessionStorage），因此详情页内多级跳转后返回仍能恢复。
const STORAGE_KEY = 'goose:courseCatalogReturn'

export function rememberCourseCatalogUrl(url: string) {
  try {
    sessionStorage.setItem(STORAGE_KEY, url)
  } catch {
    // sessionStorage 不可用（隐私模式等）时静默降级：详情页回退默认 /courses。
  }
}

// 返回可安全使用的目录 URL（pathname+search），仅接受同源且路径为 /courses 的记录。
export function readCourseCatalogReturn(base: string): string | null {
  try {
    const stored = sessionStorage.getItem(STORAGE_KEY)
    if (!stored) return null
    const baseUrl = new URL(base)
    const storedUrl = new URL(stored)
    if (storedUrl.origin !== baseUrl.origin || storedUrl.pathname !== '/courses') return null
    return storedUrl.pathname + storedUrl.search
  } catch {
    return null
  }
}
