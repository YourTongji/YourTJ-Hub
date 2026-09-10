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

for (const lang of ['zh', 'en', 'ja', 'de']) {
  for (const width of [320, 1024]) {
    test(`materialization controls and report fit ${lang} ${width}px`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 800 } })
      try {
        await page.route('**/api/admin/pk/materialize-calendar', route => {
          assert.deepEqual(route.request().postDataJSON(), { term: '122', audience: 'undergraduate' })
          return route.fulfill({ json: { code: 0, result: { calendarId: 122, coursesInserted: 1, coursesUpdated: 3, instructorsInserted: 1, aliasesInserted: 2, aliasesSkipped: 0, offeringsInserted: 1, offeringsUpdated: 3 } } })
        })
        await page.goto(`${origin}/assets/test/fixtures/browser/course-materialize.html?lang=${lang}`)
        await page.waitForSelector('html[data-ready="true"]')
        await page.locator('button').click()
        await page.locator('[role="status"]').waitFor()
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'page must not overflow horizontally')
        const button = await page.locator('button').boundingBox()
        assert.ok(button && button.x >= 0 && button.x + button.width <= width, 'action remains in viewport')
        if (process.env.COURSE_MATERIALIZE_SCREENSHOT && lang === 'zh' && width === 1024) {
          await page.screenshot({ path: process.env.COURSE_MATERIALIZE_SCREENSHOT, fullPage: true })
        }
      } finally { await page.close() }
    })
  }
}
