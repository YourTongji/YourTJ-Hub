import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { mkdir, mkdtemp } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server, browser, origin, evidence
before(async () => {
  evidence = process.env.CATEGORY_PINS_EVIDENCE_DIR || await mkdtemp(join(tmpdir(), 'category-pins-'))
  await mkdir(evidence, { recursive: true })
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })

for (const theme of ['gf-light', 'gf-dark']) {
  for (const width of [1280, 375]) {
    test(`category latest pins remain reachable: ${theme}, ${width}px`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: 'reduce' })
      try {
        const open = async (extra = '') => {
          await page.goto(`${origin}/assets/test/fixtures/browser/category-pins.html?theme=${theme}${extra}`)
          await page.waitForFunction(() => document.documentElement.dataset.ready === 'true')
        }
        await open()
        const strip = page.getByRole('region', { name: '置顶话题' })
        const toggle = strip.getByRole('button')
        assert.equal(await strip.locator('a').count(), 1)
        assert.equal(await toggle.getAttribute('aria-expanded'), 'false')
        assert.equal(await page.locator('[data-feed-view] a[href="/p/post/1"]').count(), 0)
        await page.screenshot({ path: join(evidence, `web-${theme}-${width}-collapsed.png`) })
        await toggle.focus()
        await page.keyboard.press('Enter')
        assert.equal(await toggle.getAttribute('aria-expanded'), 'true')
        assert.equal(await strip.locator('a').count(), 2)
        assert.equal(await strip.locator('a[href="/p/post/2"]').getAttribute('href'), '/p/post/2')
        await strip.locator('a[href="/p/post/2"]').click({ trial: true })
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'category must fit viewport')
        await page.screenshot({ path: join(evidence, `web-${theme}-${width}-expanded.png`) })
        await toggle.press('Enter')
        assert.equal(await toggle.getAttribute('aria-expanded'), 'false')

        await open('&state=only')
        assert.equal(await strip.locator('a').count(), 1)
        assert.equal(await page.locator('[data-feed-view] [data-feed-id]').count(), 0)
        await open('&state=none')
        assert.equal(await strip.count(), 0)
        await open('&sort=new')
        assert.equal(await strip.count(), 0)
        assert.equal(await page.locator('[data-feed-view] [data-feed-id]').count(), 4)
      } finally { await page.close() }
    })
  }
}
