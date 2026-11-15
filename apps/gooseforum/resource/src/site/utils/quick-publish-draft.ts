// 快速发布弹层（瞬间/提问）的本地草稿暂存：输入内容按登录账号、内容类型与编辑话题键自动
// 写入 localStorage，作为刷新/关闭标签页/崩溃时的本地防丢失网（不消耗每日新主题
// 上限）。读写骨架与回复草稿共享 local-draft（issue #1284 #35）：window 缺失时
// 静默跳过，存储不可用或载荷损坏时静默回落。

import { draftStorageAvailable, readDraftPayload, removeDraftPayload, writeDraftPayload } from './local-draft'

const STORAGE_PREFIX = 'gf:quick-publish-draft:v2'

export interface QuickPublishDraftStash {
  agentRepliesDisabled?: boolean
  identity?: 'member' | 'persona'
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
    (record.agentRepliesDisabled === undefined || typeof record.agentRepliesDisabled === 'boolean') &&
    (record.identity === undefined || record.identity === 'member' || record.identity === 'persona') &&
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
  if (!draftStorageAvailable(userId)) return null
  return readDraftPayload(storageKey(userId, type, editTopicId), isStash)
}

export function writeQuickPublishDraft(
  userId: number,
  type: number,
  stash: Omit<QuickPublishDraftStash, 'updatedAt'>,
  editTopicId?: number,
): void {
  if (!draftStorageAvailable(userId)) return
  writeDraftPayload(storageKey(userId, type, editTopicId), stash)
}

export function clearQuickPublishDraft(userId: number, type: number, editTopicId?: number): void {
  if (!draftStorageAvailable(userId)) return
  removeDraftPayload(storageKey(userId, type, editTopicId))
}