import { fileURLToPath } from 'node:url'
import { createServer, loadEnv } from 'vite'

// Worktree-friendly local preview. Uses the real collectors and API with memory-only snapshots;
// it never connects to Netlify Blobs or writes to production. Secrets stay in this Node process.
const root = fileURLToPath(new URL('../', import.meta.url))
const data = new Map()
let version = 0
const store = {
  async read(key) { return structuredClone(data.get(key) ?? null) },
  async write(key, value, etag) {
    if (data.get(key)?.etag !== etag) return false
    data.set(key, { value: structuredClone(value), etag: String(++version) })
    return true
  },
}
let handler, config
const server = await createServer({
  root,
  server: { host: '127.0.0.1', port: 5247, strictPort: true },
  plugins: [{ name: 'local-status-snapshots', configureServer(vite) {
    vite.middlewares.use(async (req, res, next) => {
      if (new URL(req.url, 'http://localhost').pathname !== '/api/status') return next()
      try {
        const response = await handler(new Request(new URL(req.url, 'http://localhost'), { method: req.method }), store, config)
        res.writeHead(response.status, Object.fromEntries(response.headers))
        res.end(await response.text())
      } catch { res.writeHead(503); res.end('Local snapshot unavailable') }
    })
  } }],
})
const { loadConfig } = await server.ssrLoadModule('/server/config.ts')
const { collect } = await server.ssrLoadModule('/server/collect.ts')
handler = (await server.ssrLoadModule('/server/snapshots.ts')).serveSnapshot
config = loadConfig({ ...loadEnv('development', root, ''), ...process.env })
await server.listen()
server.printUrls()
const inFlight = new Set()
async function refresh(kind) {
  if (inFlight.has(kind)) return
  inFlight.add(kind)
  try {
    await collect(kind, store, config)
    console.log(`Local ${kind} collection completed.`)
  } catch { console.warn(`Local ${kind} collection could not save all snapshots.`) }
  finally { inFlight.delete(kind) }
}
void refresh('current')
void refresh('history')
const timers = [setInterval(() => void refresh('current'), 60_000), setInterval(() => void refresh('history'), 300_000)]
for (const signal of ['SIGINT', 'SIGTERM']) process.once(signal, async () => {
  timers.forEach(clearInterval)
  await server.close()
  process.exit(0)
})
