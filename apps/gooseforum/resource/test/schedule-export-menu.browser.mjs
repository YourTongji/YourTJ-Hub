import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server
let browser
let origin
before(async () => {
  server = await createServer({
    root: fileURLToPath(new URL('../', import.meta.url)),
    server: { host: '127.0.0.1', port: 0, open: false },
  })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => {
  await browser?.close()
  await server?.close()
})

async function openPage(width, lang, outdated) {
  const page = await browser.newPage({ viewport: { width, height: 900 } })
  await page.route('**/api/**', route => route.fulfill({ json: { code: 0, msg: '', data: [] } }))
  await page.goto(`${origin}/assets/test/fixtures/browser/schedule.html?lang=${lang}`)
  await page.waitForSelector('html[data-ready="true"]')
  await page.waitForLoadState('networkidle')
  await page.evaluate(async value => {
    const { useScheduleStore } = await import('/assets/src/site/composables/useScheduleStore.ts')
    useScheduleStore().setDataOutdated(value)
  }, outdated)
  return page
}

async function assertMenuInViewport(page) {
  const menu = page.locator('.gf-menu-surface')
  await menu.waitFor({ state: 'visible' })
  // Wait for the opening transition / floating-position update, without disabling either.
  await page.waitForTimeout(250)
  const box = await menu.boundingBox()
  assert.ok(box, 'export menu is visible')
  const { width, height } = page.viewportSize()
  assert.ok(box.x >= 0 && box.x + box.width <= width,
    `menu [${box.x}, ${box.x + box.width}] must fit viewport width ${width}`)
  assert.ok(box.y >= 0 && box.y + box.height <= height, 'menu fits viewport height')
}

for (const lang of ['zh', 'en', 'de', 'ja']) {
  for (const width of [320, 360, 375, 640, 1280]) {
    for (const outdated of [false, true]) {
      test(`export menu fits ${lang} ${width}px, sync button ${outdated ? 'shown' : 'hidden'}`, async () => {
        const page = await openPage(width, lang, outdated)
        try {
          await page.locator('[aria-haspopup="menu"]').click()
          await assertMenuInViewport(page)
        } finally {
          await page.close()
        }
      })
    }
  }
}

test('open menu repositions on resize and restores trigger focus on Escape', async () => {
  const page = await openPage(1280, 'en', true)
  try {
    const trigger = page.locator('[aria-haspopup="menu"]')
    await trigger.focus()
    await page.keyboard.press('Enter')
    await assertMenuInViewport(page)
    await page.setViewportSize({ width: 320, height: 900 })
    await assertMenuInViewport(page)
    await page.keyboard.press('Escape')
    await page.locator('.gf-menu-surface').waitFor({ state: 'hidden' })
    await page.waitForFunction(() => document.activeElement?.getAttribute('aria-haspopup') === 'menu')
    assert.equal(await trigger.getAttribute('aria-expanded'), 'false')
  } finally {
    await page.close()
  }
})

for (const format of ['csv', 'xls']) {
  test(`${format} menu action downloads selected courses and restores focus`, async () => {
    const page = await openPage(320, 'en', true)
    try {
      await page.evaluate(async () => {
        const { useScheduleStore } = await import('/assets/src/site/composables/useScheduleStore.ts')
        useScheduleStore().pushStagedCourse({
          courseCode: 'MATH101', courseName: 'MATH101', courseNameReserved: 'Calculus',
          credit: 2, courseType: '', teacher: [], status: 1,
          courseDetail: [{
            code: 'MATH101.01', status: 1, teachers: [], campus: '', teachingLanguage: '',
            arrangementInfo: [{
              occupyDay: 1, occupyTime: [1, 2], occupyWeek: [1, 2], occupyRoom: 'A101',
              teacherAndCode: '', arrangementText: '',
            }],
          }],
        })
      })
      await page.locator('[aria-haspopup="menu"]').click()
      const downloaded = page.waitForEvent('download')
      await page.getByRole('menuitem').nth(format === 'csv' ? 0 : 1).click()
      const download = await downloaded
      assert.equal(download.suggestedFilename(), `yourtj-schedule.${format}`)
      const stream = await download.createReadStream()
      let text = ''
      for await (const chunk of stream) text += chunk.toString('utf8')
      assert.ok(text.includes('Calculus'), 'download contains the selected course')
      await page.locator('.gf-menu-surface').waitFor({ state: 'hidden' })
      await page.waitForFunction(() => document.activeElement?.getAttribute('aria-haspopup') === 'menu')
      assert.equal(await page.evaluate(() => getComputedStyle(document.body).overflow), 'visible')
    } finally {
      await page.close()
    }
  })
}

test('outside click closes menu without stealing focus', async () => {
  const page = await openPage(320, 'en', true)
  try {
    await page.locator('[aria-haspopup="menu"]').click()
    await page.locator('#schedule-tab-list').click()
    await page.locator('.gf-menu-surface').waitFor({ state: 'hidden' })
    assert.equal(await page.evaluate(() => document.activeElement?.id), 'schedule-tab-list')
  } finally {
    await page.close()
  }
})
