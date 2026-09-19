import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server, browser, origin
const calendar = JSON.parse(readFileSync(new URL('../../../../packages/api-contract/fixtures/campus-calendar-export-success.json', import.meta.url), 'utf8'))
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })

const translations = {
  "zh": {
    "timetable": "我的课表",
    "exportCalendar": "导出课程日历",
    "applyAdjustments": "开启调休规则"
  },
  "en": {
    "timetable": "My timetable",
    "exportCalendar": "Export course calendar",
    "applyAdjustments": "Apply holiday adjustments"
  },
  "ja": {
    "timetable": "マイ時間割",
    "exportCalendar": "授業カレンダーをエクスポート",
    "applyAdjustments": "休講・振替規則を適用"
  },
  "de": {
    "timetable": "Mein Stundenplan",
    "exportCalendar": "Stundenplan exportieren",
    "applyAdjustments": "Feiertagsregeln anwenden"
  }
}
for (const [locale, text] of Object.entries(translations)) for (const width of [320, 1280]) {
  test(`campus calendar downloads a complete ICS and fits ${width}px in ${locale}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    try {
      await page.route('**/api/**', route => {
        const path = new URL(route.request().url()).pathname
        if (path === '/api/campus/calendar-export') return route.fulfill({ json: calendar })
        if (path === '/api/campus/status') return route.fulfill({ json: { code: 0, result: { enabled: true, binding: { maskedId: 'DE••MO', revision: 'demo', needsAuthorization: false }, candidate: null } } })
        const key = path.split('/').at(-1)
        return route.fulfill({ json: { code: 0, data: { sectionTimes: [] }, result: {
          key, status: 'ready', updatedAt: '', metrics: key === 'calendar' ? [{ label: '教学周', value: '1' }] : [], columns: [], rows: [], series: [],
          events: key === 'timetable' ? [{ name: '示例课程', teacher: '示例教师', room: 'A101', campus: '四平路', day: 1, start: 1, end: 2, weeks: [1, 3], credits: '2' }] : [],
        } } })
      })
      await page.goto(`${origin}/assets/test/fixtures/browser/campus.html?lang=${locale}`)
      await page.getByRole('button', { name: text.timetable, exact: true }).click()
      const button = page.getByRole('button', { name: text.exportCalendar, exact: true })
      await button.waitFor()
      const box = await button.boundingBox()
      assert.ok(box && box.x >= 0 && box.x + box.width <= width, 'export control stays in the viewport')
      const toggle = page.getByRole('switch', { name: text.applyAdjustments })
      assert.equal(await toggle.isChecked(), true)
      await page.getByText(text.applyAdjustments, { exact: true }).click()
      assert.equal(await toggle.isChecked(), false)
      const pending = page.waitForEvent('download')
      const exportRequest = page.waitForRequest(r => r.url().includes('calendar-export?applyAdjustments=false'))
      await button.click()
      await exportRequest
      const download = await pending
      assert.equal(download.suggestedFilename(), calendar.result.filename)
      const stream = await download.createReadStream()
      const chunks = []
      for await (const chunk of stream) chunks.push(chunk)
      assert.equal(Buffer.concat(chunks).toString('utf8'), calendar.result.content)
      assert.equal(await download.failure(), null)
    } finally { await page.close() }
  })
}
