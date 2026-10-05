import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdir } from 'node:fs/promises'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), optimizeDeps: { entries: ['test/fixtures/browser/managed-content.html'] }, server: { host: '127.0.0.1', port: 0, open: false } })
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
        assert.match(request.content, /修改后的完整回复/)
        assert.doesNotMatch(request.content, /被拒的修改|完整正文/)
        assert.ok(request.content.includes('/file/replacement.png'))
        await route.fulfill({ json: fail ? { code: 1, messageCode: 'common.operation.failed', message: '网络保存失败' } : { code: 0, messageCode: 'content.moderation.checking', result: { id: 42, postNo: 2 } } })
      })
      const imagePath = fileURLToPath(new URL('../static/pic/icons/favicon-32.png', import.meta.url))
      await page.route('**/file/img-upload/init', route => route.fulfill({ json: { code: 0, result: { mode: 'proxy' } } }))
      await page.route('**/file/img-upload', route => route.fulfill({ json: { code: 0, result: { url: '/file/replacement.png' } } }))
      await page.route('**/file/replacement.png', route => route.fulfill({ path: imagePath, contentType: 'image/png' }))
      await page.route('**/fixture/content', route => route.fulfill({ json: { id: 42, contentType: 'post', title: '校园生活中的一次讨论', content: '修改后的完整回复', processStatus: 0, reviewReason: '', hasPublishedVersion: true, images: [] } }))
      await page.goto(`${origin}/assets/test/fixtures/browser/managed-content.html?theme=${theme}`)
      await page.getByText('请修改正文后重新提交。').waitFor()
      await page.getByRole('button', { name: '查看内容' }).click()
      await page.locator('strong', { hasText: '完整正文' }).waitFor()
      await page.getByRole('button', { name: '修改并重新提交' }).click()
      const dialog = page.getByRole('dialog', { name: '编辑自己的回复' })
      const editor = dialog.locator('.vditor [contenteditable="true"]:visible').first()
      await editor.waitFor()
      assert.ok((await editor.innerText()).includes('完整正文'))
      assert.equal(await dialog.locator('.vditor--dark').count(), theme === 'gf-dark' ? 1 : 0)
      assert.equal(await dialog.locator('[data-type="upload"]').count(), 1)
      await editor.fill('修改后的完整回复')
      await dialog.getByRole('button', { name: '关闭', exact: true }).click()
      await dialog.waitFor({ state: 'hidden' })
      await page.getByRole('button', { name: '修改并重新提交' }).click()
      await editor.waitFor()
      assert.equal((await editor.innerText()).trim(), '修改后的完整回复')
      await dialog.locator('input[type="file"]').setInputFiles(imagePath)
      await editor.locator('img[src="/file/replacement.png"]').waitFor()
      await dialog.getByRole('button', { name: '保存', exact: true }).click()
      await page.getByText('操作失败', { exact: true }).waitFor()
      assert.equal((await editor.innerText()).trim(), '修改后的完整回复')
      if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
        await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
        await page.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/reply-editor-${theme}.png` })
      }
      fail = false
      await dialog.getByRole('button', { name: '保存', exact: true }).click()
      await page.getByText('审核中', { exact: true }).waitFor()
      await dialog.waitFor({ state: 'hidden' })
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

for (const theme of ['gf-light', 'gf-dark']) {
  for (const contentType of [1, 2]) {
    test(`content manager opens and submits short editor type ${contentType} (${theme})`, async () => {
      const page = await browser.newPage({ viewport: { width: 390, height: 844 } })
      page.setDefaultTimeout(10000)
      try {
        await page.route('**/publish?id=42', route => route.fulfill({ json: {
          version: '1.0', component: 'publish.index', layout: {}, meta: {},
          props: { topicId: 42, isEditing: true, categories: [], topic: { contentType, title: '校园生活中的一次讨论', content: '这是等待审核的新版本内容', categoryIds: [101], images: [], topicStatus: 1 } },
        } }))
        let submitted
        await page.route('**/api/forum/topics/write', route => {
          submitted = route.request().postDataJSON()
          return route.fulfill({ json: { code: 1, messageCode: 'common.operation.failed' } })
        })
        await page.goto(`${origin}/assets/test/fixtures/browser/managed-content.html?theme=${theme}&topic=1`)
        await page.getByRole('button', { name: '修改并重新提交' }).click()
        const dialog = page.getByRole('dialog')
        await dialog.waitFor()
        const editor = dialog.locator('.vditor [contenteditable="true"]:visible').first()
        await editor.waitFor()
        assert.ok((await editor.innerText()).includes('这是等待审核的新版本内容'))
        assert.equal(await dialog.locator('.vditor--dark').count(), theme === 'gf-dark' ? 1 : 0)
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
        if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
          await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
          await page.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/short-editor-${contentType}-${theme}.png` })
        }
        await dialog.getByRole('button', { name: '保存', exact: true }).click()
        await page.getByText('操作失败', { exact: true }).waitFor()
        assert.equal(submitted.topicId, 42)
        assert.equal(submitted.contentType, contentType)
        assert.ok(submitted.content.includes('这是等待审核的新版本内容'))
        assert.deepEqual(submitted.categoryId, [101])
      } finally { await page.close() }
    })
  }
}
