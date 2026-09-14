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
function status() {
  return { maintenanceEnabled: true, configured: true, available: true, engineVersion: '1.11.3', jobs: [], indexes: ['topics', 'users', 'categories', 'courses', 'wiki_pages'].map((index, i) => ({
    index, expectedVersion: 1, exists: true, documents: [8600, 3240, 18, 11540, 489][i], indexing: false,
    lastCheck: { index, expectedVersion: 1, observedVersions: { 0: 100, 1: 500 }, expected: 600, indexed: 600, missing: 2, extra: 2, outdated: 100, settingsOK: true, stable: true, complete: false, checkedAt: '2026-09-14T05:00:00Z' },
  })) }
}
for (const lang of ['zh', 'en', 'ja', 'de']) {
  for (const width of [320, 1440]) {
    test(`all index controls fit ${lang} ${width}px`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 1000 } })
      try {
        const errors = []; page.on('pageerror', error => errors.push(error.message))
        await page.route('**/api/admin/search/indexes', route => route.fulfill({ json: { code: 0, result: status() } }))
        await page.goto(`${origin}/assets/test/fixtures/browser/search-maintenance.html?lang=${lang}`)
        await page.getByText('Meilisearch', { exact: false }).first().waitFor()
        await page.waitForFunction(() => document.querySelectorAll('button').length >= 13)
        assert.deepEqual(errors, [])
        assert.equal(await page.getByText(/Meilisearch.*1\.11\.3/).isVisible(), true)
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'page must not overflow')
        if (process.env.SEARCH_MAINTENANCE_SCREENSHOT && lang === 'zh' && width === 1440) await page.screenshot({ path: process.env.SEARCH_MAINTENANCE_SCREENSHOT, fullPage: true })
      } finally { await page.close() }
    })
  }
}
test('rebuild confirmation, server deduplication and unavailable controls', async () => {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } })
  try {
    const data = status()
    let posts = 0
    await page.route('**/api/admin/search/indexes', route => route.fulfill({ json: { code: 0, result: data } }))
    await page.route('**/api/admin/search/maintenance', route => {
      posts++
      assert.deepEqual(route.request().postDataJSON(), { index: 'all', action: 'rebuild' })
      const job = { id: 42, status: 0, createdAt: '2026-09-14T05:00:00Z', updatedAt: '0001-01-01T00:00:00Z', retryCount: 0, errorCode: '', index: 'all', action: 'check', requestedBy: 1, phase: 'queued', currentIndex: '', processed: 0, reports: [] }
      data.jobs = [job]
      return route.fulfill({ json: { code: 0, result: { created: false, job } } })
    })
    await page.goto(`${origin}/assets/test/fixtures/browser/search-maintenance.html?lang=zh`)
    await page.getByRole('button', { name: '重建全部', exact: true }).click()
    assert.equal(posts, 0, 'opening confirmation must not start rebuild')
    await page.getByTestId('admin-confirm').click()
    await page.getByText('#42', { exact: true }).waitFor()
    assert.equal(posts, 1)
    assert.equal(await page.getByRole('button', { name: '检查全部', exact: true }).isDisabled(), true)
    data.jobs = []; data.available = false
    await page.getByRole('button', { name: '刷新状态', exact: true }).click()
    await page.getByRole('alert').waitFor()
    assert.equal(await page.getByRole('button', { name: '重建全部', exact: true }).isDisabled(), true)
  } finally { await page.close() }
})
