import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { mkdir } from 'node:fs/promises'
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
for (const lang of ['zh', 'en', 'ja', 'de']) for (const width of [320, 1280]) {
 test(`existing Feishu report endpoint accepts signup alerts: ${lang} ${width}`, async () => {
  const page = await browser.newPage({ viewport: { width, height: 900 } })
  const endpointId = '11111111-1111-4111-8111-111111111111'
  let settings = { enabled: true, endpoints: [{ id: endpointId, name: 'Existing report channel', channelType: 'feishu', enabled: true, url: '', urlConfigured: true, secret: '', secretConfigured: true, events: ['moderation.report.post.created'], timeoutSeconds: 2 }] }
  let writes = 0
  let submitted
  const unexpected = []
  try {
   await page.route('https://open.feishu.cn/**', route => { unexpected.push(route.request().url()); return route.abort() })
   await page.route('**/api/admin/http-notify-settings', route => route.fulfill({ json: { code: 0, result: settings } }))
   await page.route('**/api/admin/save-http-notify-settings', async route => {
    writes++
    submitted = route.request().postDataJSON().settings
    // The API preserves stored credentials by endpoint ID. Configured flags
    // are read-only response metadata and intentionally absent from writes.
    settings = { ...submitted, endpoints: submitted.endpoints.map(endpoint => ({ ...endpoint, secretConfigured: true, urlConfigured: true })) }
    await route.fulfill({ json: { code: 0, result: true } })
   })
   await page.goto(`${origin}/assets/test/fixtures/browser/registration-notify.html?lang=${lang}&dark=${width === 320}`)
   const header = page.locator(`[aria-controls="http-endpoint-${endpointId}"]`)
   await header.waitFor()
   await header.click()
   const panel = page.locator(`#http-endpoint-${endpointId}`)
   const signupLabel = await page.locator('#app').getAttribute('data-signup-label')
   const signup = panel.getByRole('checkbox', { name: signupLabel, exact: true })
   await signup.waitFor()
   assert.equal(await signup.getAttribute('aria-checked'), 'false')
   await signup.click()
   assert.equal(await signup.getAttribute('aria-checked'), 'true')
   const hintText = await page.locator('#app').getAttribute('data-feishu-hint')
   const hint = panel.getByText(hintText, { exact: true })
   await hint.scrollIntoViewIfNeeded()
   const box = await hint.boundingBox()
   assert.ok(box.x >= 0 && box.x + box.width <= width)
   if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
    await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
    await hint.locator('..').screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/signup-webhook-${lang}-${width}.png` })
   }
   const saveLabel = await page.locator('#app').getAttribute('data-save-label')
   await Promise.all([page.waitForResponse('**/api/admin/save-http-notify-settings'), page.getByRole('button', { name: saveLabel, exact: true }).click()])
   assert.equal(writes, 1)
   assert.equal(settings.endpoints[0].id, endpointId)
   assert.equal(settings.endpoints[0].channelType, 'feishu')
   assert.deepEqual(settings.endpoints[0].events.sort(), ['moderation.report.post.created', 'user.signup'])
   assert.equal(settings.endpoints[0].secret, '')
   assert.ok(!('secretConfigured' in submitted.endpoints[0]))
   assert.ok(!('urlConfigured' in submitted.endpoints[0]))
   await page.reload()
   await header.waitFor()
   await header.click()
   await signup.waitFor()
   assert.equal(await signup.getAttribute('aria-checked'), 'true')
   assert.equal(writes, 1)
   assert.deepEqual(unexpected, [])
  } finally { await page.close() }
 })
 test(`mail switch communicates and preserves the selected policy: ${lang} ${width}`, async () => {
  const page = await browser.newPage({ viewport: { width, height: 900 } })
  let settings = { enableSignup: true, enableEmailVerification: false, maxDailySignups: -1, allowedDomains: [], reservedUsernames: [], bannedUsernames: [], sensitiveWords: [], sensitiveAction: 'block', captchaRequired: true }
  let writes = 0
  try {
   await page.route('**/api/admin/security-settings', route => route.fulfill({ json: { code: 0, result: settings } }))
   await page.route('**/api/admin/save-security-settings', async route => { writes++; settings = route.request().postDataJSON().settings; await route.fulfill({ json: { code: 0, result: true } }) })
   await page.goto(`${origin}/assets/test/fixtures/browser/registration-notify.html?kind=security&lang=${lang}&dark=${width === 320}`)
   await page.locator('#app[data-ready="true"]').waitFor()
   const hintText = await page.locator('#app').getAttribute('data-verification-hint')
   const hint = page.getByText(hintText, { exact: true })
   await hint.waitFor()
   const section = hint.locator('../..')
   const toggle = section.getByRole('switch')
   assert.equal(await toggle.getAttribute('aria-checked'), 'false')
   await hint.scrollIntoViewIfNeeded()
   const box = await hint.boundingBox()
   assert.ok(box.x >= 0 && box.x + box.width <= width)
   if (process.env.YOURTJ_REVIEW_SCREENSHOTS) {
    await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS, { recursive: true })
    await section.screenshot({ path: `${process.env.YOURTJ_REVIEW_SCREENSHOTS}/mail-switch-${lang}-${width}.png` })
   }
   const label = await page.locator('#app').getAttribute('data-save-label')
   await toggle.click()
   await Promise.all([page.waitForResponse('**/api/admin/save-security-settings'), page.getByRole('button', { name: label, exact: true }).click()])
   assert.equal(settings.enableEmailVerification, true)
   await toggle.click()
   await Promise.all([page.waitForResponse('**/api/admin/save-security-settings'), page.getByRole('button', { name: label, exact: true }).click()])
   assert.equal(settings.enableEmailVerification, false)
   assert.equal(writes, 2)
  } finally { await page.close() }
 })
}
