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
  test(`rejected topic gallery can be removed before saving a draft (${theme})`, async () => {
    const page = await browser.newPage({ viewport: { width: 390, height: 844 } })
    try {
      let submitted
      await page.route('**/api/forum/topics/write', async route => {
        submitted = route.request().postDataJSON()
        await route.fulfill({ json: { code: 1, messageCode: 'common.operation.failed' } })
      })
      await page.goto(`${origin}/assets/test/fixtures/browser/publish-images.html?theme=${theme}`)
      const gallery = page.locator('[data-test="editable-gallery"]')
      await gallery.waitFor()
      await gallery.scrollIntoViewIfNeeded()
      await page.locator('.vditor [contenteditable="true"]:visible').first().waitFor()
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
      if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
        await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
        await page.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/publish-images-${theme}.png` })
      }
      await page.getByRole('button', { name: '删除图片', exact: true }).click()
      await gallery.waitFor({ state: 'detached' })
      await page.getByRole('button', { name: '保存草稿', exact: true }).click()
      await page.getByText('操作失败', { exact: true }).waitFor()
      assert.equal(submitted.topicStatus, 0)
      assert.deepEqual(submitted.images, [])
      assert.match(submitted.content, /重新提交审核/)
    } finally { await page.close() }
  })
}
