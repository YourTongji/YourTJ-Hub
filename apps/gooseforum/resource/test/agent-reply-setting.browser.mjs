import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false }, logLevel: 'error' })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })
for (const width of [390, 1280]) {
  test(`author save/failure and reader restriction at ${width}px`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    let fail = false
    const requests = []
    await page.route('**/api/forum/topics/agent-replies', route => {
      requests.push(route.request().postDataJSON())
      return route.fulfill({ json: fail ? { code: 1, message: '保存失败', result: null } : { code: 0, result: true } })
    })
    await page.goto(`${origin}/assets/test/fixtures/browser/agent-reply-setting.html`)
    const toggle = page.getByRole('switch', { name: '允许机器人回复' })
    await toggle.waitFor()
    assert.equal(await page.locator('#reader [role="switch"]').count(), 0)
    assert.equal(await toggle.getAttribute('aria-checked'), 'true')
    await toggle.click()
    await page.waitForFunction(() => document.querySelector('#author [role="switch"]')?.getAttribute('aria-checked') === 'false')
    await page.getByText('作者已关闭机器人回复', { exact: true }).waitFor()
    assert.deepEqual(requests[0], { topicId: 42, agentRepliesDisabled: true })
    if (process.env.YOURTJ_SCREENSHOT_DIR) await page.screenshot({ path: join(process.env.YOURTJ_SCREENSHOT_DIR, `agent-replies-${width}.png`) })
    fail = true
    await toggle.click()
    await page.getByRole('alert').waitFor()
    assert.equal(await toggle.getAttribute('aria-checked'), 'false')
    fail = false
    await toggle.click()
    await page.waitForFunction(() => document.querySelector('#author [role="switch"]')?.getAttribute('aria-checked') === 'true')
    await page.waitForFunction(() => !document.querySelector('#reader p'))
    assert.deepEqual(requests.at(-1), { topicId: 42, agentRepliesDisabled: false })
    const fits = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)
    assert.equal(fits, true)
    assert.deepEqual(errors, [])
    await page.close()
  })
}
