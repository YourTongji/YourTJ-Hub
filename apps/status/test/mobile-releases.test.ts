import { expect, it, vi } from 'vitest'
import { serveMobileReleases } from '../server/mobile-releases'

const payload = JSON.stringify({ schemaVersion: 1, releases: [], historyCoverage: { byChannel: {} } })
const asset = () => new Response(payload, { headers: { 'content-type': 'application/octet-stream' } })

it('proxies the fixed GitHub asset through the signed-off asset host and supports ETags', async () => {
  const fetcher = vi.fn<typeof fetch>()
    .mockResolvedValueOnce(new Response(null, { status: 302, headers: { location: 'https://release-assets.githubusercontent.com/object?token=opaque' } }))
    .mockImplementationOnce(async () => asset())
  const first = await serveMobileReleases(new Request('https://status.yourtj.de/mobile/releases.json'), fetcher)
  expect(first.status).toBe(200)
  expect(await first.text()).toBe(payload)
  expect(first.headers.get('cache-control')).toContain('s-maxage=900')
  const etag = first.headers.get('etag')!
  expect(String(fetcher.mock.calls[0]![0])).toBe('https://github.com/YourTongji/YourTJ-Hub/releases/download/mobile-notes/releases.json')
  expect(fetcher.mock.calls.every(call => !new Headers(call[1]?.headers).has('authorization'))).toBe(true)

  fetcher.mockImplementationOnce(async () => asset())
  const cached = await serveMobileReleases(new Request('https://status.yourtj.de/mobile/releases.json', {
    headers: { 'if-none-match': etag },
  }), fetcher)
  expect(cached.status).toBe(304)
  expect(await cached.text()).toBe('')
})

it.each([
  new Response(null, { status: 302, headers: { location: 'https://attacker.example/releases.json' } }),
  new Response('x'.repeat(1024 * 1024 + 1)),
  new Response('<html>not JSON</html>'),
  new Response('unavailable', { status: 404 }),
])('fails closed on untrusted, oversized, invalid or unavailable upstream data', async response => {
  const result = await serveMobileReleases(new Request('https://status.yourtj.de/mobile/releases.json'), async () => response)
  expect(result.status).toBe(503)
  expect(await result.json()).toEqual({ error: 'Release catalog unavailable' })
})

it('limits redirect chains and methods', async () => {
  const redirect = () => new Response(null, { status: 302,
    headers: { location: 'https://release-assets.githubusercontent.com/object' } })
  const result = await serveMobileReleases(new Request('https://status.yourtj.de/mobile/releases.json'), vi.fn<typeof fetch>()
    .mockResolvedValueOnce(redirect()).mockResolvedValueOnce(redirect()).mockResolvedValueOnce(redirect()))
  expect(result.status).toBe(503)
  expect((await serveMobileReleases(new Request('https://status.yourtj.de/mobile/releases.json', { method: 'POST' }))).status).toBe(405)
})
