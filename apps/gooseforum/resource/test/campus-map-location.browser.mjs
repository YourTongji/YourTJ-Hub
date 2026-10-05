import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

const resource = fileURLToPath(new URL('../', import.meta.url))
const evidence = resolve(process.env.CAMPUS_MAP_BROWSER_EVIDENCE_DIR ?? `${tmpdir()}/yourtj-campus-map-location`)
const fixturePath = '/assets/test/campus-map-location.html'
const virtualId = 'virtual:campus-map-location-browser'
const courses = [
  { code: 'SYN_DATE', name: '日期条件示例', campus: '四平路校区', room: '北楼115室（9月24日）' },
  { code: 'SYN_ODD', name: '单周条件示例', campus: '四平路校区', room: '北楼115室（单周）' },
  { code: 'SYN_EVEN', name: '双周条件示例', campus: '四平路校区', room: '北楼115室（双周）' },
  { code: 'SYN_MULTI', name: '独立楼名示例', campus: '嘉定校区', room: 'A101、B201' },
]
const targets = {
  north: { id: 'way/183383474', name: '教学北楼' },
  an: { id: 'way/266167562', name: '安楼（A楼）' },
  bo: { id: 'way/263922904', name: '博楼（B楼）' },
}
const details = Object.fromEntries(courses.map((course, index) => [course.code, [{
  campus: course.campus, teachingClassId: index + 1,
  arrangementInfo: [{ occupyDay: 2, occupyTime: [1, 2], occupyWeek: [1, 2],
    occupyRoom: course.room, arrangementText: '周二第1–2节（合成示例）', teacherAndCode: '' }],
}]]))
const receipt = { source: 'Production Vue page, CSS and GeoJSON; anonymous viewer; synthetic public PK responses',
  fixtures: courses, tests: [], screenshots: [] }
let server, browser, origin

before(async () => {
  await mkdir(evidence, { recursive: true })
  const hashFiles = ['src/site/campus-map/official-location.ts', 'src/site/campus-map/CampusCanvas.vue',
    'src/site/pages/CampusMapPage.vue', 'src/site/components/CampusMapSchedulePanel.vue',
    'src/site/components/CampusMapLocationChoices.vue', 'src/site/campus-map/data/siping.geojson',
    'src/site/campus-map/data/jiading.geojson', 'src/styles/resource.css']
  const hash = createHash('sha256')
  for (const path of hashFiles) hash.update(path).update(await readFile(resolve(resource, path)))
  receipt.sourceIdentity = process.env.GITHUB_SHA
    ? { kind: 'github-workflow-sha', value: process.env.GITHUB_SHA }
    : { kind: 'source-sha256', value: hash.digest('hex'), files: hashFiles }
  // A virtual entry keeps the fixture in this test while Vite handles real imports and styles.
  server = await createServer({ root: resource,
    build: { rollupOptions: { input: { site: resolve(resource, 'src/site/main.ts'), admin: resolve(resource, 'src/admin/main.ts') } } },
    optimizeDeps: { include: ['vue', 'vue-i18n', '@lucide/vue', 'maplibre-gl'], noDiscovery: true },
    server: { host: '127.0.0.1', port: 0, open: false, hmr: false },
    plugins: [{ name: 'campus-map-location-browser-fixture',
      resolveId(id) { if (id === virtualId || id.endsWith(`/@id/${virtualId}`)) return `\0${virtualId}` },
      load(id) {
        if (id !== `\0${virtualId}`) return
        return `import { createApp, h } from 'vue';
          import CampusMapPage from '@/site/pages/CampusMapPage.vue';
          import { i18n, setLocale } from '@/runtime/i18n';
          import '@/styles/resource.css';
          await setLocale('zh');
          createApp({ render: () => h(CampusMapPage, { layout: { viewer: { isAuthenticated: false } } }) }).use(i18n).mount('#app');
          document.documentElement.dataset.ready = 'true';`
      },
      configureServer(vite) {
        vite.middlewares.use((request, response, next) => {
          if (!request.url?.split('?')[0].endsWith('/test/campus-map-location.html')) return next()
          const html = `<!doctype html><html lang="zh" data-theme="gf-light"><head><meta name="viewport" content="width=device-width,initial-scale=1"></head><body><div id="app"></div><script type="module" src="/assets/@id/${virtualId}"></script></body></html>`
          vite.transformIndexHtml(request.url, html).then(body => {
            response.setHeader('Content-Type', 'text/html'); response.end(body)
          }).catch(next)
        })
      },
    }],
  })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch({ ...(process.platform === 'win32' ? { channel: 'msedge' } : {}),
    args: ['--use-angle=swiftshader', '--enable-unsafe-swiftshader'] })
})
after(async () => {
  await browser?.close()
  await server?.close()
  await writeFile(resolve(evidence, 'receipt.json'), JSON.stringify(receipt, null, 2))
})

