export interface ReplyDraft { content: string; identity: 'member' | 'persona'; targetPostId: number }
const prefix = 'gf:reply-draft:v1'
const maxAge = 7 * 24 * 60 * 60 * 1000
function key(owner: number, topic: number) { return `${prefix}:${owner}:${topic}` }
export function readReplyDraft(owner: number, topic: number): ReplyDraft | null {
  if (typeof window === 'undefined' || owner <= 0) return null
  try {
    const raw = window.localStorage.getItem(key(owner, topic))
    if (!raw) return null
    const d = JSON.parse(raw)
    if (typeof d.content !== 'string' || !['member', 'persona'].includes(d.identity) || !Number.isSafeInteger(d.targetPostId) || d.targetPostId < 0 || !Number.isFinite(d.updatedAt) || d.updatedAt > Date.now() || Date.now() - d.updatedAt > maxAge) {
      window.localStorage.removeItem(key(owner, topic))
      return null
    }
    return d
  } catch { return null }
}
export function writeReplyDraft(owner: number, topic: number, draft: ReplyDraft) {
  if (typeof window === 'undefined' || owner <= 0) return
  try {
    if (!draft.content.trim()) window.localStorage.removeItem(key(owner, topic))
    else window.localStorage.setItem(key(owner, topic), JSON.stringify({ ...draft, updatedAt: Date.now() }))
  } catch { /* An unavailable store must not interrupt composing. */ }
}
