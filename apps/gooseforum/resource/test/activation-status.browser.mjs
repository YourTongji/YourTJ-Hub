import assert from 'node:assert/strict'
import { before, after, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdir } from 'node:fs/promises'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server, browser, origin
before(async () => {server = await createServer({root:fileURLToPath(new URL('../',import.meta.url)),server:{host:'127.0.0.1',port:0,open:false}});await server.listen();origin=`http://127.0.0.1:${server.httpServer.address().port}`;browser=await chromium.launch()})
after(async()=>{await browser?.close();await server?.close()})
for(const width of [320,1280]) for(const theme of ['light','dark']) for(const locale of ['zh','de']) test(`activation ${width} ${theme} ${locale}`,async()=>{
 const page=await browser.newPage({viewport:{width,height:844}})
 let state='ready',posts=0,reads=0
 try {
 await page.route('**/api/resend-activation-email',route=>{
  if(route.request().method()==='POST'){posts++;state='cooldown';return route.fulfill({json:{code:0,result:{remainingToday:2},messageCode:'auth.activation.resendSuccess',params:{remainingToday:2}}})}
  reads++;return route.fulfill({json:{code:0,result:{currentEmail:'owner@example.test',pendingEmail:state==='verified'?'':'pending-owner-with-long-name@example.test',targetEmail:state==='verified'?'':'pending-owner-with-long-name@example.test',targetKind:state==='verified'?'none':'pending',state,remainingToday:state==='ready'?3:2,dailyLimit:3,retryAfterSeconds:state==='cooldown'?60:0}}})
 })
 await page.goto(`${origin}/assets/test/fixtures/browser/activation-status.html?theme=${theme}&locale=${locale}`)
 const action=page.locator('[data-testid="activation-email-action"]')
 await action.getByText('pending-owner-with-long-name@example.test',{exact:false}).waitFor()
 assert.equal(posts,0)
 const send=action.getByRole('button').nth(0)
 await send.click()
 await page.waitForFunction(()=>document.querySelector('[data-testid="activation-email-action"] button')?.disabled)
 await action.getByText(/2 \/ 3/).waitFor()
 assert.equal(posts,1);assert.ok(reads>=2)
 assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
 state='verified'
 await action.getByRole('button').nth(1).click()
 await page.waitForFunction(()=>!document.querySelectorAll('[data-testid="activation-email-action"] button')[1]?.disabled)
 assert.equal(await send.isDisabled(),true);assert.equal(posts,1)
 if(process.env.YOURTJ_TEST_ARTIFACTS){await mkdir(process.env.YOURTJ_TEST_ARTIFACTS,{recursive:true});await page.screenshot({path:`${process.env.YOURTJ_TEST_ARTIFACTS}/activation-${width}-${theme}-${locale}.png`,fullPage:true})}
 }finally{await page.close()}
})
