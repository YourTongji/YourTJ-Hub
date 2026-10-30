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

for (const lang of ['zh', 'en', 'ja', 'de']) for (const width of [320, 1280]) for (const theme of ['gf-light', 'gf-dark']) {
 test(`created school account recovery ${lang} ${width} ${theme}`, async () => {
  const page = await browser.newPage({ viewport: { width, height: 950 } })
  const errors = []; let submissions = 0
  page.on('pageerror', e => errors.push(e.message))
  try {
   await page.route('**/api/auth/tongji/registration', async route => {
    if (route.request().method() === 'GET') await route.fulfill({ json: { code: 0, result: { csrfToken:'synthetic-proof', email:'1234567@tongji.edu.cn', expiresAt:'2030-01-01T00:00:00Z' } } })
    else {
     submissions++
     assert.equal(route.request().postDataJSON().csrfToken, 'synthetic-proof')
     await route.fulfill({ status:500, json: { code:1, result:null, messageCode:'auth.tongji.retryLogin', params:{ restartUrl:'/api/auth/tongji?redirect=%2Fcampus' } } })
    }
   })
   await page.goto(`${origin}/assets/test/fixtures/browser/tongji-login.html?schoolRegistration=true&lang=${lang}&theme=${theme}`)
   await page.locator('input[autocomplete=username]').fill('new_student')
   const passwords = page.locator('input[autocomplete=new-password]')
   await passwords.nth(0).fill('Password123'); await passwords.nth(1).fill('Password123')
   await page.locator('input[type=checkbox]').check()
   await page.locator('button[type=submit]').click()
   const alert = page.locator('[role=alert]'); await alert.waitFor()
   assert.equal(await page.locator('form').count(), 0)
   assert.equal(submissions, 1)
   const restart = page.locator('a[href^="/api/auth/tongji?"]')
   assert.equal(await restart.getAttribute('href'), '/api/auth/tongji?redirect=%2Fcampus')
   await restart.focus()
   assert.equal(await restart.evaluate(e => document.activeElement === e), true)
   const box = await restart.boundingBox()
   assert.ok(box && box.height >= 40 && box.x >= 0 && box.x + box.width <= width + 1)
   assert.ok((await alert.textContent()).length > 20)
   assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true)
   assert.deepEqual(errors, [])
   if (process.env.YOURTJ_TEST_ARTIFACTS) {
    await page.screenshot({path:`${process.env.YOURTJ_TEST_ARTIFACTS}/school-${lang}-${width}-${theme}.png`})
   }
  } finally { await page.close() }
 })
}
