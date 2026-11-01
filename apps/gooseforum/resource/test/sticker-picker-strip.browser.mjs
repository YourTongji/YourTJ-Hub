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

const square = '<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><rect width="64" height="64" rx="12" fill="#3b82f6"/></svg>'
const packs = Array.from({ length: 14 }, (_, i) => ({ id: i + 1, name: `表情包 ${i + 1}`, coverUrl: `/sticker/cover-${i}.png`, stickerCount: 4 }))
const officialPacks = Array.from({ length: 9 }, (_, i) => ({ key: `pack-${i}`, name: `官方分组 ${i + 1}`, coverUrl: '', stickerCount: 1 }))
const stickers = officialPacks.map((pack, i) => ({ name: `s${i}`, url: `/sticker/s${i}.png`, pack: pack.key }))

async function openPicker() {
  const page = await browser.newPage({ viewport: { width: 480, height: 640 } })
  const errors = []
  page.on('pageerror', (err) => errors.push(err.message))
  await page.route('**/sticker/*.png', (route) => route.fulfill({ contentType: 'image/svg+xml', body: square }))
  await page.route('**/api/forum/stickers', (route) => route.fulfill({ json: { code: 0, result: stickers } }))
  await page.route('**/api/forum/sticker-packs', (route) => route.fulfill({ json: { code: 0, result: officialPacks } }))
  await page.route('**/api/forum/my-sticker-packs', (route) => route.fulfill({ json: { code: 0, result: packs } }))
  await page.route('**/api/forum/sticker-market*', (route) => route.fulfill({ json: { code: 0, result: { topics: [], packs: [], total: 0 } } }))
  await page.goto(`${origin}/assets/test/fixtures/browser/sticker-picker-strip.html`)
  await page.locator('.gf-sticker-tabs .gf-sticker-tab').nth(15).waitFor()
  await page.locator('.gf-sticker-topics .gf-sticker-chip').nth(9).waitFor()
  return { page, errors }
}

const strip = (page, selector) => page.locator('.gf-scroll-strip').filter({ has: page.locator(selector) }).first()
const edges = (locator) => locator.evaluate((element) => ({
  back: element.hasAttribute('data-back'),
  forward: element.hasAttribute('data-forward'),
  scrollLeft: element.querySelector('.gf-scroll-strip-track').scrollLeft,
}))

test('the wheel scrolls overflowing pack tabs sideways and the edges fade', async () => {
  const { page, errors } = await openPicker()
  try {
    const tabs = strip(page, '.gf-sticker-tabs')
    assert.deepEqual(await edges(tabs), { back: false, forward: true, scrollLeft: 0 })
    const box = await tabs.boundingBox()
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
    await page.mouse.wheel(0, 240)
    await page.waitForFunction(() => document.querySelector('.gf-sticker-tabs').scrollLeft > 0)
    const state = await edges(tabs)
    assert.equal(state.back, true)
    assert.equal(await page.evaluate(() => window.scrollY), 0, 'the page must not scroll while the strip can')
    const mask = await tabs.locator('.gf-scroll-strip-track').evaluate((element) => getComputedStyle(element).maskImage || getComputedStyle(element).webkitMaskImage)
    assert.match(mask, /linear-gradient/)

    // At the far end the wheel goes back to the page instead of being swallowed.
    await page.locator('.gf-sticker-tabs').evaluate((element) => { element.scrollLeft = element.scrollWidth })
    await page.waitForFunction(() => !document.querySelector('.gf-scroll-strip:has(> .gf-sticker-tabs)').hasAttribute('data-forward'))
    await page.mouse.wheel(0, 240)
    await page.waitForFunction(() => window.scrollY > 0)
    assert.deepEqual(errors, [])
  } finally { await page.close() }
})

test('hover arrows page through the strip and the active tab stays in view', async () => {
  const { page, errors } = await openPicker()
  try {
    const tabs = strip(page, '.gf-sticker-tabs')
    const forward = tabs.locator('.gf-scroll-strip-arrow-forward')
    await tabs.hover()
    await page.waitForFunction(() => getComputedStyle(document.querySelector('.gf-scroll-strip-arrow-forward')).opacity === '1')
    assert.equal(await forward.getAttribute('aria-hidden'), 'true')
    assert.equal(await forward.getAttribute('tabindex'), '-1')
    await forward.click()
    await page.waitForFunction(() => document.querySelector('.gf-sticker-tabs').scrollLeft > 100)
    await tabs.locator('.gf-scroll-strip-arrow-back').click()
    await page.waitForFunction(() => document.querySelector('.gf-sticker-tabs').scrollLeft === 0)

    // The market tab is the last one: choosing it scrolls it fully into view.
    await page.locator('.gf-sticker-tabs .gf-sticker-tab').last().evaluate((element) => element.click())
    await page.waitForFunction(() => {
      const track = document.querySelector('.gf-sticker-tabs').getBoundingClientRect()
      const tab = document.querySelector('.gf-sticker-tabs [aria-pressed="true"]').getBoundingClientRect()
      return tab.right <= track.right && tab.left >= track.left
    })
    assert.deepEqual(errors, [])
  } finally { await page.close() }
})

