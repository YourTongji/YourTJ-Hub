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
for (const theme of ['gf-light', 'gf-dark']) {
  test(`rejected reply retains input on failure, resubmits and reconciles approval (${theme})`, async () => {
    const page = await browser.newPage({ viewport: { width: 390, height: 844 } })
    page.setDefaultTimeout(5000)
    try {
      await page.addInitScript(() => {
        window.EventSource = class extends EventTarget {
          constructor() { super(); window.contentStream = this }
          close() {}
        }
      })
      let fail = true
      await page.route('**/api/forum/posts/update', async route => {
        const request = route.request().postDataJSON()
        assert.equal(request.postId, 42)
        assert.equal(request.content, '修改后的完整回复')
        await route.fulfill({ json: fail ? { code: 1, messageCode: 'common.operation.failed', message: '网络保存失败' } : { code: 0, messageCode: 'content.moderation.checking', result: { id: 42, postNo: 2 } } })
      })
      await page.route('**/fixture/content', route => route.fulfill({ json: { id: 42, contentType: 'post', title: '校园生活中的一次讨论', content: '修改后的完整回复', processStatus: 0, reviewReason: '', hasPublishedVersion: true, images: [] } }))
      await page.goto(`${origin}/assets/test/fixtures/browser/managed-content.html?theme=${theme}`)
      await page.getByText('请修改正文后重新提交。').waitFor()
      await page.getByRole('button', { name: '查看内容' }).click()
      await page.locator('strong', { hasText: '完整正文' }).waitFor()
      await page.getByRole('button', { name: '修改并重新提交' }).click()
      await page.getByRole('textbox').fill('修改后的完整回复')
      await page.getByRole('button', { name: '修改并重新提交' }).last().click()
      await page.getByText('操作失败', { exact: true }).waitFor()
      assert.equal(await page.getByRole('textbox').inputValue(), '修改后的完整回复')
      fail = false
      await page.getByRole('button', { name: '修改并重新提交' }).last().click()
      await page.getByText('审核中', { exact: true }).waitFor()
      assert.equal(await page.getByRole('textbox').count(), 0)
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
      if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
        await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
        await page.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/managed-content-${theme}.png` })
      }
      await page.evaluate(() => window.contentStream.dispatchEvent(new Event('content.changed')))
      await page.getByText('审核中', { exact: true }).waitFor({ state: 'hidden' })
      await page.getByText('修改后的完整回复', { exact: true }).waitFor()
    } finally { await page.close() }
  })
}
