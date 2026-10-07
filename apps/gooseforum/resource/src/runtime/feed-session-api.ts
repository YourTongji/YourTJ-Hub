import type { FeedSessionResponse } from '@gooseforum/client'
import { acknowledgeSeen, confirmPendingSeen, feedAccount, feedAccountRevision, invalidateSeen, pendingSeenPatches, resetFeedAccount } from './feed-telemetry'

async function request(path: string, body: unknown, signal?: AbortSignal): Promise<FeedSessionResponse> {
  const owner = feedAccount()
  const revision = feedAccountRevision()
  const response = await fetch(`/api/forum/feed/${path}`, { method: 'POST', credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', 'X-Goose-Feed-Version': '2' }, body: JSON.stringify(body), signal })
  if (revision !== feedAccountRevision()) throw new Error("feed account changed")
  if (response.status === 401) { resetFeedAccount(0); throw new Error('session expired') }
  if (response.status === 400 && path === 'refresh') {
    const pending = (body as { seenPatches: import('@gooseforum/client').SeenPatch[] }).seenPatches
    if (pending.length) invalidateSeen(pending)
  }
  if (!response.ok) throw new Error('feed request unavailable')
  const envelope = await response.json()
  if (revision !== feedAccountRevision()) throw new Error("feed account changed")
  if (envelope.code !== 0 || !envelope.result) throw new Error('feed request unavailable')
  const result = envelope.result as FeedSessionResponse
  if (result.viewerId !== owner || feedAccount() !== owner) {
    resetFeedAccount(result.viewerId)
    throw new Error('feed account changed')
  }
  return result
}
export async function refreshForYou(signal?: AbortSignal) {
  // Large pending queues are confirmed in bounded batches. The final small
  // queue is carried in the build request, so no five-second telemetry race.
  let pending = pendingSeenPatches()
  if (pending.length > 50 || new TextEncoder().encode(JSON.stringify({ seenPatches: pending })).length > 30000) {
    await confirmPendingSeen()
    pending = pendingSeenPatches()
  }
  const owner = feedAccount()
  const result = await request('refresh', { seenPatches: pending }, signal)
  if (!result.seenConfirmed) throw new Error('seen confirmation unavailable')
  if (owner === feedAccount()) acknowledgeSeen(pending)
  return result
}
export async function reconcileForYou(topicIds: number[], signal?: AbortSignal) {
  return request('reconcile', { topicIds }, signal)
}
