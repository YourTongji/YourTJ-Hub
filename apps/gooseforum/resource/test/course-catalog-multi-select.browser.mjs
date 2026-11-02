import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
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

for (const width of [320, 800]) for (const side of ['up', 'down']) test(`course filters remain visible and selectable at ${width}px when opening ${side}`, async () => {
  const page = await browser.newPage({ viewport: { width, height: 700 } })
  try {
    await page.goto(`${origin}/assets/test/fixtures/browser/course-catalog-multi-select.html?side=${side}`)
    const trigger = page.getByRole('button', { name: '全部院系' })
    await trigger.click()
    const panel = page.locator('[role="group"]')
    await panel.waitFor()
    const card = page.locator('.gf-card')
    const bounds = await page.evaluate(() => {
      const card = document.querySelector('.gf-card').getBoundingClientRect()
      const panel = document.querySelector('[role="group"]').getBoundingClientRect()
      return { cardTop: card.top, cardBottom: card.bottom, panelTop: panel.top, panelBottom: panel.bottom, panelLeft: panel.left, panelRight: panel.right }
    })
    assert.ok(side === 'up' ? bounds.panelTop < bounds.cardTop : bounds.panelBottom > bounds.cardBottom, JSON.stringify(bounds))
    assert.ok(bounds.panelLeft >= 0 && bounds.panelRight <= width, JSON.stringify(bounds))
    assert.equal(await card.evaluate((el) => getComputedStyle(el).overflow), 'hidden')

    const option = panel.getByText('数学科学学院', { exact: true })
    const box = await option.boundingBox()
    assert.ok(box)
    const checkbox = panel.getByRole('checkbox', { name: '数学科学学院' })
    assert.equal(await checkbox.evaluate((input) => input.form?.id), 'course-more-filters')
    const hit = await page.evaluate(({ x, y }) => document.elementFromPoint(x, y)?.closest('label')?.textContent?.trim(), { x: box.x + box.width / 2, y: box.y + box.height / 2 })
    assert.equal(hit, '数学科学学院')
    await page.mouse.click(box.x + box.width / 2, box.y + box.height / 2)
    assert.equal(await checkbox.isChecked(), true)

    await page.evaluate(() => window.scrollTo(0, 100))
    assert.equal(await page.evaluate(() => window.scrollY), 100)
    await page.waitForFunction(() => {
      const trigger = document.querySelector('button[aria-expanded="true"]').getBoundingClientRect()
      const panel = document.querySelector('[role="group"]').getBoundingClientRect()
      const opensUp = panel.bottom < trigger.top
      return Math.abs((opensUp ? panel.bottom + 4 : panel.top - 4) - (opensUp ? trigger.top : trigger.bottom)) < 1
    })
  } finally {
    await page.close()
  }
})
