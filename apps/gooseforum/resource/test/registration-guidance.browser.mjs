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
  await page.route('**/static/pic/brand-default*.webp', route => {
   const name=new URL(route.request().url()).pathname.split('/').pop()
   assert.ok(['brand-default.webp','brand-default-dark.webp'].includes(name))
   return route.fulfill({path:fileURLToPath(new URL(`../static/pic/${name}`,import.meta.url)),contentType:'image/webp'})
  })
  await page.route('**/api/**',route=>{
   if(route.request().method()!=='GET')writes.push(route.request().url())
   const result=route.request().url().includes('get-captcha')?{captchaRequired:false,captchaId:'',captchaImg:''}:{enabled:false}
   return route.fulfill({json:{code:0,result}})
  })
  await page.goto(`${origin}/assets/test/fixtures/browser/tongji-login.html?lang=${lang}&theme=${theme}&mode=register&captcha=off&domains=${theme==='gf-dark'?'multiple':'single'}`)
  await page.locator('[data-tongji-login]:visible').first().waitFor()
  // Finish the real theme view transition before changing text scale or
  // scrolling. Its captured root otherwise becomes misleading visual evidence.
  await page.waitForFunction(() => !document.documentElement.classList.contains('theme-switching'))
  await page.addStyleTag({content:`html{font-size:${scale*100}%}`})
  if(!baseline)for(const [field,hint]of [['input[autocomplete="username"]','#register-username-rules'],['input[autocomplete="new-password"]','#register-password-rules']]) {
   const input=page.locator(field).first(),description=page.locator(hint)
   assert.equal(await input.getAttribute('aria-describedby'),hint.slice(1))
   assert.equal(await input.inputValue(),'')
   await description.scrollIntoViewIfNeeded()
   assert.ok(await description.isVisible())
   const box=await description.boundingBox();assert.ok(box.x>=0&&box.x+box.width<=width+1)
  }
  if(!baseline && scale===2 && width===320) {
   const copy=page.locator('.login-column img[src*="brand-default"]').locator('..')
   const box=await copy.boundingBox()
   const font=await copy.locator('p').first().evaluate(e=>parseFloat(getComputedStyle(e).fontSize))
   assert.ok(box.width>=Math.min(width*.7,font*6),`brand copy squeezed to ${box.width}px at 200% text`)
  }
  if(!baseline) {
   const room=await page.locator('#register-email').evaluate(e=>{
    const style=getComputedStyle(e)
    return {text:e.clientWidth-parseFloat(style.paddingLeft)-parseFloat(style.paddingRight),font:parseFloat(style.fontSize)}
   })
   assert.ok(room.text>=room.font*4,`email prefix has only ${room.text}px of editable text space`)
  }
  if(!baseline)assert.deepEqual(await page.locator('.gf-button:visible,.gf-segmented-item:visible').evaluateAll(elements=>elements.filter(e=>e.scrollHeight>e.clientHeight+1).map(e=>e.textContent.trim())),[])
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
  assert.deepEqual(errors,[]);assert.deepEqual(writes,[])
  await page.evaluate(() => document.fonts.ready)
  await page.waitForFunction(() => [...document.images].every(image => image.complete && image.naturalWidth > 0))
  if(process.env.YOURTJ_TEST_ARTIFACTS)await page.screenshot({path:`${process.env.YOURTJ_TEST_ARTIFACTS}/web-${baseline?'before':'after'}-${lang}-${theme}-${width}-${scale}.png`,fullPage:true})
  if(!baseline && width===320 && scale===2) {
   if(process.env.YOURTJ_TEST_ARTIFACTS)await page.screenshot({path:`${process.env.YOURTJ_TEST_ARTIFACTS}/web-fields-${lang}-${theme}.png`})
   await page.locator('.gf-segmented-item').first().click()
   await page.locator('input[autocomplete="current-password"]').waitFor()
   const copy=page.locator('.login-column img[src*="brand-default"]').locator('..')
   const box=await copy.boundingBox()
   const font=await copy.locator('p').first().evaluate(e=>parseFloat(getComputedStyle(e).fontSize))
   assert.ok(box.width>=Math.min(width*.7,font*6))
   assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
  }
  await page.close()
 })
}
