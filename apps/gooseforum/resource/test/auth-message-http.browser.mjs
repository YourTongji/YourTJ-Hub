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

for (const lang of ['zh','en','ja','de']) for (const width of [320,1280]) for (const theme of ['gf-light','gf-dark']) for (const status of [403,429]) {
 test(`registration HTTP recovery ${status} ${lang} ${width} ${theme}`,async()=>{
  const page=await browser.newPage({viewport:{width,height:950}});let submissions=0
  try {
   const messageCode=status===429?'common.rateLimited':'auth.signupDisabled'
   await page.route('**/api/passkey/status*',r=>r.fulfill({json:{code:0,result:{enabled:false}}}))
   await page.route('**/api/register',r=>{submissions++;return r.fulfill({status,headers:{'Retry-After':'37'},json:{code:1,messageCode,params:status===429?{retryAfterSeconds:12}:undefined,message:'private provider failure'}})})
   await page.goto(`${origin}/assets/test/fixtures/browser/tongji-login.html?mode=register&captcha=off&lang=${lang}&theme=${theme}`)
   const form=page.locator('form')
   await form.locator('input[autocomplete=username]').fill('synthetic_user')
   await form.locator('#register-email').fill('synthetic')
   const passwords=form.locator('input[autocomplete=new-password]')
   await passwords.nth(0).fill('Password123');await passwords.nth(1).fill('Password123')
   await form.locator('input[type=checkbox]').check();await form.locator('button[type=submit]').click()
   const expected=await page.evaluate(async({messageCode,status})=>{
    const {resolveApiMessage}=await import('/assets/src/runtime/api-message.ts')
    return resolveApiMessage({messageCode,params:status===429?{retryAfterSeconds:37}:{}},'missing translation')
   },{messageCode,status})
   const error=page.getByText(expected,{exact:true});await error.waitFor()
   assert.equal(await form.locator('input[autocomplete=username]').inputValue(),'synthetic_user')
   assert.equal(await form.locator('button[type=submit]').isEnabled(),true)
   assert.equal(submissions,1)
   assert.ok(!(await page.locator('body').textContent()).includes('private provider'))
   assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
   if(process.env.YOURTJ_TEST_ARTIFACTS){await mkdir(process.env.YOURTJ_TEST_ARTIFACTS,{recursive:true});await error.scrollIntoViewIfNeeded();await page.screenshot({path:`${process.env.YOURTJ_TEST_ARTIFACTS}/http-${status}-${lang}-${width}-${theme}.png`})}
  }finally{await page.close()}
 })
}