async function open(width, query) {
  const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: 'reduce' })
  page.setDefaultTimeout(15_000)
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  await page.route('**/api/**', async route => {
    const path = new URL(route.request().url()).pathname
    let data
    if (path === '/api/pk/calendars') data = [{ calendarId: 122, calendarName: '合成验收学期（无起止日期）' }]
    else if (path === '/api/pk/latest-update') data = '2026-10-05'
    else if (path === '/api/pk/courses-by-time') data = { courses: courses.map(course => ({ courseCode: course.code, courseName: course.name, campus: [course.campus] })) }
    else if (path === '/api/pk/course-details') data = details
    else { errors.push(`Unexpected API request: ${path}`); return route.fulfill({ status: 500, json: { code: 1 } }) }
    await route.fulfill({ json: { code: 0, data } })
  })
  await page.goto(`${origin}${fixturePath}${query}`)
  await page.locator('html[data-ready="true"]').waitFor()
  await page.locator('.maplibregl-canvas').waitFor()
  return { page, errors }
}
async function query(panel, week) {
  await panel.locator('.atlas-schedule__filters select').first().selectOption('2')
  await panel.locator('input[type="number"]').fill(String(week))
  await panel.locator('.atlas-schedule__submit:not(:disabled)').click()
  await panel.locator('.atlas-schedule__list button').first().waitFor()
  assert.equal(await panel.locator('input[type="number"]').inputValue(), String(week))
  assert.equal(await panel.locator('.atlas-schedule__filters select').first().inputValue(), '2')
  assert.match(await panel.locator('.atlas-schedule__provenance').innerText(), new RegExp(`第 ${week} 周.*周二`))
}
async function selectedTarget(page, target) {
  await page.waitForFunction(id => new URLSearchParams(location.hash.slice(1)).get('place') === id, target.id)
  const pin = page.locator('.campus-label--selected')
  await pin.waitFor()
  assert.equal(await pin.getAttribute('aria-label'), target.name)
}
async function capture(page, filename, state) {
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, 'no horizontal overflow')
  await page.screenshot({ path: resolve(evidence, filename), fullPage: true })
  receipt.screenshots.push({ filename, width: page.viewportSize().width, ...state })
}
async function mapCapture(page, filename, target) {
  await page.locator('.atlas-building-schedule__header button').click()
  await page.locator('dialog[open]').waitFor({ state: 'hidden' })
  await selectedTarget(page, target)
  await capture(page, filename, { view: 'map', selectedTarget: target })
}

