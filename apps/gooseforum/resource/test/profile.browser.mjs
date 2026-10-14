import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server
let browser
let origin

before(async () => {
  server = await createServer({
    root: fileURLToPath(new URL('../', import.meta.url)),
    server: { host: '127.0.0.1', port: 0, open: false },
  })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})

after(async () => {
  await browser?.close()
  await server?.close()
})

for (const width of [320, 1280]) {
  for (const hidden of [false, true]) {
    test(`anonymous profile remains usable when content is ${hidden ? 'hidden' : 'visible'} at ${width}px`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 900 } })
      try {
        await page.route('**/api/forum/unread-status', route => route.fulfill({
          json: { code: 0, result: { notifications: false, messages: false } },
        }))
        await page.goto(
          `${origin}/assets/test/fixtures/browser/profile.html?${hidden ? 'hidden=1' : ''}`,
        )
        await page.getByText('躲进云里的猫', { exact: true }).first().waitFor()
        assert.ok(
          await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
          'anonymous profile must not overflow horizontally',
        )
        if (hidden) {
          await page.locator('[data-test="anonymous-content-hidden"]:visible').waitFor()
          assert.equal(await page.locator('article a[href*="?tab="]').count(), 0)
          assert.equal(await page.getByText('分享一件最近让你开心的小事', { exact: true }).count(), 0)
        } else {
          await page.locator('article a[href$="?tab=topics"]').waitFor()
          await page.locator('article a[href$="?tab=replies"]').waitFor()
          await page.getByText('分享一件最近让你开心的小事', { exact: true }).waitFor()
        }
      } finally {
        await page.close()
      }
    })
  }
}

for (const width of [320, 1280]) {
  test(`personal website tooltip stays readable inside the profile card at ${width}px`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    const website = `https://example.com/${'profile/'.repeat(18)}`
    try {
      const query = new URLSearchParams({ member: '1', website })
      await page.goto(`${origin}/assets/test/fixtures/browser/profile.html?${query}`)
      const websiteLink = page.locator(`a[href="${website}"]`)
      await websiteLink.waitFor()
      await websiteLink.hover()
      const tooltip = websiteLink.locator('.gf-tooltip')
      await tooltip.waitFor()
      await page.waitForFunction(element => getComputedStyle(element).opacity === '1', await tooltip.elementHandle())

      const bounds = await page.evaluate(() => {
        const tooltip = document.querySelector('a[aria-label^="https://example.com/"] .gf-tooltip')
        const card = tooltip?.closest('section.gf-card')
        const trigger = tooltip?.parentElement
        if (!tooltip || !card || !trigger) return null
        const tooltipRect = tooltip.getBoundingClientRect()
        const cardRect = card.getBoundingClientRect()
        const triggerRect = trigger.getBoundingClientRect()
        return {
          tooltipLeft: tooltipRect.left,
          tooltipRight: tooltipRect.right,
          tooltipBottom: tooltipRect.bottom,
          cardLeft: cardRect.left,
          cardRight: cardRect.right,
          triggerTop: triggerRect.top,
          tooltipScrollWidth: tooltip.scrollWidth,
          tooltipClientWidth: tooltip.clientWidth,
          pageScrollWidth: document.documentElement.scrollWidth,
        }
      })

      assert.ok(bounds, 'tooltip and profile card must be mounted')
      assert.ok(bounds.tooltipLeft >= bounds.cardLeft, 'tooltip must not be clipped at the card left edge')
      assert.ok(bounds.tooltipRight <= bounds.cardRight, 'tooltip must not be clipped at the card right edge')
      assert.ok(bounds.tooltipBottom <= bounds.triggerTop, 'tooltip must appear above its icon')
      assert.ok(bounds.tooltipScrollWidth <= bounds.tooltipClientWidth, 'the complete URL must wrap instead of truncate')
      assert.ok(bounds.pageScrollWidth <= width, 'tooltip must not introduce horizontal page overflow')
    } finally {
      await page.close()
    }
  })
}
