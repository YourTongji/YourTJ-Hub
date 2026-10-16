import { afterAll, beforeAll, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { transpileModule, ModuleKind, ScriptTarget } from 'typescript'
import { Miniflare, convertV4MiniflareOptions } from 'miniflare'

let mf: Miniflare
let status = 204
const calls: Array<{ url: string, method: string, authorization: string | null }> = []

beforeAll(async () => {
  const code = transpileModule(readFileSync('worker/scheduler.ts', 'utf8'), {
    compilerOptions: { module: ModuleKind.ESNext, target: ScriptTarget.ES2022 },
  }).outputText.replace('export default', 'const handler =')
  // Invoke the shipped scheduled handler inside workerd. Only the transport is
  // replaced: native Request/fetch validation must run, unlike a Node fetch spy.
  mf = new Miniflare(convertV4MiniflareOptions({
    modules: true,
    compatibilityDate: '2026-10-03',
    bindings: { GITHUB_DISPATCH_TOKEN: 'test-only-token' },
    script: `${code}
      export default { async fetch(request, env) {
        try {
          await handler.scheduled({ cron: '2,17,32,47 * * * *' }, env)
          return new Response(null, { status: 204 })
        } catch (error) { return new Response(error.message, { status: 502 }) }
      } }
    `,
    outboundService: async request => {
      calls.push({ url: request.url, method: request.method, authorization: request.headers.get('Authorization') })
      if (request.url.includes('/runs?') && request.url.includes('status=waiting')) {
        return new Response(JSON.stringify({ workflow_runs: [] }), { status: 200 })
      }
      return new Response(null, { status, headers: status === 302 ? { Location: 'https://other.example/private' } : {} })
    },
  }))
  await mf.ready
})
afterAll(async () => { await mf?.dispose() })

it('dispatches in Workers and rejects redirects without forwarding the token', async () => {
  const success = await mf.dispatchFetch('http://worker/trigger')
  expect(await success.text()).toBe('')
  expect(success.status).toBe(204)
  status = 302
  const redirect = await mf.dispatchFetch('http://worker/trigger')
  expect(redirect.status).toBe(502)
  expect(await redirect.text()).toBe('Status dispatch rejected (302)')
  expect(calls).toHaveLength(4)
  for (const i of [0, 2]) {
    expect(calls[i].method).toBe('GET')
    expect(calls[i].authorization).toBe('Bearer test-only-token')
    expect(calls[i].url).toContain('/actions/workflows/collect-status.yml/runs?')
    expect(calls[i].url).toContain('status=waiting')
    expect(calls[i].url).toContain('branch=main')
    expect(calls[i].url).toContain('per_page=100')
  }
  for (const i of [1, 3]) {
    expect(calls[i]).toEqual({
      url: 'https://api.github.com/repos/YourTongji/YourTJ-Hub/actions/workflows/collect-status.yml/dispatches',
      method: 'POST',
      authorization: 'Bearer test-only-token',
    })
  }
})
