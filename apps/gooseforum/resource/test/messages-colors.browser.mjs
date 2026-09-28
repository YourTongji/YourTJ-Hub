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

for (const theme of ['gf-light', 'gf-dark']) {
  test(`${theme}: outgoing text and forwarded previews use readable white on blue`, async () => {
    const page = await browser.newPage({ viewport: { width: 1000, height: 800 } })
    try {
      await page.route('**/api/**', (route) => {
        const list = [
          { id: 1, isSelf: false, content: 'Incoming message', createdAt: '2026-09-28T10:00:00Z' },
          { id: 2, isSelf: true, content: 'Outgoing message', createdAt: '2026-09-28T10:01:00Z' },
          { id: 3, isSelf: true, content: '[Chat history]', createdAt: '2026-09-28T10:02:00Z', forwarded: {
            version: 1, messages: [{ senderName: 'Bob', content: 'Forwarded preview', createdAt: '2026-09-28T09:00:00Z', msgType: 1 }],
          } },
        ]
        return route.fulfill({ json: { code: 0, result: { list, hasMoreBefore: false } } })
      })
      await page.goto(`${origin}/assets/test/fixtures/browser/messages-colors.html?theme=${theme}&userId=2`)
      const outgoing = page.getByText('Outgoing message', { exact: true })
      await outgoing.waitFor()
      const surface = await outgoing.evaluate((el) => ({ background: getComputedStyle(el).backgroundColor, color: getComputedStyle(el).color }))
      assert.deepEqual(surface, { background: 'rgb(37, 99, 235)', color: 'rgb(255, 255, 255)' })
      const preview = page.getByText('Bob: Forwarded preview', { exact: true })
      const previewStyle = await preview.evaluate((el) => ({ color: getComputedStyle(el).color, opacity: getComputedStyle(el).opacity }))
      assert.deepEqual(previewStyle, { color: 'rgb(255, 255, 255)', opacity: '1' })
      assert.equal(await preview.evaluate((el) => getComputedStyle(el.closest('button').parentElement).backgroundColor), surface.background)
      // Opening the forwarded history returns to the neutral incoming surface.
      await preview.click()
      const detail = page.getByRole('dialog').getByText('Forwarded preview', { exact: true })
      await detail.waitFor()
      assert.notEqual(await detail.evaluate((el) => getComputedStyle(el).color), surface.color)
      const incoming = page.getByText('Incoming message', { exact: true })
      assert.notEqual(await incoming.evaluate((el) => getComputedStyle(el).backgroundColor), surface.background)
    } finally { await page.close() }
  })
}
