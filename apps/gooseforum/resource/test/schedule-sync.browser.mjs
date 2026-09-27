import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server, browser, origin
before(async () => {
  server = await createServer({
    root: fileURLToPath(new URL('../', import.meta.url)),
    server: { host: '127.0.0.1', port: 0, open: false },
  })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })

for (const width of [320, 1280]) {
  test(`per-field conflict choice preserves independent edits at ${width}px`, async () => {
    const page = await browser.newPage({ viewport: { width, height: 900 } })
    page.setDefaultTimeout(15000)
    let remote = {
      plan: { id: 'p', name: 'Original', createdAt: 1, stagedCourses: [], selectedCourses: [],
        customEvents: [{ id: 'e', label: 'Original event', day: 1, sections: [1], weeks: [1] }] },
      revision: 1, updatedAt: '2026-09-22T00:00:00Z',
    }
    const writes = []
    await page.route('**/api/**', async route => {
      if (!route.request().url().endsWith('/api/pk/plan-items')) {
        await route.fulfill({ json: { code: 0, msg: '', data: [] } }); return
      }
      if (route.request().method() === 'PUT') {
        const body = route.request().postDataJSON()
        writes.push(body)
        assert.equal(body.baseRevision, remote.revision)
        remote = { ...remote, plan: body.plan, revision: remote.revision + 1 }
        await route.fulfill({ json: { code: 0, msg: '', data: remote } }); return
      }
      await route.fulfill({ json: { code: 0, msg: '', data: [remote] } })
    })
    try {
      await page.goto(`${origin}/assets/test/fixtures/browser/schedule.html?lang=en`)
      await page.waitForSelector('html[data-ready="true"]')
      await page.evaluate(async () => {
        const { startScheduleSync, scheduleSync } = await import('/assets/src/site/composables/useScheduleSync.ts')
        startScheduleSync(7)
        await scheduleSync.syncOnPageEnter()
        const { useScheduleStore } = await import('/assets/src/site/composables/useScheduleStore.ts')
        const store = useScheduleStore()
        store.renamePlan('p', 'Local title')
        const plan = JSON.parse(JSON.stringify(store.state.plans[0]))
        plan.customEvents[0].label = 'Local event'
        store.applyPlanItems([plan])
        scheduleSync.onLocalChange()
      })
      remote = { ...remote, revision: 2, plan: { ...remote.plan, name: 'Remote title',
        customEvents: [{ ...remote.plan.customEvents[0], day: 2 }] } }
      await page.evaluate(async () => {
        const { scheduleSync } = await import('/assets/src/site/composables/useScheduleSync.ts')
        await scheduleSync.syncOnPageEnter()
      })
      const panel = page.locator('section[aria-live="polite"]')
      await panel.waitFor({ state: 'visible' })
      assert.equal(await panel.locator('fieldset').count(), 1, 'only name overlaps')
      const apply = panel.getByRole('button', { name: 'Merge and save' })
      assert.equal(await apply.isDisabled(), true)
      const remoteChoice = panel.locator('input[value="remote"]')
      await remoteChoice.focus()
      await page.keyboard.press('Space')
      assert.equal(await apply.isEnabled(), true)
      const bounds = await panel.boundingBox()
      assert.ok(bounds.x >= 0 && bounds.x + bounds.width <= width, 'conflict panel fits viewport')
      await apply.click()
      await panel.waitFor({ state: 'hidden' })
      await page.waitForFunction(async () => {
        const { scheduleSync } = await import('/assets/src/site/composables/useScheduleSync.ts')
        return !scheduleSync.isDirty()
      })
      assert.equal(writes.length, 1)
      assert.equal(writes[0].plan.name, 'Remote title')
      assert.equal(writes[0].plan.customEvents[0].label, 'Local event')
      assert.equal(writes[0].plan.customEvents[0].day, 2)
      assert.equal(writes[0].plan.id, 'p', 'merge does not clone the plan')
    } finally { await page.close() }
  })
}
