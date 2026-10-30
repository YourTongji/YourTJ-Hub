import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
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

for (const width of [320,1280]) for (const scenario of ['first-input','last-tail','ime-commit','clear']) {
 test(`local draft captures committed editor content before unload: ${scenario} ${width}`,async()=>{
  const page=await browser.newPage({viewport:{width,height:950}})
  try {
   await page.route('**/api/forum/anonymous/state',r=>r.fulfill({status:503,json:{code:1,messageCode:'anonymous.unavailable'}}))
   await page.goto(`${origin}/assets/test/fixtures/browser/quick-publish.html`)
   const editor=page.locator('.vditor [contenteditable="true"]:visible').first();await editor.waitFor();await editor.focus()
   if(scenario!=='first-input'){
    await editor.fill('之前已保存的正文')
    await page.waitForFunction(()=>JSON.parse(localStorage.getItem('gf:quick-publish-draft:v2:1:2')||'null')?.content?.includes('之前已保存的正文'))
   }
   const result=await editor.evaluate((element,scenario)=>{
    element.focus()
    const selection=getSelection();const range=document.createRange();range.selectNodeContents(element)
    if(scenario!=='clear')range.collapse(false)
    selection.removeAllRanges();selection.addRange(range)
    if(scenario==='ime-commit')element.dispatchEvent(new CompositionEvent('compositionstart',{bubbles:true}))
    document.execCommand(scenario==='clear'?'delete':'insertText',false,scenario==='clear'?null:'最后确认的几个字')
    if(scenario==='ime-commit')element.dispatchEvent(new CompositionEvent('compositionend',{bubbles:true,data:'最后确认的几个字'}))
    // The editor's real delayed input callback cannot run between this committed
    // input and the synchronous page-exit handler.
    const exit=new Event('beforeunload',{cancelable:true})
    window.dispatchEvent(exit)
    return {draft:JSON.parse(localStorage.getItem('gf:quick-publish-draft:v2:1:2')||'null'),prevented:exit.defaultPrevented}
   },scenario)
   assert.equal(result.prevented,scenario!=='clear')
   if(scenario==='clear')assert.equal(result.draft,null,'clearing the body must not resurrect the saved draft')
   else assert.ok(result.draft?.content?.includes('最后确认的几个字'),JSON.stringify(result))
   page.on('dialog',dialog=>dialog.accept())
   await page.reload()
   await page.locator('.vditor [contenteditable="true"]:visible').first().waitFor()
   if(scenario==='clear')assert.ok(!(await page.locator('.vditor [contenteditable="true"]:visible').first().textContent()).includes('之前已保存'))
   else await page.waitForFunction(()=>document.querySelector('.vditor [contenteditable="true"]')?.textContent?.includes('最后确认的几个字'))
  }finally{await page.close()}
 })
}
