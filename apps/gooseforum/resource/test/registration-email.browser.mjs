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

for (const width of [320, 1280]) for (const theme of ['gf-light', 'gf-dark']) for (const domains of ['single', 'multiple']) {
  test(`full email paste and recovery: ${width}px, ${theme}, ${domains}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 950 } })
    const emails = []
    try {
      await page.route('**/api/**', route => route.fulfill({ json: { code: 0, result: { enabled: false } } }))
      await page.route('**/api/register', route => {
        emails.push(route.request().postDataJSON().email)
        return route.fulfill({ json: { code: 1, result: null, messageCode: 'auth.register.failed' } })
      })
      await page.goto(`${origin}/assets/test/fixtures/browser/tongji-login.html?lang=en&mode=register&captcha=off&domains=${domains}&theme=${theme}`)
      await page.locator('html[data-ready="true"]').waitFor()
      await page.locator('input[autocomplete="username"]').fill('new-user')
      const domain = domains === 'multiple' ? 's.tongji.edu.cn' : 'tongji.edu.cn'
      await page.locator('#register-email').fill(`  student@${domain.toUpperCase()}  `)
      for (const field of await page.locator('input[autocomplete="new-password"]').all()) await field.fill('password123')
      await page.getByRole('checkbox').check()
      const submit = page.locator('form button[type="submit"]')
      await Promise.all([page.waitForResponse('**/api/register'), submit.click()])
      await page.waitForFunction(() => document.querySelector('#register-email')?.value === 'student')
      assert.deepEqual(emails, [`student@${domain}`])
      if (domains === 'multiple') assert.match(await page.getByRole('combobox').textContent(), /@s\.tongji\.edu\.cn/)
      await page.locator('#register-email').fill('student@example.test')
      await submit.click()
      await page.getByText('This email domain is not allowed for registration', { exact: true }).waitFor()
      assert.equal(emails.length, 1, 'unlisted full addresses must stay on the form')
      assert.equal(await page.locator('#register-email').inputValue(), 'student@example.test')
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
      if (process.env.YOURTJ_REGISTRATION_SCREENSHOT_DIR) await page.screenshot({ path: `${process.env.YOURTJ_REGISTRATION_SCREENSHOT_DIR}/email-${domains}-${theme}-${width}.png`, fullPage: true })
    } finally { await page.close() }
  })
}
