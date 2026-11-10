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

for (const width of [375, 1280]) for (const theme of ['gf-light', 'gf-dark']) {
  test(`feedback remains above the open export dialog: ${width}px ${theme}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    try {
      await page.goto(`${origin}/assets/test/fixtures/browser/schedule-export-feedback.html`)
      await page.locator('html[data-ready="true"]').waitFor()
      await page.evaluate(theme => { document.documentElement.dataset.theme = theme }, theme)
      await page.getByRole('dialog').waitFor()
      await page.evaluate(async () => {
        const { useFlashMessages } = await import('/assets/src/runtime/flash-message.ts')
        useFlashMessages().push('导出反馈可见性回归', 'error')
      })
      const banner = page.locator('.gf-flash-banner')
      await banner.waitFor()
      assert.equal(await page.getByRole('alert').count(), 1, 'feedback remains accessible while the dialog is open')
      await page.waitForTimeout(300)
      assert.equal(await banner.evaluate(el => {
        const box = el.getBoundingClientRect()
        return el.contains(document.elementFromPoint(box.x + box.width / 2, box.y + box.height / 2))
      }), true, 'the dialog must not cover immediate feedback')
      if (process.env.YOURTJ_TEST_SCREENSHOT_DIR) {
        await mkdir(process.env.YOURTJ_TEST_SCREENSHOT_DIR, { recursive: true })
        await page.screenshot({ path: `${process.env.YOURTJ_TEST_SCREENSHOT_DIR}/export-feedback-${width}-${theme}.png`, fullPage: true })
      }
    } finally { await page.close() }
  })
}
