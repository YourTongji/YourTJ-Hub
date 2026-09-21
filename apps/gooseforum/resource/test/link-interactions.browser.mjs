import assert from 'node:assert/strict'
import { before, after, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen(); origin = `http://127.0.0.1:${server.httpServer.address().port}`; browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })
async function setup(width = 320) {
  const page = await browser.newPage({ viewport: { width, height: 900 } })
  await page.route('**/api/link-previews/resolve', async route => {
    const { urls } = route.request().postDataJSON()
    await route.fulfill({ json: { code: 0, result: urls.map(url => ({ requestedUrl: url, kind: 'external', status: 'ready', url, displayHost: new URL(url).hostname, title: 'Preview' })) } })
  })
  await page.context().route('https://outside.example/**', route => route.fulfill({ body: 'External destination' }))
  await page.goto(`${origin}/assets/test/fixtures/browser/link-interactions.html`)
  await page.waitForFunction(() => window.linkFixture?.ready())
  return page
}
test('guard preserves keyboard, focus, modifier and middle-click intent at 320px', async () => {
  const page = await setup()
  try {
    const anchor = page.locator('#external')
    await anchor.focus(); await page.keyboard.press('Enter')
    const dialog = page.locator('dialog[open]'); await dialog.waitFor()
    assert.equal(await dialog.getAttribute('aria-modal'), 'true')
    assert.match(await dialog.innerText(), /outside\.example/)
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true)
    for (let i = 0; i < 10; i++) { await page.keyboard.press('Tab'); assert.equal(await page.evaluate(() => document.querySelector('dialog').contains(document.activeElement)), true) }
    await page.keyboard.press('Escape'); await dialog.waitFor({ state: 'hidden' })
    await page.waitForFunction(() => document.activeElement?.id === 'external')
    assert.equal(page.url().includes('link-interactions.html'), true)
    for (const modifiers of [[process.platform === 'darwin' ? 'Meta' : 'Control'], ['Shift']]) {
      await anchor.click({ modifiers }); await dialog.waitFor()
      assert.equal(await page.evaluate(() => window.linkFixture.pending().newTab), true)
      await page.keyboard.press('Escape'); await dialog.waitFor({ state: 'hidden' })
    }
    await anchor.click({ button: 'middle' }); await dialog.waitFor()
    assert.equal(await page.evaluate(() => window.linkFixture.pending().newTab), true)
    await page.keyboard.press('Escape')
    await page.locator('#blocked').click(); await dialog.waitFor()
    assert.equal(await page.evaluate(() => window.linkFixture.pending().risk), 'blocked')
    assert.equal(await dialog.locator('button.gf-button-primary').count(), 0)
    await page.keyboard.press('Escape')
    await anchor.click({ button: 'middle' }); await dialog.waitFor()
    await dialog.locator('input[type="checkbox"]').check()
    const popupPromise = page.waitForEvent('popup')
    await dialog.locator('button.gf-button-primary').click()
    const popup = await popupPromise; await popup.waitForLoadState()
    assert.equal(popup.url(), 'https://outside.example/article'); await popup.close()
    await page.locator('#suspicious').click(); await dialog.waitFor()
    assert.equal(await page.evaluate(() => window.linkFixture.pending().risk), 'suspicious')

  } finally { await page.close() }
})
test('real Vditor hint stays outside source and preserves selection during IME and undo', async () => {
  const page = await setup(640)
  try {
    const editable = page.locator('.vditor-wysiwyg [contenteditable="true"]')
    await page.locator('.gf-link-preview-editor-hint:not([hidden])').waitFor()
    assert.equal(await editable.locator('.gf-link-preview-editor-hint').count(), 0)
    await editable.click(); await page.keyboard.press('End')
    const focusBefore = await page.evaluate(() => document.activeElement?.className)
    await editable.dispatchEvent('compositionstart')
    await page.keyboard.insertText(' 中文')
    await editable.dispatchEvent('compositionend')
    await page.waitForFunction(() => document.querySelector('.gf-link-preview-editor-hint').hidden)
    assert.equal(await page.evaluate(() => document.activeElement?.className), focusBefore)
    const value = await page.evaluate(() => window.linkFixture.getValue())
    assert.match(value, /中文/); assert.doesNotMatch(value, /gf-link|Preview|<!--/)
    await page.waitForFunction(() => window.linkFixture.model().trim() === window.linkFixture.getValue().trim())
    await page.keyboard.press(process.platform === 'darwin' ? 'Meta+z' : 'Control+z')
    const undone = await page.evaluate(() => window.linkFixture.getValue())
    assert.doesNotMatch(undone, /gf-link|Preview/)
    assert.notEqual(undone, value)
    await page.evaluate(() => window.linkFixture.setValue('https://old.example/article ordinary text'))
    await page.waitForFunction(() => document.querySelector('.gf-link-preview-editor-hint').hidden)
    await page.evaluate(() => window.linkFixture.setValue('[plain](https://old.example/article)'))
    await page.waitForTimeout(550)
    assert.equal(await page.locator('.gf-link-preview-editor-hint').isHidden(), true)
    assert.doesNotMatch(await page.evaluate(() => window.linkFixture.getValue()), /gf-link|Preview/)
  } finally { await page.close() }
})
