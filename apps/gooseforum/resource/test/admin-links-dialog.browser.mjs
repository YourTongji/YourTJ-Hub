import assert from 'node:assert/strict'
import {after,before,test} from 'node:test'
import {mkdir} from 'node:fs/promises'
import {fileURLToPath} from 'node:url'
import {chromium} from 'playwright'
import {createServer} from 'vite'
let server,browser,origin
before(async()=>{server=await createServer({root:fileURLToPath(new URL('../',import.meta.url)),server:{host:'127.0.0.1',port:0,open:false}});await server.listen();origin=`http://127.0.0.1:${server.httpServer.address().port}`;browser=await chromium.launch()})
after(async()=>{await browser?.close();await server?.close()})
for(const lang of ['zh','en','ja','de']) for(const theme of ['gf-light','gf-dark']) for(const kind of ['link','group']) {
 test(`short-viewport ${kind} dialog fits ${lang}/${theme}`,async()=>{
  const page=await browser.newPage({viewport:{width:402,height:353}})
  try {
   await page.route('**/api/admin/friend-links',route=>route.fulfill({json:{code:0,result:[{name:'Acceptance',emoji:'',color:'#3b82f6',links:[]}]}}))
   await page.goto(`${origin}/assets/test/fixtures/browser/admin-links.html?lang=${lang}&theme=${theme}`)
   await page.locator('#app[data-ready="true"]').waitFor()
   if(kind==='link')await page.getByTestId('admin-link-add').click()
   else await page.getByRole('button',{name:await page.locator('#app').getAttribute('data-add-group-label'),exact:true}).click()
   const dialog=page.getByRole('dialog');await dialog.waitFor();await page.waitForTimeout(250)
   const r=await dialog.boundingBox();assert.ok(r.y>=0&&r.y+r.height<=353.1,JSON.stringify(r))
   const submit=dialog.locator('button[type=submit]');await submit.scrollIntoViewIfNeeded()
   const b=await submit.boundingBox();assert.ok(b.y>=0&&b.y+b.height<=353.1,JSON.stringify(b))
   await dialog.locator('input').first().scrollIntoViewIfNeeded()
   if(process.env.YOURTJ_REVIEW_SCREENSHOTS){await mkdir(process.env.YOURTJ_REVIEW_SCREENSHOTS,{recursive:true});await page.screenshot({path:`${process.env.YOURTJ_REVIEW_SCREENSHOTS}/${kind}-${lang}-${theme}-short.png`})}
  }finally{await page.close()}
 })
}
