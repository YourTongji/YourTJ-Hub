import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium, webkit } from 'playwright'
import { createServer } from 'vite'

let server, origin
const browsers = {}
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false }, logLevel: 'error' })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browsers.chromium = await chromium.launch()
  browsers.webkit = await webkit.launch()
})
after(async () => { await Promise.all(Object.values(browsers).map(browser => browser.close())); await server?.close() })

function fixture(path) {
  if (/namespaces$|\/tree$|\/runs$|migrate-tasks$|sync-status$/.test(path)) return []
  if (path.endsWith('/sponsors')) return { sponsors: [] }
  if (path.includes('feed')) return { enabled: true, rankingReady: true, paramsHash: 'fixture', rows: [], periods: [], health: {} }
  return {}
}

for (const engine of ['chromium', 'webkit']) for (const kind of ['feed', 'wiki', 'sponsors', 'security', 'onesystem', 'storage']) for (const lang of ['zh', 'en', 'ja', 'de']) for (const theme of ['gf-light', 'gf-dark']) {
  test(`${engine}: ${kind} translated controls fit 320px ${lang}/${theme}`, async () => {
    const page = await browsers[engine].newPage({ viewport: { width: 320, height: 900 } })
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    try {
      await page.route('**/api/**', route => route.fulfill({ json: { code: 0, result: fixture(new URL(route.request().url()).pathname) } }))
      await page.goto(`${origin}/assets/test/fixtures/browser/admin-translated-layout.html?kind=${kind}&lang=${lang}&theme=${theme}`)
      await page.locator('#app[data-ready="true"] h1').waitFor()
      await page.waitForFunction(() => !document.querySelector('svg.animate-spin'))
      await page.evaluate(() => document.fonts.ready)
      assert.deepEqual(errors, [])
      const size = await page.evaluate(() => ({ width: innerWidth, scroll: document.documentElement.scrollWidth }))
      assert.ok(size.scroll <= size.width + 1, JSON.stringify(size))
      if (kind === 'wiki') {
        const tabs = page.getByRole('tab')
        for (let index = 0; index < await tabs.count(); index++) {
          assert.ok(await tabs.nth(index).isVisible())
          const box = await tabs.nth(index).boundingBox()
          assert.ok(box.x >= 0 && box.x + box.width <= 321)
        }
        await tabs.first().focus(); await page.keyboard.press('ArrowRight')
        assert.equal(await tabs.nth(1).getAttribute('data-state'), 'active')
      }
    } finally { await page.close() }
  })
}
