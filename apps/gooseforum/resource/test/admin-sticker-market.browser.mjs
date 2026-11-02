import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { mkdir } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen(); origin = `http://127.0.0.1:${server.httpServer.address().port}`; browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })
for (const lang of ['zh', 'en', 'ja', 'de']) for (const theme of ['gf-light', 'gf-dark']) for (const width of [320, 1280]) {
  test(`sticker market actions and translated filters fit ${lang}/${theme}/${width}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    try {
      await page.route('**/api/admin/sticker-market?**', route => route.fulfill({ json: { code: 0, result: { enabled: true, source: 'fluxdo', topics: [], packs: [], total: 0, page: 1, pageSize: 20, lastSyncedAt: null } } }))
      await page.goto(`${origin}/assets/test/fixtures/browser/admin-sticker-market.html?lang=${lang}&theme=${theme}`)
      await page.locator('#app[data-ready="true"]').waitFor()
      await page.evaluate(() => document.fonts.ready)
      const labels = await page.locator('#app').evaluate(e => e.dataset)
      const upload = page.getByRole('button', { name: labels.uploadLabel, exact: true })
      await upload.waitFor()
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'the page must fit without horizontal document scrolling')
      const actions = await page.locator('header aside button').evaluateAll(buttons => buttons.map(button => ({ left: button.getBoundingClientRect().left, right: button.getBoundingClientRect().right, width: innerWidth })))
      assert.equal(actions.length, 3)
      assert.ok(actions.every(b => b.left >= 0 && b.right <= b.width), JSON.stringify(actions))
      assert.equal((await page.getByRole('combobox', { name: labels.sourceField, exact: true }).innerText()).trim(), labels.sourceLabel)
      if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
        await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
        await page.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/market-${lang}-${theme}-${width}.png`, fullPage: true })
      }
      await upload.click(); await page.getByRole('dialog').waitFor(); await page.keyboard.press('Escape'); await page.getByRole('dialog').waitFor({ state: 'hidden' })
    } finally { await page.close() }
  })
}

for (const lang of ['zh', 'en', 'ja', 'de']) for (const theme of ['gf-light', 'gf-dark']) {
  test(`uploaded pack details remain inside a 320px viewport ${lang}/${theme}`, async () => {
    const page = await browser.newPage({ viewport: { width: 320, height: 900 } })
    try {
      const pack = { id: 9201, source: 'upload', upstreamId: 'upload-9201', name: 'UI acceptance 1793376869017', topic: 'qa', status: 'ready', published: true, sortOrder: 0, itemCount: 1, failedCount: 0, upstreamCount: 1, coverUrl: '/assets/icon.svg', iconUrl: '' }
      await page.route('**/api/admin/sticker-market?**', route => route.fulfill({ json: { code: 0, result: { enabled: true, topics: [{id:'qa',label:'QA'}], packs: [pack], total: 1, page: 1, pageSize: 20 } } }))
      await page.route('**/api/admin/sticker-market-pack?**', route => route.fulfill({ json: { code: 0, result: { pack, items: [{ stickerId: 1, upstreamId: 'one', name: 'qa_sticker', url: '', aliases: [], isEnabled: true }] } } }))
      await page.goto(`${origin}/assets/test/fixtures/browser/admin-sticker-market.html?lang=${lang}&theme=${theme}`)
      await page.locator('#app[data-ready="true"]').waitFor()
      await page.locator('tbody tr').filter({hasText:pack.name}).locator('button[title]').last().click()
      const dialog = page.getByRole('dialog'); await dialog.waitFor()
      await page.evaluate(() => document.fonts.ready)
      await page.waitForTimeout(250)
      const bounds = await dialog.locator('[data-slot="dialog-header"], [data-slot="dialog-header"] *').evaluateAll(nodes => nodes.filter(n=>n.getBoundingClientRect().width>0).map(n=>({tag:n.tagName,left:n.getBoundingClientRect().left,right:n.getBoundingClientRect().right})))
      assert.ok(bounds.every(r=>r.left>=0 && r.right<=320.1), JSON.stringify(bounds))
      if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
        await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
        await page.screenshot({path:`${process.env.YOURTJ_REVIEW_SCREENSHOTS}/detail-${lang}-${theme}-320.png`})
      }
    } finally {await page.close()}
  })
}
