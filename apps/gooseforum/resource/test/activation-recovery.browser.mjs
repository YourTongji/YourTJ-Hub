import assert from 'node:assert/strict'
import { before,after,test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdtemp,readFile,mkdir,rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { execFileSync } from 'node:child_process'
import { createServer } from 'node:http'
import { chromium } from 'playwright'
let browser,server,origin,fixtures
before(async()=>{
 fixtures=await mkdtemp(join(tmpdir(),'yourtj-activation-recovery-'))
 execFileSync('go',['test','./app/http/controllers','-run','TestActivationRecoveryBrowserFixtures','-count=1'],{cwd:fileURLToPath(new URL('../../',import.meta.url)),env:{...process.env,YOURTJ_TEST_HTML_DIR:fixtures},stdio:'pipe'})
 server=createServer(async(req,res)=>{
  const name=new URL(req.url,'http://local').pathname.slice(1)
  if(!/^(missing|invalid|expired)-(zh|en|ja|de)\.html$/.test(name)){res.writeHead(404);res.end();return}
  res.writeHead(200,{'Content-Type':'text/html; charset=utf-8','Cache-Control':'private, no-store','Referrer-Policy':'no-referrer'})
  res.end(await readFile(join(fixtures,name)))
 })
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve))
 origin=`http://127.0.0.1:${server.address().port}`
 browser=await chromium.launch()
})
after(async()=>{await browser?.close();if(server)await new Promise(resolve=>server.close(resolve));if(fixtures)await rm(fixtures,{recursive:true,force:true})})
for(const lang of ['zh','en','ja','de']) for(const state of ['missing','invalid','expired']) for(const width of [320,1280]) test(`activation recovery ${lang} ${state} ${width}`,async()=>{
 const page=await browser.newPage({viewport:{width,height:900}})
 try{
 await page.goto(`${origin}/${state}-${lang}.html`)
 assert.equal(await page.locator('html').getAttribute('lang'),lang)
 assert.equal(await page.locator('a.button.primary').getAttribute('href'),'/settings?tab=profile')
 assert.ok((await page.locator('a[href^="mailto:"]').getAttribute('href')).includes('help@campus.example'))
 assert.equal(await page.locator('a[href*="gooseforum.online"]').count(),0)
 assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
 const recovery=page.locator('a.button.primary');await recovery.focus()
 assert.equal(await recovery.evaluate(el=>el===document.activeElement),true)
 const box=await recovery.boundingBox();assert.ok(box&&box.height>=44&&box.x>=0&&box.x+box.width<=width)
 if(process.env.YOURTJ_TEST_ARTIFACTS){await mkdir(process.env.YOURTJ_TEST_ARTIFACTS,{recursive:true});await page.screenshot({path:`${process.env.YOURTJ_TEST_ARTIFACTS}/activation-${lang}-${state}-${width}.png`,fullPage:true})}
 }finally{await page.close()}
})
