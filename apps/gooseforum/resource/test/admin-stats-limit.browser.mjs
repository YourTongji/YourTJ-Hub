import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { mkdir, readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server, browser, origin
const defaults = JSON.parse(await readFile(new URL('../../app/models/defaultconfig/pageconfig/ratelimit.json', import.meta.url), 'utf8'))
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })
for (const lang of ['zh', 'en', 'ja', 'de']) for (const width of [320, 1280]) {
 test(`admin stats exception and configurable budget: ${lang} ${width}`, async () => {
  const page = await browser.newPage({ viewport: { width, height: 900 } })
  let saved
  try {
    await page.route('**/api/admin/rate-limit-settings', route => route.fulfill({ json: { code: 0, result: defaults } }))
    await page.route('**/api/admin/save-rate-limit-settings', async route => {
      saved = route.request().postDataJSON()
      await route.fulfill({ json: { code: 0, result: true } })
    })
    await page.goto(`${origin}/assets/test/fixtures/browser/admin-stats-limit.html?lang=${lang}&dark=${width === 320}`)
    await page.getByText('admin.stats', { exact: true }).waitFor()
    assert.equal(await page.evaluate(() => getComputedStyle(document.documentElement).colorScheme), width === 320 ? 'dark' : 'light')
    const hintText = await page.locator('#app').getAttribute('data-limit-hint')
    const hint = page.getByText(hintText, { exact: true })
    await hint.waitFor()
    assert.ok(!hintText.startsWith('statsAdmin.'))
    const box = await hint.boundingBox()
    assert.ok(box.width > 100 && box.height > 10 && box.x >= 0 && box.x + box.width <= width)
    assert.equal(await page.getByRole('switch').nth(1).getAttribute('aria-checked'), 'true')
    const rule = page.getByText('admin.stats', { exact: true }).locator('..')
    assert.equal(await rule.getByRole('spinbutton').nth(2).inputValue(), '60')
    if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
      await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
      await hint.locator('..').screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/stats-exemption-${lang}-${width}.png` })
    }
    await rule.getByRole('spinbutton').nth(2).fill('30')
    const label = await page.locator('#app').getAttribute('data-save-label')
    await Promise.all([page.waitForResponse('**/api/admin/save-rate-limit-settings'), page.getByRole('button', { name: label, exact: true }).click()])
    assert.equal(saved.settings.skipAdmin, true)
    assert.deepEqual(saved.settings.actions.find(x => x.action === 'admin.stats'), { action: 'admin.stats', windowSeconds: 60, limitPerIp: 0, limitPerUser: 30 })
  } finally { await page.close() }
 })
}
