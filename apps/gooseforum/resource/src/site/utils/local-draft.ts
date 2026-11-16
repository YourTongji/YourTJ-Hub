// 本地草稿暂存的共享骨架（issue #1284 #35）：回复草稿（reply-draft）与快速发布
// 草稿（quick-publish-draft）此前各自手写同一套「owner 作用域 key + 7 天过期 +
// 载荷校验 + 静默失败」逻辑。key 结构与校验规则仍由各调用方持有，这里只收敛
// 读写骨架：window/owner 守卫、JSON 解析、updatedAt 新鲜度检查、损坏即删、
// 存储不可用时静默回落（不阻断编辑）。

export const DRAFT_MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000

// 草稿读写仅在浏览器且 owner 为有效正整数时生效；与两个调用方原有守卫一致。
export function draftStorageAvailable(owner: number): boolean {
  return typeof window !== 'undefined' && Number.isSafeInteger(owner) && owner > 0
}

// readDraftPayload 读取并校验一份带 updatedAt 的草稿载荷：
// 缺失返回 null；载荷不合 shape、时间戳非法/超前/超过 maxAge 时删除该 key 并
// 返回 null；存储不可用或 JSON 损坏时静默返回 null（不删 key，与原行为一致）。
export function readDraftPayload<T extends { updatedAt: number }>(
  key: string,
  isValid: (value: unknown) => value is T,
  maxAge: number = DRAFT_MAX_AGE_MS,
): T | null {
  try {
    const raw = window.localStorage.getItem(key)
    if (!raw) return null
    const parsed: unknown = JSON.parse(raw)
    if (!isValid(parsed) || !Number.isFinite(parsed.updatedAt) || parsed.updatedAt > Date.now() || Date.now() - parsed.updatedAt > maxAge) {
      // 恶意/损坏/过期载荷：直接丢弃，避免污染编辑区。
      window.localStorage.removeItem(key)
      return null
    }
    return parsed
  } catch {
    // Storage 可能不可用（隐私模式/受限浏览环境），或内容损坏——静默回落。
    return null
  }
}

// writeDraftPayload 写入载荷并盖上 updatedAt 时间戳；存储不可用（配额不足等）
// 时静默失败，不阻断编辑。
export function writeDraftPayload(key: string, payload: unknown): void {
  try {
    window.localStorage.setItem(key, JSON.stringify({ ...(payload as object), updatedAt: Date.now() }))
  } catch { /* An unavailable store must not interrupt composing. */ }
}

// removeDraftPayload 删除草稿 key；存储不可用时静默失败。
export function removeDraftPayload(key: string): void {
  try {
    window.localStorage.removeItem(key)
  } catch { /* 同上，静默失败。 */ }
}