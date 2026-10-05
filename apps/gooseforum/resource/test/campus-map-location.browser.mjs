import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
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
  { code: 'SYN_MISSING', name: '未收录地点示例', campus: '四平路校区', room: '未知教学楼9999室（单周）' },
  { code: 'SYN_TIME', name: '周次条件示例', campus: '嘉定校区', room: '1-2周济事楼430;3-16周济事楼109' },
  { code: 'SYN_REVIEW', name: '待复核地点示例', campus: '嘉定校区', room: '学院教室' },
  { code: 'SYN_MULTI', name: '多个楼宇示例', campus: '四平路校区', room: '瑞安楼403、505、507，物理馆319、301、302' },
  { code: 'SYN_ONLINE', name: '线上课程示例', campus: '嘉定校区', room: '线上课堂' },
  { code: 'SYN_DATE', name: '日期尾注示例', campus: '四平路校区', room: '北楼115室（9月24日）' },
  { code: 'SYN_PARITY', name: '单双周尾注示例', campus: '四平路校区', room: '北楼115室（单周）' },
  { code: 'SYN_ABBR', name: '跨楼缩写示例', campus: '嘉定校区', room: 'A101、B201' },
]
const targets = {
  jishi: { id: 'way/135405205', name: '济事楼（软件学院）' },
  rui: { id: 'way/183383954', name: '瑞安楼' },
  physics: { id: 'way/183383958', name: '物理馆' },
  north: { id: 'way/183383474', name: '教学北楼' },
  an: { id: 'way/266167562', name: '安楼（A楼）' },
  bo: { id: 'way/263922904', name: '博楼（B楼）' },
}
const details = Object.fromEntries(courses.map((course, index) => [course.code, [{
  campus: course.campus, teachingClassId: index + 1,
  arrangementInfo: [{ occupyDay: 2, occupyTime: [1, 2], occupyWeek: Array.from({ length: 20 }, (_, index) => index + 1),
    occupyRoom: course.room, arrangementText: '周二第1–2节（合成示例）', teacherAndCode: '' }],
}]]))
const receipt = { source: 'Production Vue page, CSS and GeoJSON; anonymous viewer; synthetic public PK responses',
  fixtures: courses, tests: [], screenshots: [] }
let server, browser, origin, cacheDir

