import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdir } from 'node:fs/promises'
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

const locales = {
  zh: { cancel: '取消', empty: '还没有草稿' },
  en: { cancel: 'Cancel', empty: 'No drafts yet' },
  ja: { cancel: 'キャンセル', empty: '下書きはまだありません' },
  de: { cancel: 'Abbrechen', empty: 'Noch keine Entwürfe' },
}

async function confirmDeletion(page) {
  await page.getByRole('button', { name: /A saved draft/ }).click()
  await page.locator('[data-test="confirm-delete"]').click()
}

async function mockDeletion(page, success = true) {
  await page.route('**/api/forum/user/content-event', route => route.fulfill({ json: { code: 0, result: true } }))
  await page.route('**/api/forum/topics/delete', route => route.fulfill({ json: { code: 0, result: success } }))
}

for (const width of [375, 1280]) for (const theme of ['gf-light', 'gf-dark']) {
  test(`普通删除即时提示且刷新不重复：${width}px ${theme}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    try {
      await mockDeletion(page)
      await page.goto(`${origin}/assets/test/fixtures/browser/drafts-page.html?theme=${theme}`)
      await confirmDeletion(page)
      const banner = page.locator('.gf-flash-banner')
      await banner.waitFor()
      assert.equal(await banner.count(), 1)
      assert.match(await banner.textContent(), /草稿已移入回收站/)
      assert.equal(await page.evaluate(() => sessionStorage.getItem('goose:flash-messages')), null)
      assert.equal(await page.locator('article').count(), 0)
      await page.waitForTimeout(250)
      if (process.env.YOURTJ_TEST_SCREENSHOT_DIR) {
        await mkdir(process.env.YOURTJ_TEST_SCREENSHOT_DIR, { recursive: true })
        await page.screenshot({ path: `${process.env.YOURTJ_TEST_SCREENSHOT_DIR}/draft-delete-${width}-${theme}.png`, fullPage: true })
      }
      await page.reload()
      await page.getByRole('button', { name: /A saved draft/ }).waitFor()
      assert.equal(await page.locator('.gf-flash-banner').count(), 0)
    } finally { await page.close() }
  })
}

test('满窗口删除重载后只显示一次提示，下次刷新不再显示', async () => {
  const page = await browser.newPage()
  let navigations = 0
  page.on('framenavigated', frame => { if (frame === page.mainFrame()) navigations++ })
  try {
    await mockDeletion(page)
    await page.goto(`${origin}/assets/test/fixtures/browser/drafts-page.html?count=100`)
    await confirmDeletion(page)
    await page.locator('.gf-flash-banner').waitFor()
    assert.equal(navigations, 2)
    assert.equal(await page.locator('.gf-flash-banner').count(), 1)
    assert.equal(await page.evaluate(() => sessionStorage.getItem('goose:flash-messages')), null)
    await page.reload()
    await page.getByRole('button', { name: /A saved draft/ }).waitFor()
    assert.equal(await page.locator('.gf-flash-banner').count(), 0)
  } finally { await page.close() }
})

test('删除失败保留确认框和草稿，不产生成功提示', async () => {
  const page = await browser.newPage()
  try {
    await mockDeletion(page, false)
    await page.goto(`${origin}/assets/test/fixtures/browser/drafts-page.html`)
    await confirmDeletion(page)
    await page.getByRole('alert').waitFor()
    assert.equal(await page.getByRole('alertdialog').count(), 1)
    assert.equal(await page.locator('article').count(), 1)
    assert.equal(await page.locator('.gf-flash-banner').count(), 0)
    assert.equal(await page.evaluate(() => sessionStorage.getItem('goose:flash-messages')), null)
  } finally { await page.close() }
})

for (const [locale, copy] of Object.entries(locales)) {
  test(`草稿删除确认可用键盘与触屏并正确更新空状态（${locale}，Chromium）`, async () => {
    const page = await browser.newPage({
      viewport: { width: 360, height: 780 },
      isMobile: true,
      hasTouch: true,
    })
    page.setDefaultTimeout(8_000)
    page.setDefaultNavigationTimeout(8_000)
    let deletionCount = 0
    let deleteBody
    try {
      await page.route('**/api/forum/user/content-event', route => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, result: true }),
      }))
      await page.route('**/api/forum/topics/delete', async route => {
        deletionCount++
        deleteBody = route.request().postDataJSON()
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ code: 0, result: true }),
        })
      })

      await page.goto(`${origin}/assets/test/fixtures/browser/drafts-page.html?lang=${locale}`)
      const deleteButton = page.getByRole('button', { name: /A saved draft/ })
      const editLink = page.getByRole('link', { name: /A saved draft/ }).first()
      await deleteButton.waitFor()
      await editLink.waitFor()
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
      assert.ok((await deleteButton.boundingBox()).height >= 40)

      await deleteButton.focus()
      await page.keyboard.press('Enter')
      const dialog = page.getByRole('alertdialog')
      await dialog.waitFor()
      assert.match(await dialog.textContent(), /A saved draft/)
      assert.equal(await page.evaluate(() => document.activeElement?.getAttribute('data-test')), 'cancel-delete')
      await page.keyboard.press('Escape')
      await dialog.waitFor({ state: 'detached' })
      assert.equal(deletionCount, 0)

      const buttonBox = await deleteButton.boundingBox()
      await page.touchscreen.tap(buttonBox.x + buttonBox.width / 2, buttonBox.y + buttonBox.height / 2)
      await dialog.waitFor()
      await page.locator('[data-test="confirm-delete"]').focus()
      await page.keyboard.press('Enter')
      await page.getByText(copy.empty, { exact: true }).waitFor()

      assert.equal(deletionCount, 1)
      assert.equal(deleteBody.topicId, 42)
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
      assert.equal(await page.evaluate(() => document.activeElement?.getAttribute('href')), '/publish')
    } finally {
      await page.close()
    }
  })
}
