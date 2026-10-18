import assert from 'node:assert/strict'
import { before, after, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdir } from 'node:fs/promises'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), optimizeDeps: { entries: ['test/fixtures/browser/moderation-appeals.html'] }, server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })
for (const [width, locale, theme] of [[360, 'en', 'gf-light'], [390, 'de', 'gf-dark'], [1280, 'en', 'gf-dark']]) {
  test(`appeal flow and preview retain content at ${width}px (${locale}, ${theme})`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    page.setDefaultTimeout(10000)
    const item = { id: 1, kind: 'post', targetId: 7, revisionId: 70, topicId: 9, postNo: 2, title: 'Candidate', content: 'Candidate '.repeat(30), images: ['/fixture/appeal.png'], reason: 'Please reconsider', originalReviewReason: 'Prior reason', status: 'pending', reviewReason: '', createdAt: '2026-10-10T00:00:00Z', reviewedAt: null }
    let submitted = false, reviewed = false
    try {
      await page.route('**/api/forum/content/appeals*', async route => {
        if (route.request().method() === 'POST') {
          assert.deepEqual(route.request().postDataJSON(), { kind: 'post', id: 7, revisionId: 70, reason: 'Please reconsider' })
          submitted = true
          return route.fulfill({ json: { code: 0, result: item } })
        }
        return route.fulfill({ json: { code: 0, result: { items: submitted ? [item] : [] } } })
      })
      await page.route('**/api/forum/moderation/appeals', route => route.fulfill({ json: { code: 0, result: { items: reviewed ? [] : [item], page: 1, pageSize: 20, total: reviewed ? 0 : 1 } } }))
      await page.route('**/api/forum/moderation/appeal-action', route => {
        assert.deepEqual(route.request().postDataJSON(), { id: 1, approve: false, reason: 'Still violates rules' })
        reviewed = true
        return route.fulfill({ json: { code: 0, result: true } })
      })
      await page.route('**/fixture/appeal.png', route => route.fulfill({ path: fileURLToPath(new URL('../static/pic/icons/favicon-32.png', import.meta.url)), contentType: 'image/png' }))
      await page.goto(`${origin}/assets/test/fixtures/browser/moderation-appeals.html?theme=${theme}&locale=${locale}`)
      const author = page.locator('[data-test="author"]')
      await author.locator('[data-test="appeal-toggle"]').click()
      // 等表单展开过渡结束（Vue 移除 grid-rows-* 过渡类）再操作：展开中按钮位移会让 pointerup 落在按钮外，提交被吞。
      await page.waitForFunction(() => {
        const wrapper = document.querySelector('[data-test="author"] [data-test="appeal-form"]')?.closest('.grid')
        return wrapper && !wrapper.className.includes('grid-rows-')
      })
      await author.locator('textarea').fill('Please reconsider')
      await author.locator('button[type="submit"]').click()
      await author.locator('textarea').waitFor({ state: 'hidden' })
      assert.ok(submitted)
      await author.locator('[data-test="appeal-state"]').waitFor()
      const moderator = page.locator('[data-test="moderator"]')
      await moderator.locator('textarea').fill('Still violates rules')
      const thumbnail = moderator.getByRole('button').filter({ has: page.locator('img') })
      await thumbnail.click()
      await page.getByRole('dialog').waitFor()
      await page.keyboard.press('Escape')
      await page.getByRole('dialog').waitFor({ state: 'hidden' })
      assert.equal(await moderator.locator('textarea').inputValue(), 'Still violates rules')
      assert.equal(await thumbnail.evaluate(el => el === document.activeElement), true)
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'no horizontal overflow')
      if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
        await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
        await page.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/appeals-${width}-${locale}-${theme}.png`, fullPage: true })
      }
      await moderator.locator('[data-test="appeal-reject"]').click()
      await moderator.locator('textarea').waitFor({ state: 'hidden' })
      assert.ok(reviewed)
    } finally { await page.close() }
  })
}
