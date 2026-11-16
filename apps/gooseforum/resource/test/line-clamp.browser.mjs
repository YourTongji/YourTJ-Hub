import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { execFileSync } from 'node:child_process'
import { mkdir, mkdtemp, readFile, readdir, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

// The fixtures mount the production Vue components, while styles are loaded
// from the current commit's production build (including Tailwind's CSS order).
const evidence = process.env.YOURTJ_LINE_CLAMP_EVIDENCE_DIR
const resourceRoot = fileURLToPath(new URL('../', import.meta.url))
const distRoot = resolve(resourceRoot, 'static/dist')
const captures = []
let server, browser, origin, cacheDir, browserVersion, stylesheetFiles

before(async () => {
  if (evidence) await mkdir(evidence, { recursive: true })
  const manifest = JSON.parse(await readFile(resolve(distRoot, '.vite/manifest.json'), 'utf8'))
  const siteStyles = manifest['src/site/main.ts']?.css ?? []
  const adminStyles = manifest['src/admin/main.ts']?.css ?? []
  const scheduleStyles = Object.values(manifest).find(entry => entry.name === 'ScheduleTimeGrid')?.css ?? []
  assert.ok(siteStyles.length && adminStyles.length && scheduleStyles.length,
    'build the current site and admin entries plus ScheduleTimeGrid CSS before browser tests')
  stylesheetFiles = {
    schedule: [...new Set([...siteStyles, ...scheduleStyles])].map(file => resolve(distRoot, file)),
    select: adminStyles.map(file => resolve(distRoot, file)),
  }
  for (const path of Object.values(stylesheetFiles).flat()) {
    assert.ok((await readFile(path)).length > 0, `built stylesheet exists: ${path}`)
  }

  cacheDir = await mkdtemp(resolve(tmpdir(), 'yourtj-line-clamp-'))
  server = await createServer({
    cacheDir,
    root: resourceRoot,
    optimizeDeps: { entries: ['test/fixtures/browser/line-clamp.html'] },
    server: { host: '127.0.0.1', port: 0, open: false },
  })
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
    prHeadSha: process.env.LINE_CLAMP_PR_HEAD_SHA || null,
    browser: 'Chromium',
    browserVersion,
    stylesheetFiles: Object.fromEntries(Object.entries(stylesheetFiles ?? {}).map(([surface, files]) => [
      surface,
      files.map(path => path.replace(`${distRoot}/`, '')),
    ])),
    expectedCases: 8,
    captures,
    scope: 'Real ScheduleTimeGrid and AdminSelect Vue components with synthetic data, served by Vite and styled with this commit\'s production site/admin CSS. Not a live account or backend test. Review all screenshots for visual evidence.',
  }
  await writeFile(resolve(evidence, 'receipt.json'), JSON.stringify(receipt, null, 2))
  await writeFile(resolve(evidence, 'index.html'), `<!doctype html><html lang="zh"><meta charset="utf-8"><title>Line clamp browser evidence</title><style>body{font:14px system-ui;margin:24px;background:#eceff2;color:#19212a}section{display:flex;flex-wrap:wrap;gap:20px}figure{margin:0;max-width:620px}img{width:100%;border:1px solid #657181}figcaption{margin:8px 0}</style><h1>Line clamp browser evidence</h1><p>Chromium ${browserVersion}; checkout ${receipt.checkoutSha}; real components and built CSS with synthetic data.</p><section>${captures.map(c => `<figure><figcaption>${c.surface} · ${c.theme} · ${c.width}px · ${c.result}</figcaption><a href="${c.file}"><img src="${c.file}" alt="${c.file}" loading="lazy"></a></figure>`).join('')}</section></html>`)
})

