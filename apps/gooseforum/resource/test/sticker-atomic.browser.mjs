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

async function openEditor() {
  const page = await browser.newPage({ viewport: { width: 720, height: 720 } })
  const errors = []
  page.on('pageerror', (err) => errors.push(err.message))
  await page.goto(`${origin}/assets/test/fixtures/browser/sticker-atomic.html`)
  const editor = page.locator(editorSelector).first()
  await editor.waitFor()
  await editor.click()
  return { page, editor, errors }
}

const assertNoPageErrors = (errors) => assert.deepEqual(errors, [], `page errors: ${errors.join('; ')}`)

const editorText = page => page.locator(editorSelector).first().innerText()
const modelValue = page => page.locator('#model-value').innerText()

test('one Backspace removes a sticker token inserted at the caret', async () => {
  const { page, errors } = await openEditor()
  try {
    await page.keyboard.type('你好 ')
    await page.evaluate(() => document.querySelector('#insert-sticker').click())
    await page.waitForFunction(text => document.querySelector('.vditor [contenteditable="true"]').textContent.includes(text), '[:sticker:huaji:]')
    assert.match(await modelValue(page), /\[:sticker:huaji:\]/, 'inserted token must reach modelValue')

    await page.keyboard.press('Backspace')
    await page.waitForFunction(() => !document.querySelector('#model-value').textContent.includes('[:sticker:'))
    const text = await editorText(page)
    assert.ok(!text.includes('[:sticker:'), `editor text must not keep a partial token: ${JSON.stringify(text)}`)
    assert.equal(text.trim(), '你好')
    assert.equal((await modelValue(page)).trim(), '你好', 'modelValue must drop the token with the editor DOM')
    assertNoPageErrors(errors)
  } finally { await page.close() }
})

test('one Delete removes the token when the caret sits at its start', async () => {
  const { page, editor, errors } = await openEditor()
  try {
    await page.evaluate(() => window.testSetValue('[:sticker:huaji:] 后文'))
    await page.waitForFunction(() => document.querySelector('.vditor [contenteditable="true"]').textContent.includes('[:sticker:huaji:]'))
    // 点正文段落本身：编辑器尾部还有一个 Vditor 维护的空块，点空白处会把光标放进空块
    await editor.locator('p').first().click()
    await page.keyboard.press('Home')
    await page.keyboard.press('Delete')
    await page.waitForFunction(() => !document.querySelector('#model-value').textContent.includes('[:sticker:'))
    const text = await editorText(page)
    assert.ok(!text.includes('[:sticker:'), `editor text must not keep a partial token: ${JSON.stringify(text)}`)
    assert.equal(text.trim(), '后文')
    assert.equal((await modelValue(page)).trim(), '后文')
    assertNoPageErrors(errors)
  } finally { await page.close() }
})

test('a half-typed [:sticker: fragment is deleted as one block', async () => {
  const { page, errors } = await openEditor()
  try {
    await page.keyboard.type('图 [:sticker:na')
    await page.waitForFunction(() => document.querySelector('.vditor [contenteditable="true"]').textContent.includes('[:sticker:na'))
    await page.keyboard.press('Backspace')
    await page.waitForFunction(() => !document.querySelector('#model-value').textContent.includes('[:sticker:'))
    const text = await editorText(page)
    assert.ok(!text.includes('[:sticker:'), `fragment must be gone in one press: ${JSON.stringify(text)}`)
    assert.equal(text.trim(), '图')
    assert.equal((await modelValue(page)).trim(), '图')
    // 删除后光标仍在原位：紧接着输入的字符落在原段落里，而不是落进 Vditor 尾部的空块
    await page.keyboard.type('!')
    await page.waitForFunction(() => document.querySelector('#model-value').textContent.replace(/\s+/g, '') === '图!')
    const firstParagraph = page.locator(`${editorSelector} p`).first()
    assert.equal((await firstParagraph.textContent()).replace(/\s+/g, ''), '图!')
    assertNoPageErrors(errors)
  } finally { await page.close() }
})

test('plain text still deletes one character per Backspace', async () => {
  const { page, errors } = await openEditor()
  try {
    await page.keyboard.type('你好')
    await page.keyboard.press('Backspace')
    await page.waitForFunction(() => document.querySelector('#model-value').textContent.trim() === '你')
    assert.equal((await editorText(page)).trim(), '你')
    assertNoPageErrors(errors)
  } finally { await page.close() }
})
