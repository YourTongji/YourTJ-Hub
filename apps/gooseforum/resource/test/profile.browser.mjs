import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server
let browser
let origin

before(async () => {
  server = await createServer({
    root: fileURLToPath(new URL('../', import.meta.url)),
    server: { host: '127.0.0.1', port: 0, open: false },
  })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})

after(async () => {
  await browser?.close()
  await server?.close()
})

for (const width of [320, 1280]) {
  for (const hidden of [false, true]) {
    test(`anonymous profile remains usable when content is ${hidden ? 'hidden' : 'visible'} at ${width}px`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 900 } })
      try {
        await page.route('**/api/forum/unread-status', route => route.fulfill({
          json: { code: 0, result: { notifications: false, messages: false } },
        }))
        await page.goto(
          `${origin}/assets/test/fixtures/browser/profile.html?${hidden ? 'hidden=1' : ''}`,
        )
        await page.getByText('躲进云里的猫', { exact: true }).first().waitFor()
        assert.ok(
          await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
          'anonymous profile must not overflow horizontally',
        )
        if (hidden) {
          await page.locator('[data-test="anonymous-content-hidden"]:visible').waitFor()
          assert.equal(await page.locator('article a[href*="?tab="]').count(), 0)
          assert.equal(await page.getByText('分享一件最近让你开心的小事', { exact: true }).count(), 0)
        } else {
          await page.locator('article a[href$="?tab=topics"]').waitFor()
          await page.locator('article a[href$="?tab=replies"]').waitFor()
          await page.getByText('分享一件最近让你开心的小事', { exact: true }).waitFor()
        }
      } finally {
        await page.close()
      }
    })
  }
}
