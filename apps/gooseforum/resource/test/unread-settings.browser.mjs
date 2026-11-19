import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdir } from 'node:fs/promises'
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
for (const locale of ['zh', 'de']) for (const width of [375, 1280]) test(`badge categories ${locale} ${width}`, async () => {
  const page = await browser.newPage({ viewport: { width, height: 844 } })
  let categories = ['replies', 'mentions', 'likes', 'following', 'system']
  let badgeStyle = 'number'
  const writes = []
  try {
    await page.route('**/api/forum/notification-preferences', route => {
      const body = route.request().postDataJSON()
      writes.push(body)
      categories = body.badgeCategories ?? categories
      badgeStyle = body.badgeStyle ?? badgeStyle
      return route.fulfill({json: {code: 0, result: {badgeStyle, badgeCategories: categories}}})
    })
    await page.route('**/api/forum/unread-status', route => route.fulfill({json: {code: 0, result: {notifications: false, messages: false, notificationCount: 0, badgeStyle, badgeCategories: categories}}}))
    await page.goto(`${origin}/assets/test/fixtures/browser/unread-settings.html?locale=${locale}&theme=${width === 375 ? 'gf-dark' : 'gf-light'}`)
    const boxes = page.getByRole('checkbox')
    await boxes.first().waitFor()
    assert.equal(await boxes.count(), 5)
    for (let index = 4; index > 0; index--) {
      await boxes.nth(index).uncheck()
      await boxes.nth(index).waitFor({state: 'visible'})
      await page.waitForFunction(() => !document.querySelector('fieldset').disabled)
    }
    assert.deepEqual(categories, ['replies'])
    assert.deepEqual(writes.at(-1), {badgeCategories: ['replies']})
    await page.getByRole('switch').click()
    await page.waitForFunction(() => !document.querySelector('fieldset').disabled)
    assert.equal(badgeStyle, 'dot')
    assert.deepEqual(categories, ['replies'])
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
    if (process.env.YOURTJ_UNREAD_SCREENSHOTS) {
      await mkdir(process.env.YOURTJ_UNREAD_SCREENSHOTS, {recursive: true})
      await page.screenshot({path: `${process.env.YOURTJ_UNREAD_SCREENSHOTS}/${locale}-${width}.png`, fullPage: true})
    }
  } finally { await page.close() }
})