test('official pack chips scroll with the wheel too', async () => {
  const { page, errors } = await openPicker()
  try {
    const chips = strip(page, '.gf-sticker-topics')
    assert.equal((await edges(chips)).forward, true)
    const box = await chips.boundingBox()
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
    await page.mouse.wheel(0, 200)
    await page.waitForFunction(() => document.querySelector('.gf-sticker-topics').scrollLeft > 0)
    assert.deepEqual(errors, [])
  } finally { await page.close() }
})

test('the panel keeps its full height whatever the tab holds', async () => {
  const { page, errors } = await openPicker()
  try {
    const panelHeight = () => page.locator('.gf-sticker-picker').evaluate((element) => element.getBoundingClientRect().height)
    // min(288px, 44vh) at a 640px-high viewport.
    const expected = Math.min(288, 640 * 0.44)
    assert.ok(Math.abs(await panelHeight() - expected) < 0.5)

    // The empty market tab keeps the same frame and leaves the rest blank.
    await page.locator('.gf-sticker-tabs .gf-sticker-tab').last().evaluate((element) => element.click())
    await page.locator('.gf-sticker-body > p').first().waitFor()
    assert.ok(Math.abs(await panelHeight() - expected) < 0.5)
    const body = await page.locator('.gf-sticker-body').evaluate((element) => ({ content: element.querySelector(':scope > p').getBoundingClientRect().height, box: element.clientHeight }))
    assert.ok(body.content < body.box / 2)
    assert.deepEqual(errors, [])
  } finally { await page.close() }
})

test('the hover preview floats outside the panel instead of covering the grid', async () => {
  const { page, errors } = await openPicker()
  try {
    const cell = page.locator('.gf-sticker-cell').first()
    await cell.hover()
    const preview = page.locator('body > .gf-sticker-preview')
    await preview.waitFor()
    await page.waitForTimeout(200)
    const [panelBox, previewBox, cellBox] = await Promise.all([page.locator('.gf-sticker-picker').boundingBox(), preview.boundingBox(), cell.boundingBox()])
    const overlaps = (a, b) => a.x < b.x + b.width && b.x < a.x + a.width && a.y < b.y + b.height && b.y < a.y + a.height
    assert.equal(overlaps(previewBox, panelBox), false)
    assert.ok(previewBox.x >= panelBox.x + panelBox.width, 'the trailing side has room at this width')
    // Level with the pointed cell, inside the viewport.
    assert.ok(Math.abs(previewBox.y + previewBox.height / 2 - (cellBox.y + cellBox.height / 2)) < 2)
    assert.ok(previewBox.y >= 0 && previewBox.y + previewBox.height <= 640)
    // No native tooltip stacks over it.
    assert.equal(await cell.getAttribute('title'), null)
    assert.deepEqual(errors, [])
  } finally { await page.close() }
})

test('a failed second market page retries from the picker without losing page one', async () => {
  const { page, errors } = await openPicker()
  const requested = []
  let attempts = 0
  try {
    await page.route('**/api/forum/sticker-market?*', async route => {
      const number = Number(new URL(route.request().url()).searchParams.get('page'))
      requested.push(number)
      if (number === 2 && attempts++ === 0) return route.fulfill({ status: 503, json: {} })
      await route.fulfill({ json: { code: 0, result: { topics: [], packs: [{ ...packs[0], id: 9000 + number, name: `Market page ${number}` }], total: 2 } } })
    })
    await page.locator('.gf-sticker-tabs .gf-sticker-tab').last().evaluate(element => element.click())
    await page.getByText('Market page 1', { exact: true }).waitFor()
    await page.locator('.gf-sticker-more').click()
    await page.locator('button[aria-label="重试"]').click()
    await page.getByText('Market page 2', { exact: true }).waitFor()
    assert.equal(await page.getByText('Market page 1', { exact: true }).count(), 1)
    assert.deepEqual(requested, [1, 2, 2])
    assert.deepEqual(errors, [])
  } finally { await page.close() }
})
