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

const editorSelector = '.vditor [contenteditable="true"]:visible'
const square = '<svg xmlns="http://www.w3.org/2000/svg" width="128" height="128"><circle cx="64" cy="64" r="60" fill="#f5b400"/></svg>'
const library = [
  { name: 'huaji', url: '/sticker/huaji.png', pack: 'official' },
  { name: 'hug', url: '/sticker/hug.png', pack: 'official' },
  { name: 'doge', url: '/sticker/doge.png', pack: 'official' },
]

async function openEditor(query = '') {
  const page = await browser.newPage({ viewport: { width: 720, height: 720 } })
  const errors = []
  page.on('pageerror', (err) => errors.push(err.message))
  await page.route('**/sticker/*.png', (route) => route.fulfill({ contentType: 'image/svg+xml', body: square }))
  await page.route('**/api/forum/stickers', (route) => route.fulfill({ json: { code: 0, result: library } }))
  await page.route('**/api/forum/sticker-packs', (route) => route.fulfill({ json: { code: 0, result: [] } }))
  await page.route('**/api/forum/my-stickers', (route) => route.fulfill({ json: { code: 0, result: [] } }))
  await page.route('**/api/forum/my-sticker-packs', (route) => route.fulfill({ json: { code: 0, result: [] } }))
  await page.route('**/api/forum/stickers/resolve', async (route) => {
    const { names } = route.request().postDataJSON()
    await route.fulfill({ json: { code: 0, result: library.filter((item) => names.includes(item.name)) } })
  })
  await page.goto(`${origin}/assets/test/fixtures/browser/sticker-editor.html${query}`)
  const editor = page.locator(editorSelector).first()
  await editor.waitFor()
  await editor.click()
  return { page, editor, errors }
}

const modelValue = (page) => page.locator('#model-value').innerText()
const waitModel = (page, expected) => page.waitForFunction((text) => document.querySelector('#model-value').textContent.trim() === text, expected)
const inlineStickers = (page) => page.locator(`${editorSelector} .gf-sticker-inline`)
const assertNoPageErrors = (errors) => assert.deepEqual(errors, [], `page errors: ${errors.join('; ')}`)

test('an inserted sticker shows as a 2.5em image and typing continues after it', async () => {
  const { page, errors } = await openEditor()
  try {
    await page.keyboard.type('你好')
    await page.evaluate(() => document.querySelector('#insert-sticker').click())
    const sticker = inlineStickers(page).first()
    await sticker.waitFor()
    assert.equal(await sticker.getAttribute('data-sticker'), 'huaji')
    assert.equal(await sticker.getAttribute('contenteditable'), 'false')
    const metrics = await sticker.evaluate((element) => {
      const style = getComputedStyle(element.parentElement)
      const box = element.getBoundingClientRect()
      return { fontSize: parseFloat(style.fontSize), width: box.width, height: box.height, background: getComputedStyle(element).backgroundImage }
    })
    assert.ok(Math.abs(metrics.height - metrics.fontSize * 2.5) < 0.5, `height ${metrics.height}`)
    assert.ok(Math.abs(metrics.width - metrics.fontSize * 2.5) < 0.5, `width ${metrics.width}`)
    assert.match(metrics.background, /sticker\/huaji\.png/)

    await page.keyboard.type('！')
    await waitModel(page, '你好[:sticker:huaji:]！')
    assert.equal(await inlineStickers(page).count(), 1, 'the sticker survives the next block re-render')
    const paragraphText = await page.locator(`${editorSelector} p`).first().textContent()
    assert.equal(paragraphText.replace(/\s+/g, ''), '你好[:sticker:huaji:]！', 'the typed text lands after the image')
    assertNoPageErrors(errors)
  } finally { await page.close() }
})

test('one Backspace after the image removes the whole sticker', async () => {
  const { page, errors } = await openEditor()
  try {
    await page.keyboard.type('图')
    await page.evaluate(() => document.querySelector('#insert-sticker').click())
    await inlineStickers(page).first().waitFor()
    await waitModel(page, '图[:sticker:huaji:]')
    await page.keyboard.press('Backspace')
    await waitModel(page, '图')
    assert.equal(await inlineStickers(page).count(), 0)
    await page.keyboard.type('!')
    await waitModel(page, '图!')
    assertNoPageErrors(errors)
  } finally { await page.close() }
})

test('typing [:sticker: suggests stickers and Enter inserts the image', async () => {
  const { page, errors } = await openEditor()
  try {
    await page.keyboard.type('看 [:sticker:hu')
    const options = page.locator('.gf-sticker-suggest-option')
    await options.first().waitFor()
    assert.deepEqual(await options.allInnerTexts(), ['huaji', 'hug'])
    await page.keyboard.press('ArrowDown')
    assert.equal(await page.locator('.gf-sticker-suggest-option.is-active').innerText(), 'hug')
    await page.keyboard.press('Enter')
    await page.locator('.gf-sticker-suggest').waitFor({ state: 'detached' })
    await waitModel(page, '看 [:sticker:hug:]')
    assert.equal(await inlineStickers(page).count(), 1)
    assert.equal(await page.locator(`${editorSelector} p`).count() <= 2, true, 'Enter must not open a new paragraph')
    assertNoPageErrors(errors)
  } finally { await page.close() }
})

test('Escape dismisses suggestions and a fully typed token becomes an image', async () => {
  const { page, errors } = await openEditor()
  try {
    await page.keyboard.type('[:sticker:doge')
    await page.locator('.gf-sticker-suggest-option').first().waitFor()
    await page.keyboard.press('Escape')
    await page.locator('.gf-sticker-suggest').waitFor({ state: 'detached' })
    await page.keyboard.type(':]')
    await inlineStickers(page).first().waitFor()
    await waitModel(page, '[:sticker:doge:]')
    assertNoPageErrors(errors)
  } finally { await page.close() }
})

test('unknown tokens and tokens in code stay as text', async () => {
  const { page, errors } = await openEditor(`?value=${encodeURIComponent('未知 [:sticker:nobody:] 代码 `[:sticker:huaji:]` 正文 [:sticker:huaji:]')}`)
  try {
    await inlineStickers(page).first().waitFor()
    assert.equal(await inlineStickers(page).count(), 1)
    const text = await page.locator(editorSelector).first().innerText()
    assert.match(text, /\[:sticker:nobody:\]/)
    assert.equal(await page.locator(`${editorSelector} code .gf-sticker-inline`).count(), 0)
    assertNoPageErrors(errors)
  } finally { await page.close() }
})

test('emoji shortcode names keep their tokens and render like posts', async () => {
  const source = '下课见[:sticker:doge:][:sticker:b:] :smile:'
  const { page, errors } = await openEditor(`?value=${encodeURIComponent(source)}`)
  try {
    await inlineStickers(page).first().waitFor()
    assert.deepEqual(await inlineStickers(page).evaluateAll((items) => items.map((item) => item.dataset.sticker)), ['doge'])
    await page.locator(`${editorSelector} p`).first().click()
    await page.keyboard.press('End')
    await page.keyboard.type('!')
    await waitModel(page, `${source}!`)
    const text = await page.locator(`${editorSelector} p`).first().textContent()
    assert.ok(text.includes('[:sticker:b:]') && text.includes(':smile:'), `shortcodes stay literal: ${text}`)
    assertNoPageErrors(errors)
  } finally { await page.close() }
})
