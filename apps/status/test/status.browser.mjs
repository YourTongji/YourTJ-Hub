import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

const fixture = JSON.parse(readFileSync(new URL('./fixtures/status-connected.json', import.meta.url), 'utf8'))
let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })

for (const [width, lang, theme] of [[320, 'zh', 'gf-light'], [390, 'de', 'gf-dark'], [1280, 'en', 'gf-dark'], [390, 'ja', 'gf-light']]) {
  test(`status stays usable at ${width}px in ${lang} / ${theme}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 844 } })
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    try {
      await page.route('**/api/status?*', async route => {
        const response = structuredClone(fixture)
        response.result.range = new URL(route.request().url()).searchParams.get('range')
        response.result.serverRange = new URL(route.request().url()).searchParams.get('serverRange')
        response.result.server.fetchedAt = response.result.traffic.fetchedAt = response.result.server.data.current.observedAt = new Date().toISOString()
        response.result.uptime.fetchedAt = response.result.uptime.data.monitors[0].current.time = new Date().toISOString()
        response.result.uptime.data.monitors[0].history = Array.from({ length: 100 }, (_, index) => ({ time: new Date(Date.now() - (99 - index) * 60_000).toISOString(), status: index % 20 === 0 ? 'down' : 'up', ping: 628 }))
        await route.fulfill({ json: response })
      })
      await page.goto(`${origin}/?lang=${lang}&theme=${theme}`)
      await page.locator('.chart-bucket').first().waitFor()
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'page must not overflow')
      assert.ok(await page.locator('.status-page button').evaluateAll(buttons => buttons.every(button => { const r = button.getBoundingClientRect(); return r.left >= 0 && r.right <= innerWidth })), 'all buttons remain in the viewport')
      const uptimeHeight = await page.locator('.uptime-section').evaluate(el => el.getBoundingClientRect().height)
      const checks = page.locator('.heartbeat-strip button')
      await checks.first().hover()
      await page.waitForFunction(() => getComputedStyle(document.querySelector('.heartbeat-detail')).opacity === '1')
      for (const index of [1, 20, 45, 99]) {
        await checks.nth(index).hover()
        assert.equal(await page.locator('.heartbeat-detail').getAttribute('aria-hidden'), 'false')
        assert.equal(await page.locator('.heartbeat-detail').textContent(), await checks.nth(index).getAttribute('aria-label'))
        assert.equal(await page.locator('.heartbeat-detail').evaluate(el => getComputedStyle(el).opacity), '1', 'moving between checks must not restart the fade')
        assert.equal(await page.locator('.uptime-section').evaluate(el => el.getBoundingClientRect().height), uptimeHeight, 'details must not resize the panel')
      }
      const firstCheck = await checks.first().boundingBox()
      await page.mouse.move(firstCheck.x + firstCheck.width + 0.5, firstCheck.y + firstCheck.height / 2)
      assert.equal(await page.locator('.heartbeat-detail').getAttribute('aria-hidden'), 'false', 'the gap between checks must retain details')
      await page.locator('#uptime-title').hover()
      await page.waitForFunction(() => getComputedStyle(document.querySelector('.heartbeat-detail')).opacity === '0')
      assert.equal(await page.locator('.uptime-section').evaluate(el => el.getBoundingClientRect().height), uptimeHeight, 'dismissal must not resize the panel')
      await page.locator('.heartbeat-strip button').first().focus()
      await page.locator('.heartbeat-detail').waitFor()
      await page.locator('.chart-bucket').last().focus()
      await page.locator('.chart-tooltip').waitFor()
      await page.locator('.status-range button').nth(1).click()
      await page.waitForFunction(() => document.querySelectorAll('.status-range button')[1].getAttribute('aria-pressed') === 'true' && !document.querySelector('.status-refresh').disabled)
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'period change must not overflow')
      for (const index of [2, 3, 0]) {
        await page.locator('.status-resource-range button').nth(index).click()
        await page.waitForFunction(index => document.querySelectorAll('.status-resource-range button')[index].getAttribute('aria-pressed') === 'true' && !document.querySelector('.status-refresh').disabled, index)
        await page.locator('.resource-history svg').waitFor()
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'resource range and date labels must not overflow')
      }
      assert.deepEqual(errors, [])
    } finally { await page.close() }
  })
}

test('keeps the independent page readable during an API failure and reports unknown health', async () => {
  const page = await browser.newPage({ viewport: { width: 390, height: 844 } })
  let fail = false
  const requests = []
  page.on('request', request => requests.push(request.url()))
  try {
    await page.route('**/api/status?*', async route => {
      const response = structuredClone(fixture)
      response.result.uptime.fetchedAt = response.result.uptime.data.monitors[0].current.time = new Date().toISOString()
      await route.fulfill(fail ? { status: 503, json: { error: 'Snapshot unavailable' } } : { json: response })
    })
    await page.goto(`${origin}/?lang=zh`)
    await page.locator('#status-signal').filter({ hasText: '服务可正常访问' }).waitFor()
    fail = true
    await page.locator('.status-refresh').click()
    await page.locator('#status-signal').filter({ hasText: '状态暂不可确认' }).waitFor()
    assert.ok(await page.locator('.status-traffic').isVisible())
    assert.ok(requests.every(url => url.startsWith(origin)), 'page must not require the forum or external runtime assets')
  } finally { await page.close() }
})
