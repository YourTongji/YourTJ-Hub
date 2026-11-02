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

for (const width of [375, 1280]) {
  test(`选课弹窗保留四角圆角且学院/校区浮层可点选 (${width}px)`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 800 }, reducedMotion: 'reduce' })
    try {
      await page.route('**/api/pk/**', (route) => {
        const pathname = new URL(route.request().url()).pathname
        const data = pathname.endsWith('/campuses')
          ? [{ campusId: 'siping', campusName: '四平路校区' }]
          : pathname.endsWith('/faculties') ? [{ facultyId: 'math', facultyName: '数学科学学院' }] : []
        return route.fulfill({ json: { code: 0, data } })
      })
      await page.goto(`${origin}/assets/test/fixtures/browser/schedule-course-picker.html`)
      const dialog = page.getByRole('dialog')
      await dialog.waitFor({ state: 'visible' })
      const corners = await dialog.evaluate((element) => {
        const header = element.firstElementChild
        const footer = element.lastElementChild
        const headerRect = header.getBoundingClientRect()
        const footerRect = footer.getBoundingClientRect()
        return [
          [header, headerRect.left + 1, headerRect.top + 1],
          [header, headerRect.right - 1, headerRect.top + 1],
          [footer, footerRect.left + 1, footerRect.bottom - 1],
          [footer, footerRect.right - 1, footerRect.bottom - 1],
        ].map(([surface, horizontal, vertical]) => surface.contains(document.elementFromPoint(horizontal, vertical)))
      })
      assert.deepEqual(corners, [false, false, false, false], '顶栏和底栏背景不应覆盖弹窗圆角外区域')
      assert.equal(await dialog.locator('.overflow-y-auto').count(), 1)
      await dialog.getByRole('tab').nth(2).click()
      for (const [index, label] of [[0, '四平路校区'], [1, '数学科学学院']]) {
        const trigger = dialog.getByRole('combobox').nth(index)
        await trigger.click()
        const listbox = page.getByRole('listbox')
        await listbox.waitFor({ state: 'visible' })
        assert.ok(await listbox.evaluate((element) => {
          const rect = element.getBoundingClientRect()
          return rect.left >= 0 && rect.top >= 0 && rect.right <= innerWidth && rect.bottom <= innerHeight
        }), '学院/校区浮层完整位于视口内')
        await page.getByRole('option', { name: label }).click()
        await listbox.waitFor({ state: 'hidden' })
        assert.equal(await dialog.count(), 1, '选择选项后选课弹窗保持打开')
        assert.ok((await trigger.innerText()).includes(label))
      }
    } finally {
      await page.close()
    }
  })
}
