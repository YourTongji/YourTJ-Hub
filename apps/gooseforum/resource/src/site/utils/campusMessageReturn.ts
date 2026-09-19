// One tab's short-lived navigation intent only: never persist school payloads,
// titles, credentials or authorization parameters across the OAuth round trip.
const key = 'yourtj:campus-message-return'
export interface CampusMessageReturn {
  userId: number
  revision: string
  messageId: string
  tab: 'overview' | 'messages'
  expiresAt: number
}
export function saveCampusMessageReturn(intent: Omit<CampusMessageReturn, 'expiresAt'> | null) {
  try {
    sessionStorage.removeItem(key)
    if (intent) sessionStorage.setItem(key, JSON.stringify({ ...intent, expiresAt: Date.now() + 10 * 60_000 }))
  } catch { /* Storage can be disabled; authorization remains usable. */ }
}
export function takeCampusMessageReturn(): CampusMessageReturn | null {
  try {
    const value = sessionStorage.getItem(key)
    sessionStorage.removeItem(key)
    if (!value) return null
    const intent = JSON.parse(value) as CampusMessageReturn
    if (!Number.isSafeInteger(intent.userId) || intent.userId <= 0 || typeof intent.revision !== 'string' ||
      typeof intent.messageId !== 'string' || !/^[A-Za-z0-9_-]{1,128}$/.test(intent.messageId) ||
      !['overview', 'messages'].includes(intent.tab) || !Number.isFinite(intent.expiresAt) ||
      intent.expiresAt <= Date.now() || intent.expiresAt > Date.now() + 10 * 60_000) return null
    return intent
  } catch { return null }
}
