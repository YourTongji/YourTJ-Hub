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
// A doubled root text size stresses 200% text magnification at the narrowest
// supported CSS viewport, alongside reduced-height soft-keyboard layouts.
for (const [width, height, fontSize] of [[320, 640, 16], [320, 640, 32], [375, 667, 16], [390, 560, 16], [640, 720, 16]]) {
  test(`quick publisher mention candidates remain tappable at ${width}x${height}, ${fontSize}px text`, async () => {
    const page = await browser.newPage({ viewport: { width, height } })
    try {
      await page.route('**/api/forum/search?*', route => route.fulfill({ json: { code: 0, result: { users: Array.from({ length: 8 }, (_, i) => ({ id: i + 2, username: `tester${i}`, nickname: `测试用户${i}`, avatarUrl: '' })) } } }))
      await page.goto(`${origin}/assets/test/fixtures/browser/quick-publish.html`)
      await page.evaluate(size => { document.documentElement.style.fontSize = `${size}px` }, fontSize)
      const editor = page.locator('.vditor [contenteditable="true"]:visible').first()
      await editor.waitFor()
      await editor.click()
      await page.keyboard.type('@test')
      const last = page.locator('.gf-mention-option').last()
      await last.waitFor()
      const panelBox = await page.locator('.gf-mention-panel').boundingBox()
      assert.ok(panelBox && panelBox.y >= 0 && panelBox.y + panelBox.height <= height, `panel must fit before any scrolling: ${JSON.stringify(panelBox)}`)
      assert.ok(panelBox.x >= 0 && panelBox.x + panelBox.width <= width, 'candidate panel must fit horizontally')
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'page must not overflow horizontally')
      await last.scrollIntoViewIfNeeded()
      const box = await last.boundingBox()
      assert.ok(box && box.y >= 0 && box.y + box.height <= height, 'last candidate must be in viewport')
      assert.ok(await last.evaluate(el => {
        const r = el.getBoundingClientRect()
        return el.contains(document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2))
      }), 'last candidate must not be clipped or covered')
      await last.click({ timeout: 3000 })
      await page.waitForFunction(() => document.querySelector('.vditor [contenteditable="true"]')?.textContent.includes('@tester7'))
      await page.locator('.gf-mention-panel').waitFor({ state: 'detached' })
    } finally { await page.close() }
  })
}
