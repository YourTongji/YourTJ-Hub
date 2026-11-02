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
const stickers = [{ name: 's0', url: '/sticker/s0.png', pack: 'pack-0' }]

async function openPicker() {
  const page = await browser.newPage({ viewport: { width: 480, height: 640 } })
  const errors = []
  page.on('pageerror', (err) => errors.push(err.message))
  await page.route('**/sticker/*.png', (route) => route.fulfill({ contentType: 'image/svg+xml', body: square }))
  await page.route('**/api/forum/stickers', (route) => route.fulfill({ json: { code: 0, result: stickers } }))
  await page.route('**/api/forum/sticker-packs', (route) => route.fulfill({ json: { code: 0, result: [] } }))
  await page.route('**/api/forum/my-sticker-packs', (route) => route.fulfill({ json: { code: 0, result: [] } }))
  await page.route('**/api/forum/sticker-market*', (route) => route.fulfill({ json: { code: 0, result: { topics: [], packs: [], total: 0 } } }))
  await page.goto(`${origin}/assets/test/fixtures/browser/sticker-picker-flip.html`)
  await page.locator('.gf-sticker-picker').waitFor()
  return { page, errors }
}

// 锚点贴在视口底部（见 fixture）：placement=below 的定高面板越出视口，
// 应翻转到 above（bottom-14）并保持完整可见。
test('the picker flips above its anchor when the preferred side leaves the viewport', async () => {
  const { page, errors } = await openPicker()
  try {
    const panel = page.locator('.gf-sticker-picker')
    assert.equal(await panel.evaluate((element) => element.classList.contains('bottom-14')), true)
    assert.equal(await panel.evaluate((element) => element.classList.contains('top-12')), false)
    const box = await panel.boundingBox()
    assert.ok(box.y >= 0 && box.y + box.height <= 640, 'flipped panel stays inside the viewport')
    assert.deepEqual(errors, [])
  } finally { await page.close() }
})
