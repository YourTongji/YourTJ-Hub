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
for (const width of [320, 768]) {
  test(`names remain readable and confirm explicitly at ${width}px; ambiguous retry uses one batch`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    const state = { persona: null, nameSelectedAt: null, nameChangeAvailableAt: null, disabled: false, governanceDisabled: false, day: '2026-10-06', remaining: 10, resetsAt: '2026-10-06T16:00:00Z', batches: [], lexiconVersion: 'THUOCL-a30ce79' }
    const long = '中华人民共和国道路交通安全法实施条例'
    const requests = [], keys = new Map()
    let confirms = 0, lost = true
    await page.route('**/a/avatar.svg', route => route.fulfill({ contentType: 'image/svg+xml', body: '<svg xmlns="http://www.w3.org/2000/svg" width="80" height="80"><rect width="80" height="80" fill="#D9E5FC"/><circle cx="40" cy="42" r="23" fill="#426DC4"/><path d="M25 44Q40 59 55 44" fill="none" stroke="white" stroke-width="4"/></svg>' }))
    await page.route('**/api/forum/anonymous/*', async route => {
      const path = new URL(route.request().url()).pathname.split('/').at(-1)
      const body = route.request().postDataJSON()
      let result = state
      if (path === 'batches') {
        requests.push(body.requestKey)
        if (!keys.has(body.requestKey)) {
          const batch = { id: 'a'.repeat(32), day: state.day, createdAt: '2026-10-06T00:00:00Z', expiresAt: state.resetsAt, words: [long, 'C++', '人', '数学', '大学', '春天', '上海', '星辰', '同学', '通济'] }
          keys.set(body.requestKey, batch); state.batches.push(batch); state.remaining--
        }
        result = keys.get(body.requestKey)
        if (lost) { lost = false; return route.abort() }
      } else if (path === 'confirm') {
        confirms++
        state.persona = { kind: 'persona', publicUid: 'd'.repeat(32), name: state.batches[0].words[body.index], avatarUrl: '/a/avatar.svg', profileUrl: '/a/' + 'd'.repeat(32) }
        state.nameChangeAvailableAt = '2100-10-06T00:00:00Z'
        state.nameSelectedAt = '2026-10-06T00:00:00Z'; result = state.persona
      } else if (path === 'disable') { state.disabled = body.disabled; result = true }
      await route.fulfill({ json: { code: 0, result } })
    })
    try {
      await page.goto(`${origin}/assets/test/fixtures/browser/anonymous.html`)
      const draw = page.getByRole('button', { name: '随机 10 个花名', exact: true })
      await draw.click()
      await page.getByRole('alert').waitFor()
      await draw.click()
      const name = page.getByRole('button', { name: long, exact: true })
      await name.waitFor()
      assert.equal(requests.length, 2); assert.equal(requests[0], requests[1]); assert.equal(state.remaining, 9)
      assert.ok(await name.evaluate(el => el.scrollWidth <= el.clientWidth), 'complete word must wrap')
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'page must fit')
      if (process.env.YOURTJ_ANONYMOUS_SCREENSHOT_DIR) await page.screenshot({ path: `${process.env.YOURTJ_ANONYMOUS_SCREENSHOT_DIR}/names-${width}.png`, fullPage: true })
      await name.click(); assert.equal(confirms, 0)
      await page.getByRole('button', { name: '确认花名（锁定一年）', exact: true }).click()
      await page.getByRole('link', { name: long, exact: true }).waitFor()
      assert.equal(confirms, 1)
      assert.equal(await draw.count(), 0, 'locked identity must not draw')
      if (process.env.YOURTJ_ANONYMOUS_SCREENSHOT_DIR) await page.screenshot({ path: `${process.env.YOURTJ_ANONYMOUS_SCREENSHOT_DIR}/selected-${width}.png`, fullPage: true })
    } finally { await page.close() }
  })
}
