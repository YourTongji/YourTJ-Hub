export type Fetcher = typeof fetch
export const object = (v: unknown): Record<string, unknown> => v !== null && typeof v === 'object' && !Array.isArray(v) ? v as Record<string, unknown> : {}
export const list = (v: unknown): unknown[] => { if (!Array.isArray(v)) throw new Error('Invalid source array'); return v }
export const count = (v: unknown): number => { if (typeof v !== 'number' || !Number.isSafeInteger(v) || v < 0) throw new Error('Invalid source count'); return v }
export const text = (v: unknown): string => { if (typeof v !== 'string' || !v || v.length > 500) throw new Error('Invalid source text'); return v }
export const percentage = (v: unknown): number => { if (typeof v !== 'number' || !Number.isFinite(v) || v < 0 || v > 100) throw new Error('Invalid source percent'); return v }
export const uuid = (v: unknown): v is string => typeof v === 'string' && /^[a-f\d]{8}(?:-[a-f\d]{4}){3}-[a-f\d]{12}$/i.test(v)
export function timestamp(v: unknown): number {
  if (typeof v !== 'string') throw new Error('Invalid source time')
  // Unzoned provider database timestamps and date buckets are UTC.
  const normalized = /^\d{4}-\d\d-\d\d$/.test(v) ? `${v}T00:00:00Z` : /^\d{4}-\d\d-\d\d[ T]\d\d:\d\d:\d\d(?:\.\d+)?$/.test(v) ? `${v.replace(' ', 'T')}Z` : v
  if (!/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?(?:Z|[+-]\d\d:\d\d)$/.test(normalized)) throw new Error('Invalid source time')
  const n = Date.parse(normalized)
  if (!Number.isFinite(n)) throw new Error('Invalid source time')
  return n
}
export async function json(fetcher: Fetcher, url: string, signal: AbortSignal, init: RequestInit = {}): Promise<unknown> {
  const response = await fetcher(url, { ...init, signal, redirect: 'error', headers: { Accept: 'application/json', ...(init.body ? { 'Content-Type': 'application/json' } : {}), ...init.headers } })
  if (response.status !== 200 || !response.body) throw new Error('Source unavailable')
  const reader = response.body.getReader(), chunks: Uint8Array[] = []
  let size = 0
  try {
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      size += value.length
      if (size > 2 * 1024 * 1024) throw new Error('Source response too large')
      chunks.push(value)
    }
  } finally { await reader.cancel(); reader.releaseLock() }
  const buffer = new Uint8Array(size)
  let offset = 0
  for (const chunk of chunks) { buffer.set(chunk, offset); offset += chunk.length }
  return JSON.parse(new TextDecoder().decode(buffer))
}
