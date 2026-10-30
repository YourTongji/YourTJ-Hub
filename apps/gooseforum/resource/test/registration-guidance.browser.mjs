import assert from 'node:assert/strict'
import { after,before,test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdir } from 'node:fs/promises'
import { chromium } from 'playwright'
import { createServer } from 'vite'
const baseline=process.env.YOURTJ_GUIDANCE_BASELINE==='1'
let server,browser,origin
before(async()=>{
 server=await createServer({root:process.env.YOURTJ_WEB_ROOT||fileURLToPath(new URL('../',import.meta.url)),server:{host:'127.0.0.1',port:0,open:false}})
 await server.listen();origin=`http://127.0.0.1:${server.httpServer.address().port}`;browser=await chromium.launch()
 if(process.env.YOURTJ_TEST_ARTIFACTS)await mkdir(process.env.YOURTJ_TEST_ARTIFACTS,{recursive:true})
})
after(async()=>{await browser?.close();await server?.close()})
for(const lang of ['zh','en','ja','de'])for(const theme of ['gf-light','gf-dark'])for(const width of baseline?[320]:[320,1280])for(const scale of baseline?[2]:[1,2]) {
 test(`registration guidance ${lang}/${theme}/${width}/${scale}`,async()=>{
  const page=await browser.newPage({viewport:{width,height:950}});const errors=[],writes=[]
  page.on('pageerror',error=>errors.push(error.message))
  await page.route('**/api/**',route=>{
   if(route.request().method()!=='GET')writes.push(route.request().url())
   const result=route.request().url().includes('get-captcha')?{captchaRequired:false,captchaId:'',captchaImg:''}:{enabled:false}
   return route.fulfill({json:{code:0,result}})
  })
  await page.goto(`${origin}/assets/test/fixtures/browser/tongji-login.html?lang=${lang}&theme=${theme}&mode=register`)
  await page.locator('[data-tongji-login]:visible').first().waitFor()
  await page.addStyleTag({content:`html{font-size:${scale*100}%}`})
  if(!baseline)for(const [field,hint]of [['input[autocomplete="username"]','#register-username-rules'],['input[autocomplete="new-password"]','#register-password-rules']]) {
   const input=page.locator(field).first(),description=page.locator(hint)
   assert.equal(await input.getAttribute('aria-describedby'),hint.slice(1))
   assert.equal(await input.inputValue(),'')
   await description.scrollIntoViewIfNeeded()
   assert.ok(await description.isVisible())
   const box=await description.boundingBox();assert.ok(box.x>=0&&box.x+box.width<=width+1)
  }
  if(!baseline)assert.deepEqual(await page.locator('.gf-button:visible,.gf-segmented-item:visible').evaluateAll(elements=>elements.filter(e=>e.scrollHeight>e.clientHeight+1).map(e=>e.textContent.trim())),[])
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
  assert.deepEqual(errors,[]);assert.deepEqual(writes,[])
  if(process.env.YOURTJ_TEST_ARTIFACTS)await page.screenshot({path:`${process.env.YOURTJ_TEST_ARTIFACTS}/web-${baseline?'before':'after'}-${lang}-${theme}-${width}-${scale}.png`,fullPage:true})
  await page.close()
 })
}
