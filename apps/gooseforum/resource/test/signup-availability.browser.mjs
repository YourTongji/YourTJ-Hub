import assert from 'node:assert/strict'
import { before,after,test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdir } from 'node:fs/promises'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server,browser,origin
before(async()=>{server=await createServer({root:fileURLToPath(new URL('../',import.meta.url)),server:{host:'127.0.0.1',port:0,open:false}});await server.listen();origin=`http://127.0.0.1:${server.httpServer.address().port}`;browser=await chromium.launch()})
after(async()=>{await browser?.close();await server?.close()})
for(const width of [320,1280]) for(const lang of ['zh','en','ja','de']) for(const theme of ['gf-light','gf-dark']) for(const state of ['disabled','daily_limit','mail_unavailable','unavailable']) test(`signup ${width} ${lang} ${theme} ${state}`,async()=>{
 const page=await browser.newPage({viewport:{width,height:950}})
 let writes=0
 try{
 await page.route('**/api/**',route=>{
  if(route.request().method()==='POST')writes++
  return route.fulfill({json:{code:0,result:{enabled:false,captchaId:'fixture',captchaImg:'data:image/png;base64,x'}}})
 })
 const url=`${origin}/assets/test/fixtures/browser/tongji-login.html?mode=register&lang=${lang}&theme=${theme}&signup=${state}`
 await page.goto(url)
 const unavailable=page.locator('[data-testid="signup-availability"]')
 await unavailable.waitFor()
 assert.equal(await page.locator('form').count(),0)
 assert.equal(await page.locator('input[type="password"]').count(),0)
 assert.equal(writes,0)
 assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
 if(process.env.YOURTJ_TEST_ARTIFACTS){await mkdir(process.env.YOURTJ_TEST_ARTIFACTS,{recursive:true});await page.screenshot({path:`${process.env.YOURTJ_TEST_ARTIFACTS}/signup-${width}-${lang}-${theme}-${state}.png`,fullPage:true})}
 await Promise.all([page.waitForNavigation(),unavailable.getByRole('button').click()])
 await unavailable.waitFor()
 await page.locator('.gf-segmented-item').nth(0).click()
 await page.locator('form input[autocomplete="username"]').waitFor()
 assert.equal(writes,0)
 }finally{await page.close()}
})
