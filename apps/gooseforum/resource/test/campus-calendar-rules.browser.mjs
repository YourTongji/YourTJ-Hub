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
const translations = {
  "zh": {
    "adminYear": "通知年份",
    "adminNotice": "学校通知",
    "adminGenerate": "AI 生成草稿",
    "adminFromLabel": "补课 1 原教学日",
    "adminToLabel": "补课 1 实际上课日",
    "adminApply": "应用规则",
    "adminApplied": "规则已应用。之后开启调休的课程日历导出将使用这些规则。",
    "errorRulesInvalid": "规则有冲突或日期无效，请检查后重试。"
  },
  "en": {
    "adminYear": "Notice year",
    "adminNotice": "School notice",
    "adminGenerate": "Generate AI draft",
    "adminFromLabel": "Makeup 1 original date",
    "adminToLabel": "Makeup 1 actual date",
    "adminApply": "Apply rules",
    "adminApplied": "Rules applied. Future exports with adjustments enabled will use these rules.",
    "errorRulesInvalid": "Rules conflict or contain invalid dates. Check them and try again."
  },
  "ja": {
    "adminYear": "通知の年",
    "adminNotice": "大学の通知",
    "adminGenerate": "AI で下書きを生成",
    "adminFromLabel": "振替 1 の元の授業日",
    "adminToLabel": "振替 1 の実施日",
    "adminApply": "規則を適用",
    "adminApplied": "規則を適用しました。今後、規則を有効にした出力に反映されます。",
    "errorRulesInvalid": "規則が競合しているか日付が無効です。確認して再試行してください。"
  },
  "de": {
    "adminYear": "Jahr der Mitteilung",
    "adminNotice": "Hochschulmitteilung",
    "adminGenerate": "KI-Entwurf erstellen",
    "adminFromLabel": "Nachholtermin 1: ursprüngliches Datum",
    "adminToLabel": "Nachholtermin 1: tatsächliches Datum",
    "adminApply": "Regeln anwenden",
    "adminApplied": "Regeln angewendet. Künftige Exporte mit aktivierten Anpassungen verwenden diese Regeln.",
    "errorRulesInvalid": "Regeln widersprechen sich oder enthalten ungültige Daten. Bitte prüfen und erneut versuchen."
  }
}
for (const [locale, text] of Object.entries(translations)) for (const width of [375, 1280]) {
  test(`admin reviews AI rules before applying and retains failed drafts at ${width}px in ${locale}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 1000 } })
    let saved = 0, parseInput, savedInput
    const initial = { revision: '0'.repeat(64), rules: { holidays: [], moves: [] } }
    try {
      await page.route('**/api/admin/campus/calendar-rules**', async route => {
        const r = route.request()
        if (r.url().endsWith('/parse')) {
          parseInput = r.postDataJSON()
          return route.fulfill({ json: { code: 0, result: { rules: { holidays: [{ name: '国庆', startDate: '2026-10-01', endDate: '2026-10-07' }], moves: [{ name: '补课', fromDate: '2026-10-06', toDate: '2026-09-20' }] }, warnings: ['请核实教学对应日期'] } } })
        }
        if (r.method() === 'POST') {
          saved++; savedInput = r.postDataJSON()
          if (saved === 1) return route.fulfill({ status: 400, json: { code: 1, messageCode: 'campus.rulesInvalid', params: { reason: '请核实日期' } } })
          return route.fulfill({ json: { code: 0, result: { ...savedInput, revision: '1'.repeat(64) } } })
        }
        return route.fulfill({ json: { code: 0, result: initial } })
      })
      await page.goto(`${origin}/assets/test/fixtures/browser/campus-rules.html?lang=${locale}`)
      await page.getByLabel(text.adminYear).fill('2026')
      await page.getByLabel(text.adminNotice, { exact: true }).fill('国庆节10月1日至7日放假，9月20日安排10月6日教学工作。')
      await page.getByRole('button', { name: text.adminGenerate }).click()
      await page.getByText('请核实教学对应日期', { exact: true }).waitFor()
      assert.equal(saved, 0, 'parse must not publish')
      assert.equal(parseInput.year, 2026)
      assert.equal(await page.getByLabel(text.adminFromLabel, { exact: true }).inputValue(), '2026-10-06')
      await page.getByLabel(text.adminToLabel, { exact: true }).fill('2026-09-21')
      await page.getByRole('button', { name: text.adminApply, exact: true }).click()
      await page.getByText(locale === 'zh' ? '请核实日期' : text.errorRulesInvalid, { exact: true }).waitFor()
      assert.equal(await page.getByLabel(text.adminToLabel, { exact: true }).inputValue(), '2026-09-21', 'save failure retains edits')
      await page.getByRole('button', { name: text.adminApply, exact: true }).click()
      await page.getByRole('status').filter({ hasText: text.adminApplied }).waitFor()
      assert.equal(saved, 2)
      assert.equal(savedInput.revision, initial.revision)
      assert.equal(savedInput.rules.moves[0].toDate, '2026-09-21')
      assert.equal(await page.getByRole('button', { name: text.adminApply, exact: true }).isDisabled(), true)
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, 'no horizontal overflow')
    } finally { await page.close() }
  })
}