for (const surface of ['schedule', 'select']) for (const width of [320, 1280]) for (const theme of ['light', 'dark']) {
  test(`${surface}: ${width}px ${theme}`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 }, colorScheme: theme, reducedMotion: 'reduce' })
    const errors = []
    const capture = { surface, width, theme, result: 'pending' }
    captures.push(capture)
    page.on('pageerror', error => errors.push(error.message))
    page.on('console', message => { if (message.type() === 'error') errors.push(message.text()) })
    await page.route('**/*', route => new URL(route.request().url()).origin === origin ? route.continue() : route.abort())
    try {
      await page.goto(`${origin}/assets/test/fixtures/browser/line-clamp.html?surface=${surface}&theme=${theme}`)
      await page.waitForFunction(() => document.documentElement.dataset.ready === 'true', null, { timeout: 10000 }).catch(error => {
        throw new Error(`${error.message}; browser errors: ${errors.join(' | ') || 'none'}`)
      })
      for (const path of stylesheetFiles[surface]) await page.addStyleTag({ path })
      await page.evaluate(() => document.fonts.ready)
      assert.equal(await page.locator('html').getAttribute('data-theme'), `gf-${theme}`)

      if (surface === 'schedule') {
        const titles = page.locator('.schedule-course-card h3')
        assert.equal(await titles.count(), 2, `real schedule component rendered: ${await page.locator('#app').innerHTML()} (${errors.join(' | ')})`)
        const readTitle = async locator => locator.evaluate(element => {
          const style = getComputedStyle(element)
          const bounds = element.getBoundingClientRect()
          return {
            text: element.textContent.trim(),
            display: style.display,
            lineClamp: style.webkitLineClamp,
            lineHeight: Number.parseFloat(style.lineHeight),
            height: bounds.height,
            scrollHeight: element.scrollHeight,
          }
        })
        capture.longTitle = await readTitle(titles.nth(0))
        capture.shortTitle = await readTitle(titles.nth(1))
        capture.cardHeight = await page.locator('.schedule-course-card').first().evaluate(card => card.getBoundingClientRect().height)
        if (evidence) {
          capture.file = `${surface}-${theme}-${width}.png`
          await page.screenshot({ path: resolve(evidence, capture.file), fullPage: true })
        }
        assert.notEqual(capture.longTitle.display, 'block', `long course title is not reset to block display: ${JSON.stringify(capture.longTitle)}`)
        assert.equal(capture.longTitle.lineClamp, '2')
        assert.ok(capture.longTitle.scrollHeight > capture.longTitle.lineHeight * 2, 'long course title contains more than two natural lines')
        assert.ok(capture.longTitle.height <= capture.longTitle.lineHeight * 2 + 7, 'long course title paints no more than two lines')
        assert.equal(capture.shortTitle.text, '概率论')
        assert.ok(capture.shortTitle.height <= capture.shortTitle.lineHeight + 7, 'short title remains one line')
        assert.equal(await page.getByText('#TJ-ALG-2030', { exact: true }).isVisible(), width >= 768, 'long course code keeps its existing desktop-only visibility')
        assert.equal(await page.getByText('#MATH-101', { exact: true }).isVisible(), width >= 768, 'short course code keeps its existing desktop-only visibility')
        assert.ok(capture.cardHeight <= 160, `long title does not expand its two-row schedule card: ${capture.cardHeight}px`)
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'schedule scroll container prevents page overflow')
      } else {
        const trigger = page.locator('[data-slot="select-trigger"]')
        const directTrigger = trigger.nth(1)
        assert.equal(await trigger.count(), 2, `AdminSelect and direct shared SelectTrigger rendered: ${await page.locator('#app').innerHTML()} (${errors.join(' | ')})`)
        assert.equal(await trigger.first().getAttribute('role'), 'combobox')
        assert.equal(await trigger.first().getAttribute('aria-label'), '管理端课程筛选')
        assert.equal(await directTrigger.getAttribute('aria-label'), '直接共享课程筛选')
        const value = trigger.first().locator('[data-slot="select-value"]')
        const valueText = value.locator('span')
        const directValue = directTrigger.locator('[data-slot="select-value"]')
        const directValueText = directValue.locator('span')
        const measure = locator => locator.evaluate(element => {
          const style = getComputedStyle(element)
          const bounds = element.getBoundingClientRect()
          return {
            text: element.textContent.trim(),
            display: style.display,
            whiteSpace: style.whiteSpace,
            textOverflow: style.textOverflow,
            overflow: style.overflow,
            width: bounds.width,
            height: bounds.height,
            clientHeight: element.clientHeight,
            scrollHeight: element.scrollHeight,
            clientWidth: element.clientWidth,
            scrollWidth: element.scrollWidth,
          }
        })
        await page.keyboard.press('Tab')
        assert.equal(await trigger.first().evaluate(element => element === document.activeElement), true, 'trigger is reachable by keyboard focus')
        capture.focusRing = await trigger.first().evaluate(element => {
          const style = getComputedStyle(element)
          return { focusVisible: element.matches(':focus-visible'), borderColor: style.borderColor, boxShadow: style.boxShadow }
        })
        assert.equal(capture.focusRing.focusVisible, true, 'keyboard focus uses the trigger focus-visible state')
        const triggerHeight = await trigger.first().evaluate(element => element.getBoundingClientRect().height)
        capture.valueRoot = await value.evaluate(element => ({ display: getComputedStyle(element).display, gap: getComputedStyle(element).columnGap }))
        capture.directValueRoot = await directValue.evaluate(element => ({ display: getComputedStyle(element).display, gap: getComputedStyle(element).columnGap }))
        capture.longValue = await measure(valueText)
        capture.directLongValue = await measure(directValueText)
        assert.match(capture.longValue.text, /^学院、专业与课程组合筛选/)
        assert.match(capture.directLongValue.text, /^学院、专业与课程组合筛选/)
        assert.equal(await trigger.first().locator('svg').count(), 1, 'dropdown icon remains present')
        assert.equal(await directTrigger.locator('svg').count(), 1, 'direct trigger keeps its dropdown icon')
        assert.equal(triggerHeight, 36, 'default trigger height remains 36px')
        assert.equal(await directTrigger.evaluate(element => element.getBoundingClientRect().height), 36)

        await page.keyboard.press('Enter')
        const items = page.locator('[data-slot="select-item"]')
        await items.first().waitFor({ state: 'visible' })
        capture.openContent = await items.count()
        assert.equal(capture.openContent, 2, 'keyboard Enter opens the real options list')
        await page.keyboard.press('Escape')
        await page.waitForFunction(() => document.querySelector('[data-slot="select-trigger"]')?.getAttribute('data-state') === 'closed')
        await page.locator('[data-slot="select-content"]').waitFor({ state: 'hidden' })
        assert.equal((await measure(valueText)).text, capture.longValue.text, 'keyboard Escape closes without changing the selected value')
        capture.triggerWidth = await trigger.first().evaluate(element => element.getBoundingClientRect().width)
        if (evidence) {
          capture.file = `${surface}-${theme}-${width}.png`
          await page.screenshot({ path: resolve(evidence, capture.file), fullPage: true })
        }
        assert.equal(capture.longValue.whiteSpace, 'nowrap', `long selected value stays on one line: ${JSON.stringify(capture.longValue)}`)
        assert.equal(capture.longValue.textOverflow, 'ellipsis', `long selected value uses an ellipsis: ${JSON.stringify(capture.longValue)}`)
        assert.equal(capture.longValue.display, 'block', 'ellipsis belongs to a block text wrapper inside the flex value')
        assert.equal(capture.valueRoot.display, 'flex', 'value root keeps flex alignment')
        assert.equal(capture.directValueRoot.display, 'flex', 'direct value root keeps flex alignment')
        assert.equal(capture.valueRoot.gap, '8px', 'value icon spacing remains 8px')
        assert.ok(capture.longValue.scrollWidth > capture.longValue.clientWidth, 'long selected value is clipped within the trigger')
        assert.equal(capture.directLongValue.textOverflow, 'ellipsis', 'a SelectValue without a custom slot also truncates')
        assert.ok(capture.directLongValue.scrollWidth > capture.directLongValue.clientWidth)
        assert.ok(capture.triggerWidth <= 260, 'long option does not expand the trigger beyond its layout width')
        assert.ok(await directTrigger.evaluate(element => element.getBoundingClientRect().width <= 260))
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'select does not overflow the page')
        await trigger.first().click()
        await page.locator('[data-slot="select-content"][data-state="open"] [data-slot="select-item"]').filter({ hasText: '短选项' }).click()
        capture.shortValue = await measure(valueText)
        assert.equal(capture.shortValue.text, '短选项')
        assert.ok(capture.shortValue.height <= capture.shortValue.clientHeight + 1, 'short value stays one line')
        await directTrigger.click()
        await page.locator('[data-slot="select-content"][data-state="open"] [data-slot="select-item"]').filter({ hasText: '短选项' }).click()
        capture.directShortValue = await measure(directValueText)
        assert.equal(capture.directShortValue.text, '短选项')
      }
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
