import assert from 'node:assert/strict'
import { before, after, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdir } from 'node:fs/promises'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server,browser,origin
before(async()=>{server=await createServer({root:fileURLToPath(new URL('../',import.meta.url)),server:{host:'127.0.0.1',port:0,open:false}});await server.listen();origin=`http://127.0.0.1:${server.httpServer.address().port}`;browser=await chromium.launch()})
after(async()=>{await browser?.close();await server?.close()})
const image=(id)=>`data:image/svg+xml;base64,${Buffer.from(`<svg xmlns="http://www.w3.org/2000/svg" width="112" height="40"><rect width="112" height="40" fill="#ddd"/><text x="12" y="29" font-size="24" fill="#222">${id}421</text></svg>`).toString('base64')}`
for(const lang of ['zh','en','ja','de']) for(const width of [320,1280]) for(const theme of ['gf-light','gf-dark']) test(`captcha refresh ${lang} ${width} ${theme}`,async()=>{
 const page=await browser.newPage({viewport:{width,height:950}})
 let calls=0,failNext=false,holdNext=false,held
 const errors=[]
 page.on('pageerror',error=>errors.push(error.message))
 const respond=(route,id)=>route.fulfill({json:{code:0,result:{captchaId:`challenge-${id}`,captchaImg:image(id)}}})
 try{
 await page.route('**/api/auth/passkey/status',route=>route.fulfill({json:{code:0,result:{enabled:false}}}))
 await page.route('**/api/get-captcha*',async route=>{
  const id=++calls
  if(holdNext){holdNext=false;held={route,id};return}
  if(failNext){failNext=false;return route.fulfill({json:{code:1,messageCode:'common.operationFailed',result:null}})}
  return respond(route,id)
 })
 await page.goto(`${origin}/assets/test/fixtures/browser/tongji-login.html?lang=${lang}&theme=${theme}`)
 const captcha=page.locator('img.gf-captcha-image')
 const input=page.locator('form input[aria-label]:not([type="password"])')
 const refresh=page.locator('button[aria-busy]')
 await captcha.waitFor()
 await input.fill('OLD-CODE')
 holdNext=true;await refresh.click()
 await page.waitForFunction(()=>document.querySelector('button[aria-busy]')?.getAttribute('aria-busy')==='true')
 assert.equal(await refresh.isDisabled(),true)
 assert.equal(await page.locator('form button[type="submit"]').first().isDisabled(),true)
 assert.equal(await input.inputValue(),'OLD-CODE')
 while(!held) await new Promise(resolve=>setTimeout(resolve,5))
 await respond(held.route,held.id);held=undefined
 await page.waitForFunction(()=>document.querySelector('button[aria-busy]')?.getAttribute('aria-busy')==='false')
 assert.equal(await input.inputValue(),'')
 await input.fill('KEEP-CODE');failNext=true;await refresh.click()
 await page.locator('#auth-captcha-error').waitFor()
 assert.equal(await input.inputValue(),'KEEP-CODE')
 assert.equal(await captcha.getAttribute('src'),image(2))
 assert.equal(await input.getAttribute('aria-describedby'),'auth-captcha-error')
 assert.ok(await refresh.getAttribute('aria-label'))
 assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
 if(process.env.YOURTJ_TEST_ARTIFACTS){await mkdir(process.env.YOURTJ_TEST_ARTIFACTS,{recursive:true});await page.screenshot({path:`${process.env.YOURTJ_TEST_ARTIFACTS}/captcha-${lang}-${width}-${theme}.png`,fullPage:true})}
 await refresh.click();await page.locator('#auth-captcha-error').waitFor({state:'detached'})
 assert.equal(await input.inputValue(),'')
 holdNext=true;await refresh.click()
 while(!held) await new Promise(resolve=>setTimeout(resolve,5))
 await page.locator('.gf-segmented-item').nth(1).click()
 await page.waitForFunction(expected=>document.querySelector('img.gf-captcha-image')?.getAttribute('src')===expected,image(6))
 await respond(held.route,held.id);held=undefined
 await page.waitForLoadState('networkidle')
 assert.equal(await captcha.getAttribute('src'),image(6))
 assert.deepEqual(errors,[])
 }finally{await page.close()}
})
