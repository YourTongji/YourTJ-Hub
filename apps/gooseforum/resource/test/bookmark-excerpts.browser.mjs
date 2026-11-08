import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { execFileSync } from 'node:child_process'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

// #1639: assert painted line height with the real Tailwind cascade. DOM-only
// tests cannot catch display:block overriding line-clamp's -webkit-box.
const evidence = process.env.YOURTJ_BOOKMARK_EXCERPT_EVIDENCE_DIR
const captures = []
let server, browser, origin, cacheDir, browserVersion
before(async () => {
  if (evidence) await mkdir(evidence, { recursive: true })
  cacheDir = await mkdtemp(resolve(tmpdir(), 'yourtj-bookmark-excerpts-'))
  server = await createServer({ cacheDir, root: fileURLToPath(new URL('../', import.meta.url)),
    optimizeDeps: { entries: ['test/fixtures/browser/bookmark-excerpts.html'] },
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
  if (evidence) await writeFile(resolve(evidence, 'receipt.json'), JSON.stringify({
    checkoutSha: execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim(),
    prHeadSha: process.env.BOOKMARK_EXCERPT_PR_HEAD_SHA || null,
    browser: 'Chromium', browserVersion, captures,
    scope: 'Real UserPage/AppShell/CSS with synthetic bookmarks and initialized memory routing. No live account/backend; activity is a retained render branch, not current server navigation. Screenshots require visual review.',
  }, null, 2))
})

for (const section of ['bookmarks', 'activity']) for (const width of [320, 1280]) {
  for (const theme of ['light', 'dark']) test(`${section}: reply excerpts clamp to two lines at ${width}px in ${theme}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 }, colorScheme: theme, reducedMotion: 'reduce', timezoneId: 'UTC' })
    const errors = []
    const capture = { section, width, theme, result: 'pending' }
    captures.push(capture)
    page.on('pageerror', error => errors.push(error.message))
    await page.route('**/*', route => new URL(route.request().url()).origin === origin ? route.continue() : route.abort())
    await page.route('**/api/**', route => route.fulfill({ status: 404, json: { code: 1, message: 'not in fixture' } }))
    await page.route('**/api/forum/unread-status', route => route.fulfill({ json: { code: 0, result: { notifications: false, messages: false } } }))
    try {
      await page.goto(`${origin}/assets/test/fixtures/browser/bookmark-excerpts.html?section=${section}&theme=${theme}`)
      const longRow = page.locator('a[href="/p/200?postNo=3"]')
      const shortRow = page.locator('a[href="/p/200?postNo=4"]')
      const emptyRow = page.locator('a[href="/p/200?postNo=5"]')
      await longRow.waitFor({ state: 'visible' })
      await page.evaluate(() => document.fonts.ready)
      const measure = locator => locator.evaluate(element => {
        const style = getComputedStyle(element)
        return { height: element.getBoundingClientRect().height, lineHeight: Number.parseFloat(style.lineHeight), scrollHeight: element.scrollHeight, display: style.display }
      })
      capture.long = await measure(longRow.locator('.line-clamp-2'))
      capture.short = await measure(shortRow.locator('.line-clamp-2'))
      if (evidence) {
        capture.file = `${section}-${theme}-${width}.png`
        await page.screenshot({ path: resolve(evidence, capture.file), fullPage: true })
      }

      assert.equal(await page.locator('html').getAttribute('data-theme'), `gf-${theme}`)
      assert.equal(await page.locator('button.fixed[aria-haspopup="true"]').count(), 0, 'profile fixture must not render the home publish FAB')
      if (width === 320) {
        assert.ok(capture.long.scrollHeight > capture.long.lineHeight * 2 + 1, 'narrow fixture must contain more than two natural lines')
        assert.ok(Math.abs(capture.long.height - capture.long.lineHeight * 2) <= 1,
          `narrow long reply excerpt must occupy two lines: ${JSON.stringify(capture.long)}`)
      } else {
        assert.ok(capture.long.height <= capture.long.lineHeight * 2 + 1,
          `wide long reply excerpt must not exceed two lines: ${JSON.stringify(capture.long)}`)
      }
      assert.ok(Math.abs(capture.short.height - capture.short.lineHeight) <= 1, 'short excerpt stays one line without reserved blank space')
      assert.equal(await shortRow.locator('.line-clamp-2').textContent().then(s => s.trim()), '简短摘要。')
      assert.equal(await emptyRow.locator('.line-clamp-2').count(), 0, 'empty excerpts do not render an empty line')
      assert.equal(await page.locator('a[href="/p/100"] .line-clamp-2').count(), 0, 'topic bookmarks gain no reply excerpt')
      for (const [row, floor] of [[longRow, '#3'], [shortRow, '#4'], [emptyRow, '#5']]) {
        assert.equal(await row.locator('.gf-badge').textContent(), floor)
        assert.equal(await row.locator('time').count(), 1)
        assert.match(await row.locator('time').textContent(), /2026-11-01/)
      }
      assert.equal(await longRow.locator('.truncate').textContent().then(s => s.trim()), '长摘要回复收藏：图书馆窗边的银杏与期末复习计划，窄屏长标题仍应省略')
      const bounds = await page.locator('a[href^="/p/"]').evaluateAll(elements => elements.map(element => {
        const { x, y, width, height } = element.getBoundingClientRect()
        return { x, y, width, height }
      }))
      assert.equal(bounds.length, 4)
      assert.ok(bounds.every(b => b.x >= 0 && b.x + b.width <= width + 1))
      assert.ok(bounds.every((b, i) => i === 0 || bounds[i - 1].y + bounds[i - 1].height <= b.y), 'bookmark rows stay ordered without overlap')
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'no horizontal overflow')
      assert.deepEqual(errors, [])
      capture.result = 'passed'
    } catch (error) {
      capture.result = 'failed'
      capture.error = error.message
      throw error
    } finally {
      await page.close()
    }
  })
}
