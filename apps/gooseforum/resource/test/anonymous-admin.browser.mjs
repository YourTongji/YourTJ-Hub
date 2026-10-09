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
  for (const lang of ['zh', 'en', 'ja', 'de']) {
    test(`anonymous admin links stay safe and usable at ${width}px (${lang})`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 900 } })
      try {
        await page.goto(`${origin}/assets/test/fixtures/browser/anonymous-admin.html?lang=${lang}`)
        await page.locator('#anonymous-view-reason').fill('browser fixture check')
        await page.locator('form button[type="submit"]').click()
        const personaLink = page.locator(
          'a[href="/a/demo"][target="_blank"][rel="noopener noreferrer"]:visible',
        )
        const ownerLink = page.locator(
          'a[href="/u/101"][target="_blank"][rel="noopener noreferrer"]:visible',
        )
        await personaLink.first().waitFor()
        await ownerLink.waitFor()
        assert.equal(await page.locator('a[href="/u/105"]:visible').count(), 0)
        assert.ok(
          await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
          'admin view must not overflow horizontally',
        )
      } finally {
        await page.close()
      }
    })
  }
}
