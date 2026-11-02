import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { readFile, mkdtemp, rm, writeFile, mkdir } from 'node:fs/promises'
import { createServer } from 'node:http'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { setTimeout as delay } from 'node:timers/promises'
import { test } from 'node:test'
import { chromium } from 'playwright'

const resource = fileURLToPath(new URL('../', import.meta.url))
const fixture = {
  component:'home.index',url:'/',version:'1',meta:{title:'Asset preload acceptance'},
  layout:{site:{name:'yourtj',description:'Fixture',brandType:'text',brandText:'yourtj'},
    viewer:{id:0,isAuthenticated:false,requiresEmailVerification:false,adminPermissions:[]},
    sidebar:{main:[],resources:[],groups:[],categories:[],activeKey:'topics'},footer:{links:[],primary:[]},
    unread:{notifications:false,messages:false,moderationReports:false},posting:{maxTitleLength:80},
    theme:{enabled:false,current:'gf-light',themeColor:'#fff'},umamiEnabled:false},
  props:{sort:'latest',actualSort:'latest',tabs:[{key:'latest',active:true,url:'/?sort=latest'}],
    topics:Array.from({length:17},(_,i)=>({id:i+1,title:`Synthetic campus topic ${i+1}`,description:'Readable campus topic description. '.repeat(4),
      url:`/p/post/${i+1}`,author:{id:100+i,username:'fixture',avatarUrl:''},participants:[],categories:[],replyCount:2,likeCount:0,viewCount:1,
      activityText:'now',lastUpdateTime:'2026-10-30T00:00:00Z',pinWeight:0,processStatus:0,contentType:0,liked:false,bookmarked:false})),
    pagination:{page:1,hasNext:false,nextUrl:''},announcement:{enabled:false,html:'',items:[]}},
}

test('initial document discovers the real Vite graph without refetching topics', async () => {
  const directory=await mkdtemp(resolve(tmpdir(),'yourtj-initial-assets-'))
  let server,browser
  try {
    try { execFileSync('go',['test','./app/http/controllers/forum/','-run','^TestResourceEntryBrowserFixture$','-count=1'],
      {cwd:resolve(resource,'..'),env:{...process.env,YOURTJ_ASSET_FIXTURE_OUTPUT:directory},stdio:'pipe'})
    } catch(error) { throw new Error(`Go asset fixture failed:\n${error.stdout?.toString().slice(-6000)||''}\n${error.stderr?.toString().slice(-6000)||''}`) }
    const head=await readFile(resolve(directory,'head.html'),'utf8')
    const preloadPaths=[...head.matchAll(/rel="modulepreload" href="([^"]+)"/g)].map(m=>m[1])
    assert.ok(preloadPaths.some(path=>/HomePage/.test(path)))
    const script=head.match(/type="module" src="([^"]+)"/)[1]
    server=createServer(async (request,response)=>{
      const path=new URL(request.url,'http://fixture').pathname
      if(path==='/') {
        const hints=request.url.includes('hints=1')
        response.setHeader('Content-Type','text/html')
        response.end(`<!doctype html><html lang="en" data-theme="gf-light"><head><meta name="viewport" content="width=device-width,initial-scale=1">${hints?head:head.replace(/<link rel="modulepreload"[^>]*>/g,'')}</head><body><script id="goose-payload" type="application/json">${JSON.stringify(fixture)}</script><div id="goose-app"></div></body></html>`)
      } else if(path.startsWith('/assets/')) {
        await delay(80) // fixed loopback latency exposes discovery order, not a user p75 estimate
        try {
          const file=resolve(resource,'static/dist',path.slice('/assets/'.length))
          if(!file.startsWith(resolve(resource,'static/dist')+'/')) throw Error('outside fixture')
          const data=await readFile(file)
          response.setHeader('Content-Type',path.endsWith('.css')?'text/css':path.endsWith('.js')?'text/javascript':'application/octet-stream')
          response.end(data)
        }catch { response.statusCode=404;response.end() }
      }else if(path==='/api/forum/stickers') {
        response.setHeader('Content-Type','application/json');response.end(JSON.stringify({code:0,data:[]}))
      }else {response.statusCode=404;response.end()}
    })
    await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve))
    const origin=`http://127.0.0.1:${server.address().port}`
    browser=await chromium.launch()
    const results=[]
    for(const [run,hints] of Array.from({length:3},(_,run)=>[[run,false],[run,true]]).flat()) {
      const context=await browser.newContext({viewport:{width:1280,height:900},serviceWorkers:'block'})
      const page=await context.newPage();const errors=[],api=[]
      await page.addInitScript(()=>{window.__lcp=0;new PerformanceObserver(list=>{window.__lcp=list.getEntries().at(-1)?.startTime??0}).observe({type:'largest-contentful-paint',buffered:true})});
      page.on('pageerror',error=>errors.push(error.message))
      page.on('request',request=>{if(request.url().startsWith(origin+'/api/'))api.push(new URL(request.url()).pathname)})
      await page.route('**/*',route=>route.request().url().startsWith(origin)?route.continue():route.abort())
      await page.goto(origin+`/?hints=${hints?1:0}`,{waitUntil:'domcontentloaded'})
      await page.locator('a[href="/p/post/1"]').first().waitFor({state:'visible',timeout:15000})
      await page.waitForTimeout(150)
      const metrics=await page.evaluate(()=>({lcp:window.__lcp,dcl:performance.getEntriesByType('navigation')[0].domContentLoadedEventEnd,resources:performance.getEntriesByType('resource').map(r=>({path:new URL(r.name).pathname,start:r.startTime,end:r.responseEnd})),rendered:document.querySelectorAll('.gf-topic-row').length}))
      assert.deepEqual(errors,[])
      assert.ok(metrics.rendered>=17)
      assert.ok(!api.some(path=>/topic|feed|home/.test(path)),`initial feed refetched: ${api}`)
      const entry=metrics.resources.find(r=>r.path===script)
      const home=metrics.resources.find(r=>/HomePage.*\.js$/.test(r.path))
      assert.ok(entry&&home)
      if(hints) {
        assert.ok(home.start<entry.end,'home requested while entry still downloads')
        for(const path of preloadPaths) assert.equal(metrics.resources.filter(r=>r.path===path).length,1,`duplicate/missing preload ${path}`)
      } else assert.ok(home.start>=entry.end,'control must discover home after entry')
      results.push({run,hints,entry,home,lcp:metrics.lcp,dcl:metrics.dcl,rendered:metrics.rendered,api})
      await context.close()
    }
    if(process.env.YOURTJ_TEST_ARTIFACTS) {
      await mkdir(process.env.YOURTJ_TEST_ARTIFACTS,{recursive:true})
      await writeFile(resolve(process.env.YOURTJ_TEST_ARTIFACTS,'built-asset-discovery.json'),JSON.stringify(results,null,2))
    }
    console.log(JSON.stringify(results))
  } finally {
    await browser?.close()
    if(server) await new Promise(resolve=>server.close(resolve))
    await rm(directory,{recursive:true,force:true})
  }
})
