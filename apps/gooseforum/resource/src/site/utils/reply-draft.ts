import { draftStorageAvailable, readDraftPayload, removeDraftPayload, writeDraftPayload } from './local-draft'

export interface ReplyDraft { content: string; identity: 'member' | 'persona'; targetPostId: number }
const prefix = 'gf:reply-draft:v1'
function key(owner: number, topic: number) { return `${prefix}:${owner}:${topic}` }
function isReplyDraft(value: unknown): value is ReplyDraft & { updatedAt: number } {
  if (!value || typeof value !== 'object') return false
  const d = value as Record<string, unknown>
  return typeof d.content === 'string' && (d.identity === 'member' || d.identity === 'persona') &&
    typeof d.targetPostId === 'number' && Number.isSafeInteger(d.targetPostId) && d.targetPostId >= 0 &&
    typeof d.updatedAt === 'number'
}
export function readReplyDraft(owner: number, topic: number): ReplyDraft | null {
  if (!draftStorageAvailable(owner)) return null
  return readDraftPayload(key(owner, topic), isReplyDraft)
}
export function writeReplyDraft(owner: number, topic: number, draft: ReplyDraft) {
  if (!draftStorageAvailable(owner)) return
  if (!draft.content.trim()) {
    removeDraftPayload(key(owner, topic))
    return
  }
  writeDraftPayload(key(owner, topic), draft)
}