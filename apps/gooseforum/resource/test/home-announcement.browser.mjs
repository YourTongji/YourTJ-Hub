import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

// issue #908: happy-dom 不渲染 hover；用真实组件与 CSS 检查着色范围。
let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })

for (const theme of ['gf-light', 'gf-dark']) {
  test(`${theme}: 空标题公告的整行点击区域保持透明，悬停和键盘均可展开`, async () => {
    const page = await browser.newPage({ viewport: { width: 1280, height: 800 }, reducedMotion: 'reduce' })
    try {
      await page.goto(`${origin}/assets/test/fixtures/browser/home-announcement.html?theme=${theme}`)
      await page.waitForFunction(() => document.documentElement.dataset.ready === 'true')
      await page.getByRole('button', { name: '收起公告', exact: true }).click()
      const expand = page.getByRole('button', { name: '展开公告', exact: true })
      await expand.waitFor({ state: 'visible' })
      await expand.hover()
      await page.waitForFunction(() => !document.querySelector('.gf-announcement-enter-active'))
      // 等待旧实现的背景色过渡结束，以实际渲染结果判定而非匹配工具类。
      await expand.evaluate(async (el) => { await Promise.all(el.getAnimations().map((animation) => animation.finished)) })
      const style = await expand.evaluate((el) => ({
        background: getComputedStyle(el).backgroundColor,
        cursor: getComputedStyle(el).cursor,
        width: el.getBoundingClientRect().width,
        text: el.textContent,
      }))
      assert.equal(style.background, 'rgba(0, 0, 0, 0)', '整行悬停不应出现覆盖空白的背景长条')
      assert.equal(style.cursor, 'pointer')
      assert.ok(style.width > 900, '仍保留整行点击热区')
      assert.ok(style.text.includes('展开公告'), '无标题时显示明确的展开提示')

      await expand.press('Tab')
      await page.keyboard.press('Shift+Tab')
      assert.equal(await expand.evaluate((el) => el.matches(':focus-visible')), true)
      assert.notEqual(await expand.evaluate((el) => getComputedStyle(el).boxShadow), 'none')
      await expand.press('Enter')
      await page.getByRole('button', { name: '收起公告', exact: true }).waitFor({ state: 'visible' })
      assert.equal(await page.locator('.gf-prose-announcement').textContent(), '欢迎来到 YourTJHub！')
    } finally { await page.close() }
  })
}
