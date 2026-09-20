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
for (const lang of ['zh', 'en', 'ja', 'de']) for (const width of [320, 1280]) {
  test(`Tongji login and registration are reachable in ${lang} at ${width}px`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 950 } })
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    try {
      await page.route('**/api/get-captcha*', route => route.fulfill({ json: { code: 0, result: { captchaId: '', captchaImg: '' } } }))
      for (const mode of ['login', 'register']) {
        await page.goto(`${origin}/assets/test/fixtures/browser/tongji-login.html?lang=${lang}&mode=${mode}`)
        const button = page.locator('[data-tongji-login]:visible')
        await button.waitFor()
        await button.scrollIntoViewIfNeeded()
        assert.equal(await button.count(), 1)
        const box = await button.boundingBox()
        assert.ok(box && box.x >= 0 && box.x + box.width <= width + 1 && box.height >= 40)
        assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
        assert.equal(await button.getAttribute('href'), `/api/auth/tongji?redirect=%2Fcampus&locale=${lang}`)
        assert.ok(await page.locator('a[href="/privacy"]:visible').count())
      }
      assert.deepEqual(errors, [])
    } finally { await page.close() }
  })
}
