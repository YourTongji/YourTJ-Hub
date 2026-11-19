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

// 1×1 PNG
const png = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==', 'base64')
const markdown = '# 导入的文章\n\n正文\n\n![流程图](img/a.png)\n\n![截图](../assets/b.png)\n\n![缺失的图](missing/c.png)\n'

for (const theme of ['gf-light', 'gf-dark']) {
  test(`markdown import matches local images and offers rollback actions (${theme})`, async () => {
    const page = await browser.newPage({ viewport: { width: 390, height: 844 } })
    const errors = []
    page.on('pageerror', (err) => errors.push(err.message))
    try {
      let uploads = 0
      await page.route('**/file/img-upload/init', (route) => route.fulfill({ json: { code: 0, result: { mode: 'proxy' } } }))
      await page.route('**/file/img-upload', (route) => route.fulfill({ json: { code: 0, result: { url: `/assets/test/fixtures/browser/topic-feed-art.svg?u=${++uploads}` } } }))
      await page.goto(`${origin}/assets/test/fixtures/browser/markdown-import.html?theme=${theme}`)
      await page.locator('[data-test="import-dropzone"]').waitFor()
      await page.locator('input[type="file"]').first().setInputFiles([
        { name: 'post.md', mimeType: 'text/markdown', buffer: Buffer.from(markdown) },
        { name: 'a.png', mimeType: 'image/png', buffer: png },
        { name: 'b.png', mimeType: 'image/png', buffer: png },
      ])
      const missing = page.locator('[data-test="import-missing-local"]')
      await missing.waitFor()
      assert.equal(uploads, 2)
      const rows = page.locator('[data-test="import-image"]')
      assert.equal(await rows.count(), 3)
      assert.equal(await page.getByRole('button', { name: '替换', exact: true }).count(), 2)
      const surface = page.locator('.gf-menu-surface')
      assert.ok(await surface.evaluate((el) => [...el.querySelectorAll('*')].every((child) => child.scrollWidth <= child.clientWidth + 1 || getComputedStyle(child).overflowX !== 'visible' || child.getBoundingClientRect().right <= el.getBoundingClientRect().right + 1)))
      if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
        await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
        await page.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/markdown-import-${theme}.png` })
      }
      await rows.first().getByRole('button', { name: '不使用这张图片' }).click()
      await rows.first().getByText('不使用，导入后只保留说明文字').waitFor()
      await rows.first().getByRole('button', { name: '恢复', exact: true }).click()
      await rows.first().getByRole('button', { name: '替换', exact: true }).waitFor()
      assert.equal(uploads, 2)
      assert.deepEqual(errors, [])
    } finally { await page.close() }
  })
}
