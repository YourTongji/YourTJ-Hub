// issue #1164：列表/卡片切换必须复用已挂载的行节点，而不是整表卸载重建。
// happy-dom 的组件测试锁行为契约；这里在真实浏览器里确认 v-show 切换与隐藏树的属性裁剪。
import assert from 'node:assert/strict'
import { fileURLToPath } from 'node:url'
import { after, before, test } from 'node:test'
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

test('切换列表/卡片复用行节点，隐藏树不携带 data-feed-id', async () => {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
  try {
    await page.goto(`${origin}/assets/test/fixtures/browser/feed-switch.html?count=8`)
    await page.waitForFunction(() => document.documentElement.dataset.ready === 'true')
    // 空闲预建完成后两棵树都在，但只有列表可见
    await page.waitForFunction(() => document.querySelector('[data-feed-view="card"]'))
    assert.equal(await page.locator('[data-feed-view="card"]').evaluate((el) => getComputedStyle(el).display), 'none')
    assert.equal(await page.locator('[data-feed-id]').count(), 8)

    await page.evaluate(() => { window.__rows = [...document.querySelectorAll('[data-feed-view="table"] .gf-topic-row')] })
    await page.getByRole('button', { name: '卡片' }).click()
    await page.waitForFunction(() => getComputedStyle(document.querySelector('[data-feed-view="card"]')).display !== 'none')

    const reused = await page.evaluate(() => {
      const rows = [...document.querySelectorAll('[data-feed-view="table"] .gf-topic-row')]
      return rows.length === window.__rows.length && rows.every((row, index) => row === window.__rows[index]) && rows.every((row) => row.isConnected)
    })
    assert.ok(reused, 'switching must reuse the mounted table rows instead of rebuilding them')
    assert.equal(await page.locator('[data-feed-view="table"] .gf-topic-row [data-feed-id]').count(), 0)

    // 传导与推荐流锚点按文档顺序查询 [data-feed-id]：命中必须全部是可见卡片树里的行
    const tagged = await page.$$eval('[data-feed-id]', (rows) => rows.map((row) => Boolean(row.closest('[data-feed-view="card"]'))))
    assert.deepEqual(tagged, Array(8).fill(true))

    await page.getByRole('button', { name: '列表' }).click()
    await page.waitForFunction(() => getComputedStyle(document.querySelector('[data-feed-view="table"]')).display !== 'none')
    const cards = await page.$$eval('[data-feed-view="card"] .gf-card', (els) => els.length)
    assert.equal(cards, 8)
    assert.equal(await page.$$eval('[data-feed-view="card"] [data-feed-id]', (els) => els.length), 0)
    assert.equal(await page.locator('[data-feed-id]').count(), 8)
  } finally {
    await page.close()
  }
})
