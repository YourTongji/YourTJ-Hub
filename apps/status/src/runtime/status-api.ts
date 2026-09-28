import type { StatusRange, StatusServerRange, StatusSnapshot } from '@/types'
export async function getStatus(range: StatusRange, signal: AbortSignal, serverRange: StatusServerRange, deviceRange: StatusRange = '7d'): Promise<StatusSnapshot> {
  const response = await fetch(`/api/status?${new URLSearchParams({ range, serverRange, deviceRange })}`, { signal, credentials: 'omit', cache: 'no-store' })
  if (!response.ok) throw new Error('Status unavailable')
  const body = await response.json()
  if (body.code !== 0 || !body.result) throw new Error('Invalid status response')
  return body.result
}
