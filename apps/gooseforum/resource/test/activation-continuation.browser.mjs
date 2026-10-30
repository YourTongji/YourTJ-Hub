import assert from 'node:assert/strict'
import { before, after, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdtemp, readFile, mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { execFileSync } from 'node:child_process'
import { createServer } from 'node:http'
import { chromium } from 'playwright'
let browser, server, origin, fixtures
before(async () => {
 fixtures = await mkdtemp(join(tmpdir(), 'yourtj-activation-continuation-'))
 execFileSync('go', ['test', './app/http/controllers', '-run', 'TestActivationContinuationBrowserFixtures', '-count=1'], { cwd:fileURLToPath(new URL('../../', import.meta.url)), env:{...process.env,YOURTJ_TEST_HTML_DIR:fixtures}, stdio:'pipe' })
 server = createServer(async (req,res) => {
  const name = new URL(req.url, 'http://local').pathname.slice(1)
  if (!/^(zh|en|ja|de)\.html$/.test(name)) {res.writeHead(404);res.end();return}
  res.writeHead(200, {'Content-Type':'text/html; charset=utf-8','Cache-Control':'private, no-store','Referrer-Policy':'no-referrer'})
  res.end(await readFile(join(fixtures,name)))
 })
 await new Promise(resolve => server.listen(0,'127.0.0.1',resolve)); origin=`http://127.0.0.1:${server.address().port}`
 browser=await chromium.launch()
})
after(async () => {await browser?.close();if(server)await new Promise(resolve=>server.close(resolve));if(fixtures)await rm(fixtures,{recursive:true,force:true})})
for (const lang of ['zh','en','ja','de']) for (const width of [320,1280]) test(`activation success continuation ${lang} ${width}`,async () => {
 const page=await browser.newPage({viewport:{width,height:950}})
 try {
  await page.goto(`${origin}/${lang}.html`)
  assert.equal(await page.locator('html').getAttribute('lang'),lang)
  assert.equal(await page.locator('strong').textContent(),'verified_student')
  assert.equal(await page.locator('a.primary').getAttribute('href'),'/publish?id=42')
  assert.equal(await page.locator('a[href^="/login?"]').getAttribute('href'),'/login?redirect=%2Fpublish%3Fid%3D42')
  assert.equal(await page.locator('a[href^="yourtj:"]').getAttribute('href'),'yourtj://account/email-verified')
  for (const action of await page.locator('a.button').all()) {
   await action.focus(); assert.equal(await action.evaluate(e=>document.activeElement===e),true)
   const box=await action.boundingBox(); assert.ok(box&&box.height>=44&&box.x>=0&&box.x+box.width<=width+1)
  }
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
  assert.equal(await page.locator('form').count(),0)
  if(process.env.YOURTJ_TEST_ARTIFACTS){await mkdir(process.env.YOURTJ_TEST_ARTIFACTS,{recursive:true});await page.screenshot({path:`${process.env.YOURTJ_TEST_ARTIFACTS}/success-${lang}-${width}.png`,fullPage:true})}
 } finally {await page.close()}
})
