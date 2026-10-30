import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium, webkit } from 'playwright'
import { createServer } from 'vite'
import { readFile } from 'node:fs/promises'
let server, browser, safari, origin
before(async () => {
 server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false }, logLevel: 'error' })
 await server.listen(); origin = `http://127.0.0.1:${server.httpServer.address().port}`
 safari = await webkit.launch()
 browser = await chromium.launch({ executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH })
})
after(async () => { await browser?.close(); await safari?.close(); await server?.close() })
for (const width of [360, 1280]) for (const theme of ['light', 'dark']) {
 test(`badge artwork editor stays usable at ${width}px in ${theme}`, async () => {
  const page = await browser.newPage({ viewport: { width, height: 900 } })
  page.setDefaultTimeout(10000)
  const errors=[];page.on('pageerror',error=>errors.push(error.message))
  const badge={ code:'test-badge', name:'校园贡献者', description:'Test badge', type:'custom', grantMode:'manual', iconType:'asset', iconKey:'', iconUrl:'/static/badges/contributor.svg', color:'blue', level:'bronze', isEnabled:true, isWearable:true, sortOrder:1 }
  await page.route('**/api/admin/badges',route=>route.fulfill({json:{code:0,result:[badge]}}))
  await page.route('**/static/**', async route=> {
   const name=new URL(route.request().url()).pathname.slice('/static/'.length)
   try {await route.fulfill({body:await readFile(new URL('../static/'+name,import.meta.url)),contentType:name.endsWith('.svg')?'image/svg+xml':'image/webp'})}catch{await route.fulfill({status:404})}
  })
  const fittedImage = await page.evaluate(() => {
    const canvas=document.createElement('canvas');canvas.width=512;canvas.height=512
    const ctx=canvas.getContext('2d');ctx.fillStyle='#2563eb';ctx.beginPath();ctx.arc(256,256,512*10/24,0,Math.PI*2);ctx.fill()
    return canvas.toDataURL('image/png').split(',')[1]
  })
  await page.route('**/static/badges/normalized/*',route=>route.fulfill({body:Buffer.from(fittedImage,'base64'),contentType:'image/png'}))
  await page.goto(`${origin}/assets/test/fixtures/browser/badge-assets-admin.html?theme=${theme}`)
  await page.locator('button[title="校园贡献者 · test-badge"]').click()
  const dialog=page.getByRole('dialog')
  await dialog.waitFor()
  await page.locator('#badge-artwork').fill('/static/badges/robot.svg')
  assert.equal(await dialog.locator('aside section').count(),2)
  assert.equal(await dialog.locator('img[src="/static/badges/robot.svg"]').count(),4)
  const normalized='/static/badges/normalized/12345678-1234-1234-1234-123456789abc.png'
  await page.locator('#badge-artwork').fill(normalized)
  assert.equal(await dialog.locator(`img[src="${normalized}"]`).count(),4)
  assert.equal(await dialog.locator('aside .group\\/avatar span img').count(),2)
  for(const image of await dialog.locator('aside .group\\/avatar span img').all()) {
    assert.match(await image.getAttribute('class'),/h-\[88%\]/)
  }

  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
  const save=dialog.getByRole('button',{name:'保存',exact:true})
  await save.scrollIntoViewIfNeeded(); assert.equal(await save.isVisible(),true)
  // Real keyboard entry can reach the visible upload action and stable footer.
  await page.locator('#badge-artwork').focus();await page.keyboard.press('Tab')
  assert.match(await page.evaluate(()=>document.activeElement.textContent),/上传徽章图片/)
  assert.deepEqual(errors,[])
  if(process.env.YOURTJ_SCREENSHOT_DIR) await page.screenshot({path:`${process.env.YOURTJ_SCREENSHOT_DIR}/badge-assets-${width}-${theme}.png`})
  await page.close()
 })
}

for (const theme of ['light', 'dark']) {
 test(`WebKit badge cards contain long names (${theme})`, async () => {
  const page=await safari.newPage({viewport:{width:402,height:722}})
  try {
   const badge={code:'long-name',name:'Long unbroken badge label '+ 'abcdefghijk'.repeat(6),description:'Synthetic badge',type:'system',grantMode:'auto',iconUrl:'/static/badges/contributor.svg',color:'blue',level:'bronze',isEnabled:true,isWearable:true,sortOrder:1}
   await page.route('**/api/admin/badges',route=>route.fulfill({json:{code:0,result:Array.from({length:9},(_,i)=>({...badge,code:badge.code+i}))}}))
   await page.goto(`${origin}/assets/test/fixtures/browser/badge-assets-admin.html?lang=en&theme=${theme}`)
   await page.locator('button[title]').filter({hasText:badge.name}).first().waitFor();await page.evaluate(()=>document.fonts.ready)
   assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1))
   const buttons=await page.locator('button[title]').filter({hasText:badge.name}).evaluateAll(nodes=>nodes.map(n=>({left:n.getBoundingClientRect().left,right:n.getBoundingClientRect().right,parent:n.parentElement.getBoundingClientRect().toJSON()})))
   assert.ok(buttons.every(b=>b.left>=b.parent.left-1&&b.right<=b.parent.right+1),JSON.stringify(buttons))
   if(process.env.YOURTJ_SCREENSHOT_DIR)await page.screenshot({path:`${process.env.YOURTJ_SCREENSHOT_DIR}/badge-list-webkit-${theme}.png`})
  }finally{await page.close()}
 })
}
