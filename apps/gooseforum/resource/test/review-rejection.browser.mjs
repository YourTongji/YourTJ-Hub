import assert from 'node:assert/strict'
import { before, after, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdir } from 'node:fs/promises'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), optimizeDeps: { entries: ['test/fixtures/browser/review-rejection.html'] }, server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })
for (const mode of ['moderator', 'admin', 'quick', 'owner']) for (const lang of ['zh', 'en', 'ja', 'de']) for (const theme of ['light', 'dark']) for (const width of [320, 1280]) for (const scale of mode !== 'owner' && lang === 'de' && theme === 'dark' && width === 320 ? [1, 2] : [1]) {
  test(`${mode} rejection recovery ${lang}/${theme}/${width}/${scale}x`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    page.setDefaultTimeout(10000)
    const errors = [], unexpected = []
    page.on('pageerror', error => errors.push(error.message))
    let writes = 0, previews = 0, fail = true, decided = false
    const item = { id: 7, revisionId: 19, title: 'Candidate', content: 'Private candidate', excerpt: 'Candidate excerpt', userId: 9, username: 'author', processStatus: 2, createdAt: '2026-10-30T00:00:00Z', images: ['/fixture/preview.svg'] }
    const reason = 'Please remove the private address. <script>window.reasonExecuted=true</script>'
    const ready = { state: 'ready', subject: 'review.topic', action: 'reject', targetType: 'topic', targetId: 7, title: 'Candidate', excerpt: 'Private candidate', workbenchUrl: '/moderation' }
    try {
      await page.route('**/fixture/preview.svg', route => route.fulfill({ contentType: 'image/svg+xml', body: '<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40"><rect width="40" height="40" fill="blue"/></svg>' }))
      await page.route('**/api/**', async route => {
        const request = route.request(), path = new URL(request.url()).pathname
        if (path.endsWith('/review-queue')) return route.fulfill({ json: { code: 0, result: { items: decided ? [] : [item], total: decided ? 0 : 1, page: 1, pageSize: 10 } } })
        if (path.endsWith('/approval-action/preview')) { previews++; return route.fulfill({ json: { code: 0, result: ready } }) }
        if (path.endsWith('/content/appeals')) return route.fulfill({ json: { code: 0, result: { items: [] } } })
        if (path.endsWith('/review-action') || path.endsWith('/approval-action/execute')) {
          writes++
          const body = request.postDataJSON()
          assert.deepEqual(body, mode === 'quick' ? { token: 'synthetic-action', reason } : { kind: 'topic', id: 7, revisionId: 19, approve: false, reason })
          if (fail) { fail = false; return mode === 'quick' ? route.fulfill({json:{code:0,result:{...ready,state:'failed'}}}) : route.fulfill({ status: 500, json: { code: 1, messageCode: 'common.operation.failed', result: null } }) }
          decided = true
          return route.fulfill({ json: { code: 0, result: mode === 'quick' ? { ...ready, state: 'done' } : 'success' } })
        }
        unexpected.push(`${request.method()} ${path}`)
        return route.abort()
      })
      await page.goto(`${origin}/assets/test/fixtures/browser/review-rejection.html?mode=${mode}&lang=${lang}&theme=${theme}&token=synthetic-action&scale=${scale}`)
      await page.waitForFunction(() => Boolean(window.reviewLabels) || Boolean(document.querySelector('vite-error-overlay')))
      assert.equal(await page.locator('vite-error-overlay').count(), 0, errors.join('\n'))
      const labels = await page.evaluate(() => window.reviewLabels)
      const baselineWidth = await page.evaluate(() => document.documentElement.scrollWidth)
      if (mode === 'owner') {
        await page.getByText(reason, { exact: true }).waitFor()
        assert.equal(await page.evaluate(() => window.reasonExecuted), undefined)
        assert.equal(writes, 0)
        if (process.env.YOURTJ_TEST_ARTIFACTS) {
          await mkdir(process.env.YOURTJ_TEST_ARTIFACTS, { recursive: true })
          await page.screenshot({ path: `${process.env.YOURTJ_TEST_ARTIFACTS}/rejection-${mode}-${lang}-${theme}-${width}-${scale}x.png`, fullPage: true })
        }
      } else {
        if (mode !== 'quick') await page.getByRole('button', { name: mode === 'admin' ? labels.adminReject : labels.reject, exact: true }).first().click()
        const select = page.locator('select').last(), note = page.locator('textarea').last()
        await select.waitFor()
        const confirm = () => page.getByRole('button', { name: mode === 'quick' ? labels.quickConfirm : labels.confirm, exact: true })
        await confirm().click()
        assert.equal(writes, 0)
        await select.selectOption('other')
        await note.fill('x'.repeat(513))
        await confirm().click()
        assert.equal(writes, 0)
        await note.fill(reason)
        if (mode === 'moderator') {
          await page.locator('button').filter({ has: page.locator('img') }).click()
          await page.getByRole('dialog').waitFor()
          await page.keyboard.press('Escape')
          await page.getByRole('dialog').waitFor({ state: 'hidden' })
          assert.equal(await note.inputValue(), reason)
          assert.equal(await select.inputValue(), 'other')
        }
        if (mode !== 'quick') {
          await page.getByRole('button', { name: labels.cancel, exact: true }).click()
          assert.equal(writes, 0)
          await page.getByRole('button', { name: mode === 'admin' ? labels.adminReject : labels.reject, exact: true }).first().click()
          assert.equal(await note.inputValue(), '')
          await select.selectOption('other')
          await note.fill(reason)
        }
        await confirm().click()
        await page.waitForFunction(() => [...document.querySelectorAll('button')].every(button => !button.disabled || button.type !== 'submit'))
        await page.waitForTimeout(50)
        assert.equal(writes, 1)
        assert.equal(await note.inputValue(), reason)
        assert.equal(await select.inputValue(), 'other')
        if (mode === 'quick') {
          const recheck = page.getByTestId('moderation-recheck')
          await recheck.waitFor()
          assert.ok(await recheck.evaluate(el => el.scrollHeight <= el.clientHeight + 1), 'recheck label fits at large text')
          assert.equal(await note.isDisabled(), true)
          assert.equal(previews, 1)
          if (process.env.YOURTJ_TEST_ARTIFACTS) {
            await mkdir(process.env.YOURTJ_TEST_ARTIFACTS, { recursive: true })
            await page.screenshot({ path: `${process.env.YOURTJ_TEST_ARTIFACTS}/failed-quick-${lang}-${theme}-${width}-${scale}x.png`, fullPage: true })
          }
          await page.getByTestId('moderation-recheck').click()
          await confirm().waitFor()
          assert.equal(previews, 2)
          assert.equal(writes, 1, 'rechecking never executes a decision')
          assert.equal(await note.inputValue(), reason)
        }
        assert.equal(await page.evaluate(() => window.reasonExecuted), undefined)
        if (mode === 'admin') {
          const bg = await page.getByRole('dialog').evaluate(el => getComputedStyle(el).backgroundColor)
          assert.notEqual(bg, 'rgba(0, 0, 0, 0)', 'admin uses its own opaque dialog theme')
        }
        assert.ok(await page.evaluate(base => document.documentElement.scrollWidth <= Math.max(innerWidth, base), baselineWidth), 'the form does not increase horizontal overflow')
        assert.ok(await confirm().evaluate(el => el.scrollWidth <= el.clientWidth), 'confirmation label stays inside its button')
        assert.ok(await confirm().evaluate(el => el.scrollHeight <= el.clientHeight + 1), 'confirmation text does not overflow the button height')
        if (mode === 'quick') assert.ok(await page.locator('a[href="/moderation"]').last().evaluate(el => el.scrollHeight <= el.clientHeight + 1), 'workbench link text stays inside its action')
        assert.ok(await page.locator('textarea').evaluate(el => { const fields = el.closest('[data-test="rejection-reason-fields"]'); return fields.scrollWidth <= fields.clientWidth }), 'reason labels and hints stay inside the form')
        if (process.env.YOURTJ_TEST_ARTIFACTS) {
          await mkdir(process.env.YOURTJ_TEST_ARTIFACTS, { recursive: true })
          await page.screenshot({ path: `${process.env.YOURTJ_TEST_ARTIFACTS}/rejection-${mode}-${lang}-${theme}-${width}-${scale}x.png`, fullPage: true })
        }
        await confirm().click()
        await page.waitForFunction(() => !document.querySelector('textarea'))
        assert.equal(writes, 2)
      }
      assert.deepEqual(unexpected, [])
      assert.deepEqual(errors, [])
    } finally { await page.close() }
  })
}
