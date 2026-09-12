import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })
for (const fontSize of [16, 32]) {
  test(`card links and actions remain reachable at 320px with ${fontSize}px root text`, async () => {
    const page = await browser.newPage({ viewport: { width: 320, height: 900 } })
    try {
      await page.route('**/api/forum/topics/like', route => route.fulfill({ json: { code: 0, result: true } }))
      await page.goto(`${origin}/assets/test/fixtures/browser/topic-card.html`)
      await page.locator('button[title="点赞"]').waitFor()
      await page.evaluate(size => { document.documentElement.style.fontSize = `${size}px` }, fontSize)
      const like = page.getByRole('button', { name: /^点赞/ })
      await like.click({ timeout: 3000 })
      assert.equal(await like.getAttribute('aria-pressed'), 'true')
      assert.equal(await like.textContent(), '6')
      assert.ok(await page.locator('a.gf-topic-chip').evaluate(el => {
        const r = el.getBoundingClientRect()
        return el.contains(document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2))
      }), 'category link must sit above the stretched card link')
      const comment = page.locator('a[href="/p/42?reply=1"]')
      await comment.click({ trial: true })
      await page.locator('a.gf-topic-chip').click({ trial: true })
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'card must not overflow the viewport')
    } finally { await page.close() }
  })
}