before(async () => {
  await mkdir(evidence, { recursive: true })
  const hashFiles = ['src/site/campus-map/official-location.ts', 'src/site/campus-map/location-types.ts',
    'src/site/campus-map/deterministic-location.ts', 'src/site/campus-map/data/locations/places.json',
    'src/site/campus-map/data/locations/overrides.json', 'src/site/campus-map/CampusCanvas.vue',
    'src/site/pages/CampusMapPage.vue', 'src/site/components/CampusMapSchedulePanel.vue',
    'src/site/components/CampusMapLocationChoices.vue', 'src/site/campus-map/data/siping.geojson',
    'src/site/campus-map/data/jiading.geojson', 'src/styles/resource.css']
  const hash = createHash('sha256')
  for (const path of hashFiles) hash.update(path).update(await readFile(resolve(resource, path)))
  receipt.sourceIdentity = process.env.GITHUB_SHA
    ? { kind: 'github-workflow-sha', value: process.env.GITHUB_SHA }
    : { kind: 'source-sha256', value: hash.digest('hex'), files: hashFiles }
  // A virtual entry keeps the fixture in this test while Vite handles real imports and styles.
  // This optimizer has a different dependency set. Sharing the default cache with concurrent
  // browser suites can replace their Vue chunks while pages are importing them.
  cacheDir = await mkdtemp(resolve(tmpdir(), 'yourtj-campus-map-vite-'))
  server = await createServer({ root: resource, cacheDir,
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
  if (cacheDir) await rm(cacheDir, { recursive: true, force: true })
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
    if (path === '/api/pk/calendars') data = [{ calendarId: 122, calendarName: '本学期合成课程' }, { calendarId: 121, calendarName: '未收录学期合成课程' }]
    else if (path === '/api/pk/latest-update') data = '2026-10-05'
    else if (path === '/api/pk/courses-by-time') data = { courses: courses.map(course => ({ courseCode: course.code, courseName: course.name, campus: [course.campus] })) }
    else if (path === '/api/pk/course-details') data = details
    else { errors.push(`Unexpected API request: ${path}`); return route.fulfill({ status: 500, json: { code: 1 } }) }
    await route.fulfill({ json: { code: 0, data } })
  })
  await page.goto(`${origin}${fixturePath}${query}`)
  await page.locator('html[data-ready="true"]').waitFor()
  await page.locator('.maplibregl-canvas').waitFor()
  // Canvas/labels exist before MapLibre's load event; screenshots must show the rendered map.
  await page.locator('.atlas-map-status').waitFor({ state: 'hidden' })
  return { page, errors }
}
async function query(panel, week, empty = false) {
  await panel.locator('.atlas-schedule__filters select').first().selectOption('2')
  await panel.locator('input[type="number"]').fill(String(week))
  await panel.locator('.atlas-schedule__submit:not(:disabled)').click()
  if (empty) await panel.locator('.atlas-schedule__status').filter({ hasText: '没有已同步' }).waitFor()
  else await panel.locator('.atlas-schedule__list button').first().waitFor()
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
  test(`override status, unknown text, online members and cross-term stable matching at ${width}px`, async () => {
    const { page, errors } = await open(width, '?mine=1')
    const result = { scenario: 'override status and cross-term stable matching', width, status: 'failed' }
    receipt.tests.push(result)
    try {
      await page.locator('.atlas-mine__views button').nth(1).click()
      const panel = page.locator('.atlas-mine .atlas-schedule')
      await query(panel, 1)
      for (const [index, expected] of [[0, '无法可靠解析'], [2, '有待复核'], [4, '线上课程']]) {
        await panel.locator('.atlas-schedule__list button').filter({ hasText: courses[index].name }).click()
        const choices = panel.locator('.atlas-location-choices')
        assert.ok((await choices.innerText()).includes(expected))
        assert.equal(await choices.locator('button:enabled').count(), 0)
        assert.equal(new URL(page.url()).hash, '')
        await choices.scrollIntoViewIfNeeded()
        await capture(page, `status-${index}-${width}.png`, { raw: courses[index].room })
      }
      await panel.locator('select').first().selectOption('121')
      await query(panel, 1)
      await panel.locator('.atlas-schedule__list button').filter({ hasText: courses[1].name }).click()
      assert.equal(await panel.locator('.atlas-location-choices button:enabled').count(), 2)
      assert.ok((await panel.locator('.atlas-location-choices').innerText()).includes('济事楼'))
      await capture(page, `term-stable-${width}.png`, { calendarId: 121, raw: courses[1].room })
      assert.deepEqual(errors, [])
      result.status = 'passed'
    } finally { await page.close() }
  })
  test(`parsed time conditions constrain building schedules at ${width}px`, async () => {
    const { page, errors } = await open(width, `?campus=jiading#place=${encodeURIComponent(targets.jishi.id)}`)
    const result = { scenario: 'parsed time conditions', width, status: 'failed' }
    receipt.tests.push(result)
    try {
      await page.locator('.atlas-schedule-open').click()
      const panel = page.locator('dialog[open]')
      await query(panel, 1)
      const entry = panel.locator('.atlas-schedule__list button')
      assert.equal(await entry.count(), 1)
      await entry.click()
      const choices = panel.locator('.atlas-location-choices button')
      assert.equal(await choices.count(), 2)
      assert.equal((await panel.locator('.atlas-location-choices').innerText()).includes('仍待复核'), false)
      assert.ok((await choices.nth(0).innerText()).includes('1-2周'))
      assert.ok((await choices.nth(1).innerText()).includes('3-16周'))
      assert.equal(new URL(page.url()).hash, '')
      await choices.nth(0).click()
      await selectedTarget(page, targets.jishi)
      await capture(page, `time-week-1-${width}.png`, { week: 1, raw: courses[1].room })
      await query(panel, 17, true)
      assert.equal(await panel.locator('.atlas-schedule__list button').count(), 0)
      await capture(page, `time-week-17-${width}.png`, { week: 17 })
      assert.deepEqual(errors, [])
      result.status = 'passed'
    } finally { await page.close() }
  })
  test(`parsed members select distinct real buildings at ${width}px`, async () => {
    const { page, errors } = await open(width, `#place=${encodeURIComponent(targets.rui.id)}`)
    const result = { scenario: 'multiple parsed members', width, status: 'failed' }
    receipt.tests.push(result)
    try {
      await page.locator('.atlas-schedule-open').click()
      const panel = page.locator('dialog[open]')
      const scope = await panel.locator('.atlas-schedule__scope').innerText()
      await query(panel, 1)
      const entry = panel.locator('.atlas-schedule__list button')
      assert.equal(await entry.count(), 1)
      await entry.click()
      const choices = panel.locator('.atlas-location-choices button')
      assert.equal(await choices.count(), 6)
      assert.equal(new URL(page.url()).hash, '', 'multiple members must not choose an initial pin')
      await capture(page, `multiple-unselected-${width}.png`, { raw: courses[3].room })
      for (const [index, target] of [[0, targets.rui], [3, targets.physics]]) {
        await choices.nth(index).click()
        await selectedTarget(page, target)
        assert.equal(await choices.nth(index).getAttribute('aria-pressed'), 'true')
        assert.equal(await panel.locator('.atlas-schedule__scope').innerText(), scope)
        await capture(page, `multiple-${index}-${width}.png`, { selectedTarget: target })
      }
      const boxes = await choices.evaluateAll(items => items.map(item => item.getBoundingClientRect().height))
      assert.ok(boxes.every(height => height >= 44))
      await mapCapture(page, `multiple-map-${width}.png`, targets.physics)
      assert.deepEqual(errors, [])
      result.status = 'passed'
    } finally { await page.close() }
  })
  test(`date and parity suffixes retain original text and constrain scoped queries at ${width}px`, async () => {
    const result = { scenario: 'review date/parity suffix counterexamples', width, status: 'failed' }
    receipt.tests.push(result)
    const unscoped = await open(width, '?mine=1')
    try {
      await unscoped.page.locator('.atlas-mine__views button').nth(1).click()
      const panel = unscoped.page.locator('.atlas-mine .atlas-schedule')
      await query(panel, 2)
      await panel.locator('.atlas-schedule__list button').filter({ hasText: courses[5].name }).click()
      assert.ok((await panel.locator('.atlas-location-choices').innerText()).includes(courses[5].room))
      await selectedTarget(unscoped.page, targets.north)
      await capture(unscoped.page, `date-original-${width}.png`, { week: 2, raw: courses[5].room, selectedTarget: targets.north })
      assert.deepEqual(unscoped.errors, [])
    } finally { await unscoped.page.close() }
    const { page, errors } = await open(width, `#place=${encodeURIComponent(targets.north.id)}`)
    try {
      await page.locator('.atlas-schedule-open').click()
      const panel = page.locator('dialog[open]')
      await query(panel, 1)
      const entries = panel.locator('.atlas-schedule__list button')
      assert.equal(await entries.count(), 1, 'the date-only course cannot confirm an arrangement')
      assert.ok((await entries.innerText()).includes(courses[6].name))
      await entries.click()
      assert.ok((await panel.locator('.atlas-location-choices').innerText()).includes(courses[6].room))
      await selectedTarget(page, targets.north)
      await capture(page, `parity-odd-${width}.png`, { week: 1, raw: courses[6].room, selectedTarget: targets.north })
      await query(panel, 2, true)
      assert.equal(await panel.locator('.atlas-schedule__list button').count(), 0)
      await capture(page, `date-parity-excluded-${width}.png`, { week: 2, excluded: [courses[5].room, courses[6].room] })
      assert.deepEqual(errors, [])
      result.status = 'passed'
    } finally { await page.close() }
  })
  test(`A101 and B201 choose their own verified buildings at ${width}px`, async () => {
    const { page, errors } = await open(width, `?campus=jiading#place=${encodeURIComponent(targets.an.id)}`)
    const result = { scenario: 'review cross-building abbreviation counterexample', width, status: 'failed' }
    receipt.tests.push(result)
    try {
      await selectedTarget(page, targets.an)
      await capture(page, `abbreviations-map-initial-${width}.png`, { view: 'map', selectedTarget: targets.an })
      await page.locator('.atlas-schedule-open').click()
      const panel = page.locator('dialog[open]')
      const scope = await panel.locator('.atlas-schedule__scope').innerText()
      await query(panel, 2)
      const entry = panel.locator('.atlas-schedule__list button')
      assert.equal(await entry.count(), 1)
      await entry.click()
      const choices = panel.locator('.atlas-location-choices button')
      assert.equal(await choices.count(), 2)
      assert.equal(new URL(page.url()).hash, '')
      await capture(page, `abbreviations-unselected-${width}.png`, { week: 2, raw: courses[7].room })
      for (const [index, target] of [[0, targets.an], [1, targets.bo]]) {
        await choices.nth(index).click()
        await selectedTarget(page, target)
        assert.equal(await choices.nth(index).getAttribute('aria-pressed'), 'true')
        assert.equal(await panel.locator('.atlas-schedule__scope').innerText(), scope)
        await capture(page, `abbreviations-${index}-${width}.png`, { week: 2, raw: courses[7].room, selectedTarget: target })
      }
      await mapCapture(page, `abbreviations-map-final-${width}.png`, targets.bo)
      assert.deepEqual(errors, [])
      result.status = 'passed'
    } finally { await page.close() }
  })
}
