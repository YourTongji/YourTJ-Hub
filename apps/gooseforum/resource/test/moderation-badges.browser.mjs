import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { mkdir } from 'node:fs/promises'
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
for (const theme of ['gf-light', 'gf-dark']) {
  for (const width of [360, 1280]) {
    test(`pending badge preserves row and card density (${theme}, ${width})`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 900 } })
      try {
        await page.goto(`${origin}/assets/test/fixtures/browser/moderation-badges.html`)
        await page.getByText('审核中', { exact: true }).first().waitFor()
        await page.evaluate(value => { document.documentElement.dataset.theme = value }, theme)
        for (const surface of ['row', 'card']) {
          const normal = await page.locator(`[data-case="${surface}-0"]`).boundingBox()
          const pending = page.locator(`[data-case="${surface}-2"]`)
          const box = await pending.boundingBox()
          assert.equal(box.height, normal.height, `${surface}: badge should not add a separate line`)
          const badge = await pending.getByText('审核中', { exact: true }).boundingBox()
          assert.ok(badge.x >= box.x && badge.x + badge.width <= box.x + box.width)
          if (surface === 'row') assert.equal(await pending.locator('article').getByText('审核中', { exact: true }).count(), 1)
        }
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
        if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
          await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
          await page.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/moderation-badges-${theme}-${width}.png` })
        }
      } finally { await page.close() }
    })
  }
}
