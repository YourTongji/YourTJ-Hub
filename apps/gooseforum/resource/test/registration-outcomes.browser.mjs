import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { mkdir } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0 } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
  if (process.env.YOURTJ_TEST_ARTIFACTS) await mkdir(process.env.YOURTJ_TEST_ARTIFACTS, { recursive: true })
})
after(async () => { await browser?.close(); await server?.close() })
const phrases = { zh: ['账号已创建', '登录'], en: ['Your account was created', 'sign-in'], ja: ['アカウント', 'ログイン'], de: ['Konto', 'Anmeldung'] }
for (const lang of ['zh', 'en', 'ja', 'de']) for (const width of [320, 1280]) for (const theme of ['gf-light', 'gf-dark']) for (const outcome of ['emailRetry', 'retryLogin']) {
  test(`created-account recovery ${outcome} ${lang} ${width} ${theme}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 950 }, reducedMotion: 'reduce' })
    const errors = []
    let posts = 0
    page.on('pageerror', e => errors.push(e.message))
    try {
      await page.route('**/api/get-captcha*', r => r.fulfill({ json: { code: 0, result: { captchaId: '', captchaImg: '' } } }))
      await page.route('**/api/passkey/status*', r => r.fulfill({ json: { code: 0, result: { enabled: false } } }))
      await page.route('**/api/register', async r => {
        posts++
        assert.equal(r.request().postDataJSON().email, 'synthetic@example.test')
        await r.fulfill({ json: { code: outcome === 'emailRetry' ? 0 : 1, messageCode: `auth.register.${outcome}` } })
      })
      await page.goto(`${origin}/assets/test/fixtures/browser/registration-outcomes.html?lang=${lang}&theme=${theme}`)
      await page.locator('html[data-ready="true"]').waitFor()
      const form = page.locator('form')
      await form.locator('input[autocomplete="username"]').fill('synthetic_user')
      await form.locator('input[type="email"]').fill('synthetic@example.test')
      const passwords = form.locator('input[autocomplete="new-password"]')
      await passwords.nth(0).fill('password123')
      await passwords.nth(1).fill('password123')
      await form.locator('button[type="submit"]').click()
      // Both the real API resolver and the real page/flash renderer must show the localized outcome.
      if (outcome === 'emailRetry') await page.waitForURL(url => url.searchParams.has('landed'))
      const text = page.getByText(new RegExp(phrases[lang][0])).last()
      await text.waitFor()
      assert.ok((await text.textContent()).includes(phrases[lang][0]))
      if (outcome === 'emailRetry') assert.ok(new URL(page.url()).searchParams.has('landed'))
      else assert.ok(!new URL(page.url()).searchParams.has('landed'))
      await text.scrollIntoViewIfNeeded()
      const box = await text.boundingBox()
      assert.ok(box && box.x >= -1 && box.x + box.width <= width + 1)
      assert.equal(posts, 1)
      assert.deepEqual(errors, [])
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
      if (process.env.YOURTJ_TEST_ARTIFACTS) await page.screenshot({ path: `${process.env.YOURTJ_TEST_ARTIFACTS}/web-${outcome}-${lang}-${width}-${theme}.png` })
    } finally { await page.close() }
  })
}
