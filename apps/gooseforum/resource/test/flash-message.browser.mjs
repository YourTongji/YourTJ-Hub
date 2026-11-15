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

test('queued feedback is consumed exactly once after reload', async () => {
  const page = await browser.newPage()
  const message = 'reload-once-regression'
  const ready = page.locator('html[data-ready="true"]')
  try {
    await page.goto(`${origin}/assets/test/fixtures/browser/flash-message.html`)
    await ready.waitFor()
    await page.evaluate(async text => {
      const { queueFlashMessage } = await import('/assets/src/runtime/flash-message.ts')
      queueFlashMessage(text, 'info')
    }, message)
    await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => resolve(undefined))))
    assert.equal(await page.getByText(message, { exact: true }).count(), 0, 'queued feedback stays hidden in the current document')
    assert.ok(await page.evaluate(() => sessionStorage.getItem('goose:flash-messages')), 'feedback is persisted for the next document')

    await page.reload()
    await ready.waitFor()
    const notice = page.getByText(message, { exact: true })
    await notice.waitFor()
    assert.equal(await notice.count(), 1, 'the queued feedback appears once')
    assert.equal(await page.evaluate(() => sessionStorage.getItem('goose:flash-messages')), null, 'hydration consumes the queue')

    await page.reload()
    await ready.waitFor()
    assert.equal(await page.getByText(message, { exact: true }).count(), 0, 'a later document does not replay consumed feedback')
  } finally {
    await page.close()
  }
})
