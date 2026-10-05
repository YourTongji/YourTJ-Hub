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
for (const theme of ['gf-light', 'gf-dark']) for (const width of [390, 1280]) {
  test(`review notifications remain readable and link to recovery ${theme} ${width}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 844 } })
    try {
      await page.goto(`${origin}/assets/test/fixtures/browser/review-notifications.html?theme=${theme}`)
      await page.getByText('你的内容正在等待人工审核', { exact: true }).waitFor()
      await page.getByText('你的内容已通过审核', { exact: true }).waitFor()
      await page.getByText('你的内容未通过审核', { exact: true }).waitFor()
      const rejected = page.getByRole('link', { name: '校******论', exact: true })
      assert.equal(await rejected.getAttribute('href'), '/settings?tab=content')
      assert.equal(await page.getByRole('link', { name: '校园生活中的一次讨论', exact: true }).first().getAttribute('href'), '/p/post/42/1')
      const detail = page.getByText('它不会公开显示，请前往内容管理自查修改后重新提交。如有疑问，请联系管理员。', { exact: true })
      assert.ok(await detail.evaluate(el => el.scrollHeight <= el.clientHeight))
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
      if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
        await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
        await page.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/review-notifications-${theme}-${width}.png`, fullPage: true })
      }
    } finally { await page.close() }
  })
}
