import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
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

for (const viewport of [{ width: 1280, height: 800 }, { width: 375, height: 700 }]) {
  test(`课程更多筛选下拉越过卡片边缘仍完整可见且可选 (${viewport.width}px)`, async () => {
    const page = await browser.newPage({ viewport })
    try {
      await page.goto(`${origin}/assets/test/fixtures/browser/course-catalog-multi-select.html`)
      await page.waitForFunction(() => document.documentElement.dataset.ready === 'true')
      const card = page.locator('.gf-card')
      const radius = await card.evaluate((el) => getComputedStyle(el).borderRadius)
      assert.notEqual(radius, '0px', '卡片继续保留圆角')
      assert.equal(await card.evaluate((el) => getComputedStyle(el).overflow), 'hidden')

      const trigger = page.getByRole('button', { name: '全部院系' })
      await trigger.click()
      const panel = page.getByRole('group')
      await panel.waitFor({ state: 'visible' })
      const geometry = await page.evaluate(() => {
        const card = document.querySelector('.gf-card').getBoundingClientRect()
        const panel = document.querySelector('[role="group"]').getBoundingClientRect()
        const trigger = document.querySelector('button').getBoundingClientRect()
        const hit = document.elementFromPoint(panel.left + 24, panel.bottom - 16)
        return {
          cardBottom: card.bottom,
          panel: panel.toJSON(),
          trigger: trigger.toJSON(),
          portaled: document.querySelector('[role="group"]').parentElement === document.body,
          hitOption: Boolean(hit?.closest('label')),
          viewportWidth: innerWidth,
          viewportHeight: innerHeight,
        }
      })
      assert.ok(geometry.panel.bottom > geometry.cardBottom, '下拉应跨出卡片边缘以验证裁切场景')
      assert.ok(geometry.portaled, '下拉脱离裁切卡片')
      assert.ok(geometry.panel.left >= 0 && geometry.panel.right <= geometry.viewportWidth, '下拉完整位于视口宽度内')
      assert.ok(geometry.panel.top >= 0 && geometry.panel.bottom <= geometry.viewportHeight, '下拉完整位于视口高度内')
      assert.ok(geometry.hitOption, '卡片边缘之外的选项仍可命中')

      const option = page.getByLabel('计算机科学与技术')
      await option.click()
      assert.equal(await option.isChecked(), true)
      assert.deepEqual(await page.locator('#course-more-filters').evaluate((form) => [...new FormData(form).getAll('department')]), ['CS'])

      const beforeScroll = await page.evaluate(() => ({ panel: document.querySelector('[role="group"]').getBoundingClientRect().top, trigger: document.querySelector('button').getBoundingClientRect().top }))
      await page.evaluate(() => window.scrollBy(0, 24))
      await page.waitForFunction((before) => {
        const panel = document.querySelector('[role="group"]').getBoundingClientRect().top
        const trigger = document.querySelector('button').getBoundingClientRect().top
        return Math.abs((panel - trigger) - (before.panel - before.trigger)) < 1
      }, beforeScroll)
      assert.ok(await panel.evaluate((el) => {
        const rect = el.getBoundingClientRect()
        return rect.top >= 0 && rect.bottom <= innerHeight
      }), '页面滚动后下拉仍完整位于视口内')
    } finally {
      await page.close()
    }
  })
}
