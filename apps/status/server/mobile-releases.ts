const SOURCE = 'https://github.com/YourTongji/YourTJ-Hub/releases/download/mobile-notes/releases.json'
const REDIRECT_HOST = 'release-assets.githubusercontent.com'
const MAX_BYTES = 1024 * 1024
const CACHE_CONTROL = 'public, max-age=300, s-maxage=900, stale-if-error=86400'
const REDIRECTS = new Set([301, 302, 303, 307, 308])

async function readBounded(response: Response): Promise<Uint8Array> {
  const length = Number(response.headers.get('content-length'))
  if (Number.isFinite(length) && length > MAX_BYTES) throw new Error('upstream response too large')
  if (!response.body) throw new Error('upstream response is empty')
  const reader = response.body.getReader()
  const chunks: Uint8Array[] = []
  let size = 0
  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      size += value.byteLength
      if (size > MAX_BYTES) throw new Error('upstream response too large')
      chunks.push(value)
    }
  } catch (error) {
    await reader.cancel().catch(() => {})
    throw error
  }
  const result = new Uint8Array(size)
  let offset = 0
  for (const chunk of chunks) { result.set(chunk, offset); offset += chunk.byteLength }
  return result
}

export async function serveMobileReleases(request: Request, fetcher: typeof fetch = fetch): Promise<Response> {
  if (request.method !== 'GET') return new Response(null, { status: 405, headers: { Allow: 'GET', 'Cache-Control': 'no-store' } })
  try {
    let url = new URL(SOURCE)
    let upstream: Response
    const signal = AbortSignal.timeout(8000)
    for (let redirects = 0; ; redirects++) {
      upstream = await fetcher(url, { redirect: 'manual', signal })
      if (!REDIRECTS.has(upstream.status)) break
      if (redirects >= 2) { await upstream.body?.cancel(); throw new Error('too many redirects') }
      const location = upstream.headers.get('location')
      if (!location) throw new Error('missing redirect location')
      const next = new URL(location, url)
      await upstream.body?.cancel()
      if (next.protocol !== 'https:' || next.hostname !== REDIRECT_HOST || next.port || next.username || next.password)
        throw new Error('untrusted asset redirect')
      url = next
    }
    if (!upstream.ok) { await upstream.body?.cancel(); throw new Error('upstream unavailable') }
    const body = await readBounded(upstream)
    const value = JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(body)) as Record<string, unknown>
    if (!value || typeof value !== 'object' || Array.isArray(value) || value.schemaVersion !== 1
        || !Array.isArray(value.releases) || !value.historyCoverage || typeof value.historyCoverage !== 'object')
      throw new Error('invalid release catalog')
    const bodyBuffer = new ArrayBuffer(body.byteLength)
    new Uint8Array(bodyBuffer).set(body)
    const hash = new Uint8Array(await crypto.subtle.digest('SHA-256', bodyBuffer))
    const etag = `"${Array.from(hash, byte => byte.toString(16).padStart(2, '0')).join('')}"`
    const headers = { 'Content-Type': 'application/json; charset=utf-8', 'Cache-Control': CACHE_CONTROL,
      ETag: etag, 'X-Content-Type-Options': 'nosniff' }
    if (request.headers.get('if-none-match')?.split(',').map(item => item.trim()).includes(etag))
      return new Response(null, { status: 304, headers })
    return new Response(bodyBuffer, { status: 200, headers })
  } catch {
    return Response.json({ error: 'Release catalog unavailable' }, { status: 503, headers: { 'Cache-Control': 'no-store' } })
  }
}
