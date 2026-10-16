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

for (const locale of ['zh', 'de']) for (const width of [375, 1280]) {
  test(`category and unread controls combine without overflow ${locale} ${width}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 844 } })
    const requests = []
    try {
      await page.route('**/api/forum/notifications?**', route => {
        requests.push(new URL(route.request().url()).searchParams)
        return route.fulfill({status:200,contentType:'application/json',body:JSON.stringify({code:0,result:{items:[],nextCursor:0,hasNext:false,unreadCount:4}})})
      })
      await page.route('**/api/forum/unread-status', route => route.fulfill({status:200,contentType:'application/json',body:JSON.stringify({code:0,result:{notifications:true,messages:false}})}))
      await page.goto(`${origin}/assets/test/fixtures/browser/review-notifications.html?taxonomy=1&locale=${locale}&theme=${width===375?'gf-dark':'gf-light'}`)
      const mention = page.getByRole('button', {name:locale==='zh'?'@我':'Erwähnungen', exact:true})
      await mention.waitFor()
      // 通知行由 #1157 的 actor 装饰昵称与内容链接分列渲染（不再是单个整句链接），
      // 断言含 Alice 与 Campus 的通知行整体可见。
      const mentionRow = page.locator('article').filter({ hasText: 'Alice' }).filter({ hasText: 'Campus' }).first()
      assert.ok(await mentionRow.isVisible())
      assert.equal(await page.getByRole('button', {pressed:true}).count(), 1)
      if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
        await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, {recursive:true})
        await page.screenshot({path:`${process.env.YOURTJ_REVIEW_SCREENSHOTS}/taxonomy-${locale}-${width}.png`,fullPage:true})
      }
      await mention.click()
      await page.waitForFunction(() => !document.querySelector('article'))
      assert.equal(requests.at(-1).get('category'), 'mentions')
      assert.equal(requests.at(-1).get('filter'), 'all')
      await Promise.all([page.waitForResponse(response => response.url().includes('filter=unread')), page.getByRole('checkbox').check()])
      assert.equal(requests.at(-1).get('category'), 'mentions')
      assert.equal(requests.at(-1).get('filter'), 'unread')
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
    } finally { await page.close() }
  })
}
