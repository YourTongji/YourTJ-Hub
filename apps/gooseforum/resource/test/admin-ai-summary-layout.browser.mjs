import assert from 'node:assert/strict'
import {after,before,test} from 'node:test'
import {mkdir} from 'node:fs/promises'
import {fileURLToPath} from 'node:url'
import {chromium} from 'playwright'
import {createServer} from 'vite'
let server,browser,origin
before(async()=>{server=await createServer({root:fileURLToPath(new URL('../',import.meta.url)),server:{host:'127.0.0.1',port:0,open:false}});await server.listen();origin=`http://127.0.0.1:${server.httpServer.address().port}`;browser=await chromium.launch()})
after(async()=>{await browser?.close();await server?.close()})
for(const lang of ['zh','en','ja','de'])for(const theme of ['gf-light','gf-dark']) {
 test(`AI model controls fit 320px ${lang}/${theme}`,async()=>{
  const page=await browser.newPage({viewport:{width:320,height:900}})
  try {
   await page.route('**/api/admin/ai-summary-settings',route=>route.fulfill({json:{code:0,result:{enabled:true,baseUrl:'https://example.invalid/v1',model:'qa',maxTokens:200,timeoutSeconds:30}}}))
   let requests=0;await page.route('**/api/admin/ai-summary-models',route=>{requests++;return route.fulfill({json:{code:0,result:{models:[{id:'qa'}]}}})})
   await page.goto(`${origin}/assets/test/fixtures/browser/admin-ai-summary.html?lang=${lang}&theme=${theme}`)
   await page.locator('#app[data-ready="true"]').waitFor();await page.getByRole('combobox').waitFor();await page.evaluate(()=>document.fonts.ready)
   assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1))
   const refresh=page.getByRole('button',{name:await page.locator('#app').getAttribute('data-refresh-label'),exact:true});const before=requests;await refresh.click();await page.waitForFunction(()=>document.querySelector('button svg.animate-spin')===null);assert.ok(requests>before)
   if(process.env.YOURTJ_REVIEW_SCREENSHOTS){await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS,{recursive:true});await page.screenshot({path:`${process.env.YOURTJ_REVIEW_SCREENSHOTS}/ai-model-${lang}-${theme}.png`,fullPage:true})}
  }finally{await page.close()}
 })
}
