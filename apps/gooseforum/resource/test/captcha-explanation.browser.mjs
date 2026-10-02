import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
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

for (const fontSize of [16, 32]) {
  test(`captcha explanation wraps within a 320px publisher at ${fontSize}px text`, async () => {
    const page = await browser.newPage({ viewport: { width: 320, height: 640 } })
    let writeRequests = 0
    let captchaRequests = 0
    try {
      await page.route('**/api/get-captcha', route => {
        captchaRequests++
        return route.fulfill({
          contentType: 'application/json',
          json: { code: 0, result: { captchaId: 'challenge', captchaImg: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/1ioAAAAASUVORK5CYII=' } },
        })
      })
      await page.route('**/api/forum/topics/write', route => {
        writeRequests++
        return route.fulfill({
          contentType: 'application/json',
          json: { code: 1, messageCode: 'common.captchaRequired', params: { action: 'topic.write' }, result: null },
        })
      })
      await page.goto(`${origin}/assets/test/fixtures/browser/quick-publish.html`)
      await page.evaluate(size => { document.documentElement.style.fontSize = `${size}px` }, fontSize)

      const editor = page.locator('.vditor [contenteditable="true"]:visible').first()
      await editor.waitFor()
      await editor.fill('近期互动')
      await page.getByRole('button', { name: '添加分区及话题' }).click()
      await page.getByRole('button', { name: '学习', exact: true }).click()
      const submit = page.getByRole('dialog').getByRole('button', { name: '立即发布', exact: true })
      await submit.waitFor()
      assert.equal(await submit.isEnabled(), true, 'production publish action must be enabled')

      const explanation = page.getByText('站点通常会对新账号的高频发布互动要求验证码。近期发布或回复较多时，请先完成验证；近期互动减少或账号注册时间达到站点设定条件后，此要求会自动解除。')
      try {
        const writeResponsePromise = page.waitForResponse(response => response.url().includes('/api/forum/topics/write'), { timeout: 5000 })
        await submit.click()
        const writeResponse = await writeResponsePromise
        assert.equal(writeResponse.status(), 200)
        const responseData = await writeResponse.json()
        assert.equal(responseData.messageCode, 'common.captchaRequired')
        await page.getByPlaceholder('验证码').waitFor({ timeout: 5000 })
        assert.equal(writeRequests, 1, 'production submit should reach the mocked writer guard')
        assert.equal(captchaRequests, 1, 'writer guard should load one captcha challenge')
        await explanation.waitFor({ timeout: 5000 })
      } catch (error) {
        const dialogText = await page.getByRole('dialog').allInnerTexts().catch(() => [])
        const screenshotPath = join(tmpdir(), `captcha-explanation-${fontSize}-${Date.now()}.png`)
        await page.screenshot({ path: screenshotPath, fullPage: true }).catch(() => {})
        throw new Error(`${error.message}; write requests=${writeRequests}, captcha requests=${captchaRequests}, dialog=${JSON.stringify(dialogText)}, screenshot=${screenshotPath}`)
      }
      const box = await explanation.boundingBox()
      assert.ok(box && box.x >= 0 && box.x + box.width <= 320, `explanation must fit horizontally: ${JSON.stringify(box)}`)
      assert.ok(await explanation.evaluate(el => el.scrollHeight > parseFloat(getComputedStyle(el).lineHeight)), 'long explanation should wrap into readable lines')
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, 'publisher must not overflow horizontally')
    } finally { await page.close() }
  })
}
