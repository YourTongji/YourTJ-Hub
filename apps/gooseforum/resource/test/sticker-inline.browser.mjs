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

const square = '<svg xmlns="http://www.w3.org/2000/svg" width="128" height="128"><circle cx="64" cy="64" r="60" fill="#f5b400"/></svg>'

for (const theme of ['gf-light', 'gf-dark']) {
  test(`post stickers sit inline at 2.5em and open their own card (${theme})`, async () => {
    const page = await browser.newPage({ viewport: { width: 390, height: 760 } })
    const errors = []
    page.on('pageerror', (err) => errors.push(err.message))
    try {
      await page.route('**/sticker.png', (route) => route.fulfill({ contentType: 'image/svg+xml', body: square }))
      await page.route('**/photo.png', (route) => route.fulfill({ contentType: 'image/svg+xml', body: square }))
      await page.goto(`${origin}/assets/test/fixtures/browser/sticker-inline.html?theme=${theme}`)
      await page.locator('#sticker').waitFor()
      await page.waitForFunction(() => [...document.images].every((img) => img.complete))

      const metrics = await page.evaluate(() => {
        const rect = (id) => document.getElementById(id).getBoundingClientRect()
        const style = (id) => getComputedStyle(document.getElementById(id))
        const range = document.createRange()
        const text = document.getElementById('mixed').firstChild
        range.setStart(text, 0)
        range.setEnd(text, 2)
        const glyph = range.getBoundingClientRect()
        return {
          fontSize: parseFloat(style('mixed').fontSize),
          headingFontSize: parseFloat(style('heading').fontSize),
          plainHeight: rect('plain').height,
          mixedHeight: rect('mixed').height,
          sticker: rect('sticker').toJSON(),
          sticker2: rect('sticker-2').toJSON(),
          heading: rect('sticker-heading').toJSON(),
          glyphCenter: (glyph.top + glyph.bottom) / 2,
          mixedAlign: style('mixed').textAlign,
          aloneAlign: style('alone').textAlign,
          border: style('sticker').borderTopWidth,
          photoDisplay: style('photo').display,
        }
      })
      assert.ok(Math.abs(metrics.sticker.height - metrics.fontSize * 2.5) < 0.5, `sticker height ${metrics.sticker.height}`)
      assert.ok(Math.abs(metrics.heading.height - metrics.headingFontSize * 2.5) < 0.5, 'heading sticker follows heading size')
      assert.equal(metrics.sticker.top, metrics.sticker2.top, 'stickers share the text line')
      assert.ok(Math.abs(metrics.sticker.top + metrics.sticker.height / 2 - metrics.glyphCenter) < 2.5, 'sticker centred on the glyphs')
      assert.notEqual(metrics.mixedAlign, 'center')
      assert.notEqual(metrics.aloneAlign, 'center')
      assert.equal(metrics.border, '0px')
      assert.equal(metrics.photoDisplay, 'block', 'ordinary photos keep the block layout')

      await page.locator('#sticker').click()
      const card = page.locator('[data-test="sticker-card"]')
      await card.waitFor()
      assert.equal(await page.locator('.gf-lightbox, [data-test="markdown-image-viewer"]').count(), 0)
      // Measure after the short zoom-in entrance settles.
      await page.waitForTimeout(250)
      const stage = await card.locator('img').boundingBox()
      assert.equal(Math.round(stage.width), 160)
      assert.match(await card.innerText(), /smile/)
      assert.match(await card.locator('[data-test="sticker-card-save"]').innerText(), /登录后收藏/)
      await page.keyboard.press('Escape')
      await card.waitFor({ state: 'detached' })
      assert.deepEqual(errors, [])
    } finally {
      await page.close()
    }
  })
}
