import { afterAll, beforeAll, expect, it } from 'vitest'
import { createServer, type Server } from 'node:http'
import { readFileSync } from 'node:fs'
import { transpileModule, ModuleKind, ScriptTarget } from 'typescript'
import { Miniflare, convertV4MiniflareOptions } from 'miniflare'

let server: Server, mf: Miniflare
const calls: string[] = []
beforeAll(async () => {
  server = createServer((req, res) => {
    calls.push(req.url!)
    if (req.url === '/redirect') { res.writeHead(302, { Location: '/private' }); res.end(); return }
    res.setHeader('Content-Type', 'application/json')
    res.end(JSON.stringify({ ok: true }))
  })
  await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve))
  const port = (server.address() as { port: number }).port
  const code = transpileModule(readFileSync('server/http.ts', 'utf8'), {
    compilerOptions: { module: ModuleKind.ESNext, target: ScriptTarget.ES2022 },
  }).outputText
  mf = new Miniflare(convertV4MiniflareOptions({ modules: true, script: `${code}
    export default { async fetch(request) {
      try {
        const result = await json(fetch, 'http://127.0.0.1:${port}' + new URL(request.url).pathname,
          AbortSignal.timeout(3000), { method: 'POST', headers: { Authorization: 'Bearer test-only' }, body: '{}' });
        return Response.json(result)
      } catch { return new Response('Source unavailable', { status: 502 }) }
    } }
  ` }))
})
afterAll(async () => { await mf?.dispose(); await new Promise<void>(resolve => server?.close(() => resolve())) })

it('fetches JSON in the real Workers runtime and never follows credential-bearing redirects', async () => {
  expect(await (await mf.dispatchFetch('http://worker/ok')).json()).toEqual({ ok: true })
  expect((await mf.dispatchFetch('http://worker/redirect')).status).toBe(502)
  expect(calls).toEqual(['/ok', '/redirect'])
})
