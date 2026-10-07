import { afterAll, beforeAll, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { transpileModule, ModuleKind, ScriptTarget } from 'typescript'
import { Miniflare, convertV4MiniflareOptions } from 'miniflare'

let mf: Miniflare
let status = 204
const calls: Array<{ url: string, authorization: string | null }> = []

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
      calls.push({ url: request.url, authorization: request.headers.get('Authorization') })
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
  expect(calls).toEqual(Array.from({ length: 2 }, () => ({
    url: 'https://api.github.com/repos/YourTongji/YourTJ-Hub/actions/workflows/collect-status.yml/dispatches',
    authorization: 'Bearer test-only-token',
  })))
})