for (const width of [1280, 375]) {
  test(`trailing date and parity conditions filter the real north building at ${width}px`, async () => {
    const { page, errors } = await open(width, '?mine=1')
    const result = { scenario: 'trailing conditions', width, status: 'failed' }
    receipt.tests.push(result)
    try {
      await page.locator('.atlas-mine__views button').nth(1).click()
      let panel = page.locator('.atlas-mine .atlas-schedule')
      await query(panel, 1)
      for (const course of courses.slice(0, 3)) assert.ok((await panel.innerText()).includes(course.room))
      await capture(page, `conditions-originals-${width}.png`, { view: 'unscoped query', week: 1, day: 2, raw: courses.slice(0, 3).map(course => course.room) })
      await page.goto(`${origin}${fixturePath}#place=${encodeURIComponent(targets.north.id)}`)
      await page.locator('.atlas-schedule-open').click()
      panel = page.locator('dialog[open]')
      const scope = await panel.locator('.atlas-schedule__scope').innerText()
      assert.ok(scope.includes(targets.north.name))
      for (const [week, course] of [[1, courses[1]], [2, courses[2]]]) {
        await query(panel, week)
        const entry = panel.locator('.atlas-schedule__list button')
        assert.equal(await entry.count(), 1, 'date and opposite parity must be excluded')
        assert.ok((await entry.innerText()).includes(course.room))
        await entry.click()
        await selectedTarget(page, targets.north)
        assert.equal(await entry.getAttribute('aria-pressed'), 'true')
        assert.equal(await panel.locator('.atlas-location-choices button').getAttribute('aria-pressed'), 'true')
        assert.equal(await panel.locator('.atlas-schedule__scope').innerText(), scope)
        await capture(page, `conditions-week-${week}-${width}.png`, { view: 'scoped query', week, day: 2, raw: course.room, selectedTarget: targets.north })
      }
      await mapCapture(page, `conditions-map-${width}.png`, targets.north)
      assert.deepEqual(errors, [])
      result.status = 'passed'
    } finally { await page.close() }
  })
  test(`A101 and B201 select distinct real Jiading buildings at ${width}px`, async () => {
    const { page, errors } = await open(width, `?campus=jiading#place=${encodeURIComponent(targets.an.id)}`)
    const result = { scenario: 'independent An and Bo', width, status: 'failed' }
    receipt.tests.push(result)
    try {
      await page.locator('.atlas-schedule-open').click()
      const panel = page.locator('dialog[open]')
      const scope = await panel.locator('.atlas-schedule__scope').innerText()
      assert.ok(scope.includes(targets.an.name))
      await query(panel, 1)
      const entry = panel.locator('.atlas-schedule__list button')
      assert.equal(await entry.count(), 1)
      assert.ok((await entry.innerText()).includes(courses[3].room))
      await entry.click()
      const choices = panel.locator('.atlas-location-choices button')
      await choices.nth(1).waitFor()
      assert.equal(await choices.count(), 2)
      assert.equal(new URL(page.url()).hash, '', 'multiple locations must not choose an initial pin')
      assert.deepEqual(await choices.evaluateAll(items => items.map(item => item.getAttribute('aria-pressed'))), ['false', 'false'])
      await capture(page, `independent-unselected-${width}.png`, { view: 'scoped query', week: 1, day: 2, raw: courses[3].room, selectedTarget: null })
      for (const [index, target] of [targets.an, targets.bo].entries()) {
        await choices.nth(index).click()
        await selectedTarget(page, target)
        assert.equal(await choices.nth(index).getAttribute('aria-pressed'), 'true')
        assert.equal(await choices.nth(1 - index).getAttribute('aria-pressed'), 'false')
        assert.equal(await panel.locator('.atlas-schedule__scope').innerText(), scope)
        assert.equal(await entry.getAttribute('aria-pressed'), 'true')
        await capture(page, `independent-${index === 0 ? 'an' : 'bo'}-${width}.png`, { view: 'scoped query', week: 1, day: 2, raw: courses[3].room, scope, selectedTarget: target })
      }
      const boxes = await choices.evaluateAll(items => items.map(item => item.getBoundingClientRect().height))
      assert.ok(boxes.every(height => height >= 44), 'location controls retain touch target height')
      await mapCapture(page, `independent-map-${width}.png`, targets.bo)
      assert.deepEqual(errors, [])
      result.status = 'passed'
    } finally { await page.close() }
  })
}
