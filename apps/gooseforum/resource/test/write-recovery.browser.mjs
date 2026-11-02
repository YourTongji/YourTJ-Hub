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
const labels = { zh: '更新内容', en: 'Update topic', ja: 'トピックを更新', de: 'Thema aktualisieren' }
for (const lang of ['zh', 'en', 'ja', 'de']) for (const width of [320, 1280]) {
 for (const code of ['permission.emailRequired', 'auth.required', 'permission.userFrozen']) {
  test(`write gate round trip: ${lang} ${width} ${code}`, async () => {
    const context = await browser.newContext({ viewport: { width, height: 900 } })
    const page = await context.newPage()
    let submissions = 0
    try {
      await context.route('**/api/forum/topics/write', async route => {
        submissions++
        await route.fulfill({ status: code === 'auth.required' ? 401 : 403,
          json: { code: 1, messageCode: code, params: { redirect: 'https://invalid.example' } } })
      })
      await context.route('**/settings?tab=profile', route => route.fulfill({ contentType: 'text/html', body: '<h1>Account security fixture</h1>' }))
      await context.route('**/login?redirect=*', route => route.fulfill({ contentType: 'text/html', body: '<h1>Sign in fixture</h1>' }))
      await page.goto(`${origin}/assets/test/fixtures/browser/publish-images.html?type=2&theme=${width === 320 ? 'gf-dark' : 'gf-light'}&lang=${lang}`)
      const editor = page.locator('.vditor [contenteditable="true"]:visible').first()
      await editor.waitFor()
      await editor.fill(`Unsent original text ${lang}`)
      // The real editor reports input asynchronously; submit also reads its live value.
      const baselineWidth = await page.evaluate(() => document.documentElement.scrollWidth)
      await page.getByRole('button', { name: labels[lang], exact: true }).click()
      await page.locator('.gf-status-message-error').waitFor()
      assert.equal(submissions, 1)
      assert.match(await editor.innerText(), /Unsent original text/)
      const recovery = page.locator('[data-write-recovery]')
      if (code === 'permission.userFrozen') {
        assert.equal(await recovery.count(), 0)
      } else {
        await recovery.waitFor()
        const action = recovery.getByRole('link')
        await action.scrollIntoViewIfNeeded()
        const box = await action.boundingBox()
        assert.ok(box.height >= 44 && box.x >= 0 && box.x + box.width <= width)
        assert.ok((await page.evaluate(() => document.documentElement.scrollWidth)) <= Math.max(width, baselineWidth), `recovery increased pre-existing page width ${baselineWidth}`)
        const href = await action.getAttribute('href')
        assert.ok(!href.includes('invalid.example'))
        if (code === 'auth.required') {
          assert.equal(new URL(href, origin).searchParams.get('redirect'), new URL(page.url()).pathname + new URL(page.url()).search)
        } else assert.equal(href, '/settings?tab=profile')
        if (process.env.YOURTJ_REVIEW_SCREENSHOTS && code === 'permission.emailRequired') {
          await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
          await page.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/write-recovery-${lang}-${width}.png` })
        }
        const opened = context.waitForEvent('page')
        await action.click()
        const destination = await opened
        await destination.waitForLoadState('domcontentloaded')
        assert.equal(await destination.locator('h1').count(), 1)
        await destination.close()
        assert.match(await editor.innerText(), /Unsent original text/)
        assert.equal(submissions, 1)
      }
    } finally { await context.close() }
  })
 }
}
