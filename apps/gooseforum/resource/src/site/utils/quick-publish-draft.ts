// 快速发布弹层（瞬间/提问）的本地草稿暂存：输入内容按登录账号、内容类型与编辑话题键自动
// 写入 localStorage，作为刷新/关闭标签页/崩溃时的本地防丢失网（不消耗每日新主题
// 上限）。与 post-view-mode.ts / home-feed-mode.ts 同一风格：window 缺失时静默
// 跳过，存储不可用或载荷损坏时静默回落。

const STORAGE_PREFIX = 'gf:quick-publish-draft:v2'
const MAX_DRAFT_AGE_MS = 7 * 24 * 60 * 60 * 1000

export interface QuickPublishDraftStash {
  title: string
  content: string
  categoryIds: number[]
  images: string[]
  updatedAt: number
}

// Never read unscoped legacy keys: their owner cannot be established.
function storageKey(userId: number, type: number, editTopicId?: number): string {
  return `${STORAGE_PREFIX}:${userId}:${type}${editTopicId ? ':edit:' + editTopicId : ''}`
}

function isStash(value: unknown): value is QuickPublishDraftStash {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false
  const record = value as Record<string, unknown>
  return (
    typeof record.title === 'string' &&
    typeof record.content === 'string' &&
    Array.isArray(record.categoryIds) &&
    record.categoryIds.every((id) => typeof id === 'number') &&
    Array.isArray(record.images) &&
    record.images.every((url) => typeof url === 'string') &&
    typeof record.updatedAt === 'number'
  )
}

export function readQuickPublishDraft(userId: number, type: number, editTopicId?: number): QuickPublishDraftStash | null {
  if (typeof window === 'undefined' || !Number.isSafeInteger(userId) || userId <= 0) return null

  try {
    const raw = window.localStorage.getItem(storageKey(userId, type, editTopicId))
    if (!raw) return null

    const parsed: unknown = JSON.parse(raw)
    if (!isStash(parsed) || !Number.isFinite(parsed.updatedAt) || parsed.updatedAt > Date.now() || Date.now() - parsed.updatedAt > MAX_DRAFT_AGE_MS) {
      // 恶意/损坏载荷：直接丢弃，避免污染编辑区。
      window.localStorage.removeItem(storageKey(userId, type, editTopicId))
      return null
    }
    return parsed
  } catch {
    // Storage 可能不可用（隐私模式/受限浏览环境），或内容损坏——静默回落。
    return null
  }
}

export function writeQuickPublishDraft(
  userId: number,
  type: number,
  stash: Omit<QuickPublishDraftStash, 'updatedAt'>,
  editTopicId?: number,
): void {
  if (typeof window === 'undefined' || !Number.isSafeInteger(userId) || userId <= 0) return

  try {
    const payload: QuickPublishDraftStash = { ...stash, updatedAt: Date.now() }
    window.localStorage.setItem(storageKey(userId, type, editTopicId), JSON.stringify(payload))
  } catch {
    // 同上，静默失败（配额不足等），不阻断编辑。
  }
}

export function clearQuickPublishDraft(userId: number, type: number, editTopicId?: number): void {
  if (typeof window === 'undefined' || !Number.isSafeInteger(userId) || userId <= 0) return

  try {
    window.localStorage.removeItem(storageKey(userId, type, editTopicId))
  } catch {
    // 同上，静默失败。
  }
}