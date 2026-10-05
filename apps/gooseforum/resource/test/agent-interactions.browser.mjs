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

for (const width of [320, 1280]) {
  test(`Agent delivery controls fit ${width}px`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    try {
      const errors = []; page.on('pageerror', error => errors.push(error.message))
      const delivery = { id: 14, eventId: `evt_${'a'.repeat(64)}`, status: 'accepted', endpointGeneration: 3, round: 1, attemptCount: 1, totalAttempts: 1, expiresAt: '2026-10-11T12:00:00Z', attempts: [{ id: 'attempt_1', number: 1, round: 1, httpStatus: 204, durationMs: 28, authorizedAt: '2026-10-04T12:00:00Z', completedAt: '2026-10-04T12:00:01Z' }] }
      await page.route('**/api/admin/agent-webhook-deliveries', route => route.fulfill({ json: { code: 0, result: { list: [delivery], total: 1, page: 1, pageSize: 10 } } }))
      await page.route('**/api/admin/agent-interaction-intents', route => route.fulfill({ json: { code: 0, result: { list: [], total: 0, page: 1, pageSize: 10 } } }))
      await page.goto(`${origin}/assets/test/fixtures/browser/agent-interactions.html?lang=en`)
      await page.getByTestId('delivery-14').waitFor()
      assert.equal(await page.getByText('Receiver accepted', { exact: true }).isVisible(), true)
      const dialog = page.getByRole('dialog')
      const box = await dialog.boundingBox()
      assert.ok(box && box.x >= 0 && box.x + box.width <= width, 'dialog fits viewport')
      assert.ok(await dialog.evaluate(element => element.scrollWidth <= element.clientWidth), 'dialog must not overflow horizontally, including long event IDs')
      await page.getByTestId('delivery-14').scrollIntoViewIfNeeded()
      if (process.env.AGENT_INTERACTIONS_SCREENSHOT_DIR) await page.screenshot({ path: `${process.env.AGENT_INTERACTIONS_SCREENSHOT_DIR}/yourtj-1042-admin-${width}.png`, fullPage: true })
      assert.deepEqual(errors, [])
    } finally { await page.close() }
  })
  test(`Agent mention identity and selection fit ${width}px`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    try {
      await page.goto(`${origin}/assets/test/fixtures/browser/agent-interactions.html?lang=en&mention=1`)
      const bot = page.getByRole('option', { name: /Forum assistant.*Bot/ })
      await bot.waitFor()
      assert.equal(await bot.getByText('Bot', { exact: true }).isVisible(), true)
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'mention candidates fit viewport')
      if (process.env.AGENT_INTERACTIONS_SCREENSHOT_DIR) await page.screenshot({ path: `${process.env.AGENT_INTERACTIONS_SCREENSHOT_DIR}/yourtj-1042-mention-${width}.png`, fullPage: true })
      await bot.click()
      assert.equal(await page.locator('html').getAttribute('data-selected'), 'true')
    } finally { await page.close() }
  })
}

for (const width of [320, 1280]) {
  test(`Agent comment policy fits ${width}px and saves controls`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    try {
      const errors = []
      page.on('pageerror', error => errors.push(error.message))
      await page.route('**/api/admin/agent-comment-policy', route => route.fulfill({ json: { code: 0, result: { allowAgentComments: true } } }))
      await page.route('**/api/admin/topics/list', route => route.fulfill({ json: { code: 0, result: { list: [{ id: 1201, title: 'A public topic', username: 'classmate', agentCommentDisabled: false }], page: 1, size: 10, total: 1, hasNext: false } } }))
      await page.route('**/api/admin/save-agent-comment-policy', route => route.fulfill({ json: { code: 0 } }))
      await page.goto(`${origin}/assets/test/fixtures/browser/agent-comment-policy.html`)
      const globalSwitch = page.getByRole('switch', { name: 'Allow Agent comments', exact: true })
      await globalSwitch.waitFor()
      await page.getByText('A public topic', { exact: true }).waitFor()
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'policy page fits viewport')
      await globalSwitch.click()
      const request = page.waitForRequest('**/api/admin/save-agent-comment-policy')
      await page.getByRole('button', { name: 'Save', exact: true }).click()
      assert.equal((await request).postDataJSON().allowAgentComments, false)
      await page.getByRole('button', { name: 'Save', exact: true }).waitFor()
      if (process.env.AGENT_INTERACTIONS_SCREENSHOT_DIR) await page.screenshot({ path: `${process.env.AGENT_INTERACTIONS_SCREENSHOT_DIR}/yourtj-1054-policy-${width}.png`, fullPage: true })
      assert.deepEqual(errors, [])
    } finally { await page.close() }
  })
}
