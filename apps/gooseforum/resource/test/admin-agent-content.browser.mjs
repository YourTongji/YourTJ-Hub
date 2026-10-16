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
  test(`bot correction keeps draft on conflict and fits at ${width}px`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    const saves = []
    await page.route('**/api/admin/agents/content-source', route => route.fulfill({ json: { code: 0, result: { topicId: 10, postId: 11, title: 'Campus announcement', content: 'Original Markdown body', categoryId: [1], revisionNo: 3, botUserId: 9, botName: 'Campus Bot' } } }))
    await page.route('**/api/admin/agents/edit-topic', route => {
      saves.push(route.request().postDataJSON())
      return route.fulfill({ status: 409, json: { code: 1, messageCode: 'admin.agent.contentConflict' } })
    })
    await page.goto(`${origin}/assets/test/fixtures/browser/agent-content.html`)
    await page.getByLabel('Topic ID', { exact: true }).fill('10')
    await page.getByRole('button', { name: 'Load source', exact: true }).click()
    const content = page.getByLabel('Markdown content', { exact: true })
    await page.getByRole('checkbox', { name: 'Study', exact: true }).check()
    await page.getByRole('checkbox', { name: 'Life', exact: true }).check()
    assert.equal(await page.getByRole('checkbox', { name: 'Help', exact: true }).isDisabled(), true)
    await content.fill('Corrected content must survive conflict')
    await page.getByRole('button', { name: 'Save correction', exact: true }).click()
    await page.getByRole('alert').waitFor()
    assert.equal(await content.inputValue(), 'Corrected content must survive conflict')
    assert.deepEqual(saves[0], { topicId: 10, title: 'Campus announcement', categoryId: [1, 2, 3], content: 'Corrected content must survive conflict', expectedRevisionNo: 3 })
    assert.match(await page.getByRole('alert').innerText(), /draft is kept/)
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true)
    assert.deepEqual(errors, [])
    if (process.env.YOURTJ_SCREENSHOT_DIR) await page.screenshot({ path: join(process.env.YOURTJ_SCREENSHOT_DIR, `agent-content-${width}.png`) })
    await page.close()
  })
}
