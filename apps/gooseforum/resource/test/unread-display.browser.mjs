import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdir } from 'node:fs/promises'
import { join } from 'node:path'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), optimizeDeps: { entries: ['test/fixtures/browser/unread-display.html'] }, server: { host: '127.0.0.1', port: 0, open: false }, logLevel: 'error' })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
  if (process.env.YOURTJ_TEST_ARTIFACTS) await mkdir(process.env.YOURTJ_TEST_ARTIFACTS, { recursive: true })
})
after(async () => { await browser?.close(); await server?.close() })

for (const language of ['zh', 'en', 'ja', 'de']) for (const theme of ['light', 'dark']) for (const width of [320, 1280]) {
  test(`unread preference propagates to every indicator: ${language}/${theme}/${width}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 }, colorScheme: theme })
    let style = 'number', failNext = true, writes = 0, countReads = 0, markedRead = 0
    const unexpected = [], errors = []
    page.on('pageerror', error => errors.push(error.message))
    await page.route('**/api/**', async route => {
      const request = route.request(), path = new URL(request.url()).pathname
      if (path === '/api/forum/unread-status') {
        countReads++
        return route.fulfill({ json: { code: 0, result: { notifications: true, messages: true, notificationCount: 3, messageCount: 12, badgeStyle: style } } })
      }
      if (path === '/api/forum/notification-preferences' && request.method() === 'PUT') {
        writes++
        const body = request.postDataJSON()
        assert.deepEqual(Object.keys(body), ['badgeStyle'])
        assert.ok(['dot', 'number'].includes(body.badgeStyle))
        if (failNext) { failNext = false; return route.fulfill({ status: 500, json: { code: 1, result: null, messageCode: 'common.operation.failed' } }) }
        style = body.badgeStyle
        return route.fulfill({ json: { code: 0, result: { badgeStyle: style } } })
      }
      if (path === '/api/forum/events') return route.fulfill({ status: 200, contentType: 'text/event-stream', body: 'retry: 60000\n\n' })
      if (path === '/api/forum/stickers') return route.fulfill({ json: { code: 0, result: [] } })
      if (path === '/api/forum/anonymous/state') return route.fulfill({ json: { code: 0, result: { persona: null } } })
      if (path === '/api/forum/chat/messages') return route.fulfill({ json: { code: 0, result: { list: [], hasMoreBefore: false, hasMoreAfter: false, nextBeforeId: 0, latestId: 0 } } })
      if (path === '/api/forum/chat/mark-read') { markedRead++; return route.fulfill({ json: { code: 0, result: true } }) }
      unexpected.push(`${request.method()} ${path}`)
      return route.fulfill({ status: 404, json: { code: 1 } })
    })
    const url = `${origin}/assets/test/fixtures/browser/unread-display.html?lang=${language}&theme=${theme}`
    await page.goto(url)
    await page.waitForFunction(() => document.documentElement.dataset.ready === 'true')
    const label = await page.evaluate(() => document.documentElement.dataset.switchLabel)
    const control = page.getByRole('switch', { name: label })
    await control.scrollIntoViewIfNeeded()
    assert.equal(await control.getAttribute('aria-checked'), 'true')
    assert.match(await page.locator('[data-unread-summary]').innerText(), /3/)
    const readsBefore = countReads, markedBefore = markedRead
    await control.click()
    await page.waitForFunction(() => !!document.querySelector('[role="alert"]'))
    assert.equal(await control.getAttribute('aria-checked'), 'true')
    await control.click()
    await page.waitForFunction(() => document.querySelector('[role="switch"]')?.getAttribute('aria-checked') === 'false' && !document.querySelector('[role="alert"]'))
    assert.equal(await page.locator('[data-unread-summary]').count(), 0)
    const badges = await page.locator('[data-unread-indicator]').evaluateAll(nodes => nodes.map(node => ({ style: node.dataset.unreadStyle, text: node.textContent, label: node.getAttribute('aria-label') })))
    assert.ok(badges.length >= 3)
    for (const badge of badges) { assert.equal(badge.style, 'dot'); assert.equal(badge.text.trim(), ''); assert.doesNotMatch(badge.label, /\d/) }
    assert.equal(countReads, readsBefore, 'changing display style does not fetch counts again')
    assert.equal(markedRead, markedBefore, 'changing style does not mark a message read')
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
    if (process.env.YOURTJ_TEST_ARTIFACTS) await page.screenshot({ path: join(process.env.YOURTJ_TEST_ARTIFACTS, `unread-dot-${language}-${theme}-${width}.png`) })
    await page.reload()
    await page.waitForFunction(() => document.documentElement.dataset.ready === 'true')
    assert.equal(await control.getAttribute('aria-checked'), 'false', 'account preference survives a fresh read')
    await control.scrollIntoViewIfNeeded()
    await control.click()
    await page.waitForFunction(() => document.querySelector('[role="switch"]')?.getAttribute('aria-checked') === 'true' && !document.querySelector('[role="status"]'))
    assert.match(await page.locator('[data-unread-summary]').innerText(), /3/)
    assert.equal(writes, 3)
    assert.deepEqual(unexpected, [])
    assert.deepEqual(errors, [])
    await page.close()
  })
}
