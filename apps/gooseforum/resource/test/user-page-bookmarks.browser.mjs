import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { execFileSync } from 'node:child_process'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

// #1624: synthetic page payloads, real UserPage/AppShell/CSS, browser fetch and
// IntersectionObserver pagination. Captures are review evidence, not goldens.
const evidence = process.env.YOURTJ_BOOKMARK_SCREENSHOT_DIR
const captures = []
let server, browser, origin, cacheDir, browserVersion
before(async () => {
  if (evidence) await mkdir(evidence, { recursive: true })
  cacheDir = await mkdtemp(resolve(tmpdir(), 'yourtj-bookmark-browser-'))
  server = await createServer({ cacheDir, root: fileURLToPath(new URL('../', import.meta.url)),
    optimizeDeps: { entries: ['test/fixtures/browser/profile-privacy.html'] },
    server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
  browserVersion = browser.version()
})
after(async () => {
  await browser?.close()
  await server?.close()
  if (cacheDir) await rm(cacheDir, { recursive: true, force: true })
  if (!evidence) return
  const receipt = {
    checkoutSha: execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim(),
    prHeadSha: process.env.BOOKMARK_PR_HEAD_SHA || null,
    browser: 'Chromium', browserVersion, expectedCases: 16, expectedCaptures: 32, captures,
    visualReview: 'not_performed',
    scope: 'Real browser rendering with synthetic bookmarks and intercepted pagination. No production account or backend verification. Captures require human visual review.',
  }
  await writeFile(resolve(evidence, 'receipt.json'), JSON.stringify(receipt, null, 2))
  await writeFile(resolve(evidence, 'index.html'), `<!doctype html><html lang="zh"><meta charset="utf-8"><title>混合收藏浏览器验收</title><style>body{font-family:system-ui;margin:24px;background:#eceff2;color:#19212a}section{display:flex;flex-wrap:wrap;gap:20px}figure{margin:0;max-width:640px}img{width:100%;border:1px solid #657181}figcaption{margin:10px 0}</style><h1>混合收藏浏览器验收（合成数据）</h1><p>实际 Chromium / Vue 页面；不是线上账号、后端或物理设备验收。截图不等于人工视觉结论。</p><p>Checkout: ${receipt.checkoutSha}</p><section>${captures.map(c => `<figure><figcaption>${c.section} · ${c.first} first · ${c.theme} · ${c.width}px · ${c.phase}</figcaption><a href="${c.file}"><img src="${c.file}" alt="${c.file}" loading="lazy"></a></figure>`).join('')}</section></html>`)
})

const hrefs = { topic: '/p/100', post: '/p/200?postNo=3' }
const rows = page => page.locator('a[href="/p/100"], a[href="/p/200?postNo=3"]')

for (const width of [320, 1280]) for (const theme of ['light', 'dark']) {
  for (const section of ['bookmarks', 'activity']) for (const first of ['topic', 'post']) {
    test(`${section}: ${first} first retains both ID 7 rows at ${width}px in ${theme}`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 900 }, colorScheme: theme, reducedMotion: 'reduce' })
      const errors = []
      const warnings = []
      let requests = 0, nextPayload, release
      const gate = new Promise(resolve => { release = resolve })
      page.on('pageerror', error => errors.push(error.message))
      page.on('console', message => { if (/duplicate keys?/i.test(message.text())) warnings.push(message.text()) })
      await page.route('**/*', route => new URL(route.request().url()).origin === origin ? route.continue() : route.abort())
      await page.route('**/api/**', route => route.fulfill({ status: 404, json: { code: 1, message: 'not in fixture' } }))
      await page.route('**/api/forum/unread-status', route => route.fulfill({ json: { code: 0, result: { notifications: false, messages: false } } }))
      await page.route('**/u/1024/bookmarks?cursor=identity', async route => {
        requests++
        await gate
        if (nextPayload) await route.fulfill({ json: nextPayload })
        else await route.abort()
      })
      const capture = async phase => {
        if (!evidence) return
        const file = `${section}-${first}-first-${theme}-${width}-${phase}.png`
        await page.screenshot({ path: resolve(evidence, file), fullPage: true })
        captures.push({ section, first, theme, width, phase, file })
      }
      try {
        const requested = page.waitForRequest(request => new URL(request.url()).pathname === '/u/1024/bookmarks')
        await page.goto(`${origin}/assets/test/fixtures/browser/profile-privacy.html?screen=bookmarks&section=${section}&first=${first}&theme=${theme}`)
        await page.locator(`a[href="${hrefs[first]}"]`).waitFor({ state: 'visible' })
        nextPayload = await page.evaluate(() => window.bookmarkAcceptancePage)
        assert.ok(nextPayload, 'fixture must provide a valid second page')
        assert.equal(await rows(page).count(), 1, 'first page has one bookmark while the next page is held')
        assert.equal(await page.locator('html').getAttribute('data-theme'), `gf-${theme}`)
        assert.equal(await page.locator('button.fixed[aria-haspopup="dialog"]').count(), 0,
          'profile routes must not render the home-only publish button')
        await capture('first-page')

        // Use the real scroll/observer path; short pages may already be loading.
        await page.keyboard.press('End')
        await requested
        release()
        const other = first === 'topic' ? 'post' : 'topic'
        await page.locator(`a[href="${hrefs[other]}"]`).waitFor({ state: 'visible' })
        assert.equal(requests, 1)
        assert.deepEqual(await rows(page).evaluateAll(elements => elements.map(el => el.getAttribute('href'))), [hrefs[first], hrefs[other]])
        assert.equal(await page.locator('a[href="/p/200?postNo=3"] .gf-badge').textContent(), '#3')
        assert.match(await page.locator('a[href="/p/200?postNo=3"]').textContent(), /这是一条合成回复摘要/)
        assert.equal(await rows(page).locator('time').count(), 2)
        const bounds = await rows(page).evaluateAll(elements => elements.map(el => {
          const { x, y, width, height } = el.getBoundingClientRect()
          return { x, y, width, height }
        }))
        assert.ok(bounds.every(b => b.width > 0 && b.height > 0 && b.x >= 0 && b.x + b.width <= width + 1), 'both bookmark rows fit the viewport')
        assert.ok(bounds[0].y + bounds[0].height <= bounds[1].y, 'rows preserve order without overlap')
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'page must not overflow horizontally')
        assert.deepEqual(warnings, [], 'composite identities must not produce duplicate Vue keys')
        assert.deepEqual(errors, [], 'the mounted page must not throw')
        await page.evaluate(() => scrollTo(0, 0))
        await capture('after-pagination')
      } finally {
        release()
        await page.close()
      }
    })
  }
}
