import { createGooseClient, type StatusRange, type StatusServerRange, type StatusSnapshot } from '@gooseforum/client'

export function getStatus(range: StatusRange, signal: AbortSignal, serverRange: StatusServerRange): Promise<StatusSnapshot> {
  return createGooseClient().request('/api/forum/status', { query: { range, serverRange }, signal, cache: 'no-store' })
}
