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
  test(`managed moment opens and submits as a moment (${theme})`, async () => {
    const page = await browser.newPage({ viewport: { width: 390, height: 844 } })
    try {
      let submitted
      await page.route('**/api/forum/topics/write', async route => {
        submitted = route.request().postDataJSON()
        await route.fulfill({ json: { code: 1, messageCode: 'common.operation.failed' } })
      })
      await page.goto(`${origin}/assets/test/fixtures/browser/publish-images.html?type=2&theme=${theme}`)
      await page.getByText('编辑瞬间', { exact: true }).waitFor()
      await page.locator('.vditor [contenteditable="true"]:visible').first().waitFor()
      await page.getByRole('button', { name: '更新内容', exact: true }).click()
      await page.getByText('操作失败', { exact: true }).waitFor()
      assert.equal(submitted.topicId, 17)
      assert.equal(submitted.contentType, 2)
      assert.equal(submitted.topicStatus, 1)
    } finally { await page.close() }
  })
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

for (const lang of ['zh', 'en', 'ja', 'de']) for (const theme of ['gf-light', 'gf-dark']) {
  test(`article toolbar and counter remain reachable at 320px (${lang}/${theme})`, async () => {
    const page = await browser.newPage({ viewport: { width: 320, height: 900 } })
    try {
      await page.goto(`${origin}/assets/test/fixtures/browser/publish-images.html?type=3&lang=${lang}&theme=${theme}`)
      const body = page.locator('.vditor [contenteditable="true"]:visible').first()
      await body.waitFor()
      const more = page.locator('.vditor-toolbar > .vditor-toolbar__item > [data-type="more"]')
      await more.waitFor()
      const within = async () => {
        await page.waitForTimeout(250)
        const bounds = await page.evaluate(() => {
          const root = document.querySelector('.vditor-toolbar').getBoundingClientRect()
          const items = [...document.querySelectorAll('.vditor-toolbar > .vditor-toolbar__item, .vditor-toolbar > .vditor-counter')]
          return { width: innerWidth, scroll: document.documentElement.scrollWidth,
            clipped: items.filter(e => e.getBoundingClientRect().right > root.right + 1).length }
        })
        assert.ok(bounds.scroll <= bounds.width, `document must fit: ${JSON.stringify(bounds)}`)
        assert.equal(bounds.clipped, 0, 'every main-row action and counter fits the toolbar')
      }
      await within()
      await body.click()
      await page.keyboard.press('ControlOrMeta+A')
      await page.keyboard.insertText('A'.repeat(1000))
      await page.keyboard.press('Space')
      await page.locator('.vditor-counter').filter({ hasText: '1001' }).waitFor()
      await within()
      await more.click()
      const foldedLink = page.locator('.hub-fold-region [data-type="link"]')
      await foldedLink.waitFor({ state: 'visible' })
      assert.equal(await foldedLink.locator('.vditor-toolbar-label').count(), 1,
        'folded icons gain visible translated menu labels')
      if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
        await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
        await page.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/article-more-${lang}-${theme}.png` })
      }
      await foldedLink.click()
      await page.locator('.vditor .vditor-input:visible').first().waitFor()
      const fields = await page.locator('.vditor .vditor-input:visible').evaluateAll(inputs => inputs.map(input => {
        const box = input.getBoundingClientRect()
        return { left: box.left, right: box.right, width: innerWidth }
      }))
      assert.ok(fields.every(box => box.left >= 0 && box.right <= box.width),
        `link fields remain fully visible: ${JSON.stringify(fields)}`)
      if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
        await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
        await page.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/article-toolbar-${lang}-${theme}.png` })
      }
    } finally { await page.close() }
  })
}
