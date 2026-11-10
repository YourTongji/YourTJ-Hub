import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { mkdtemp, mkdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server, browser, origin, cacheDir
before(async () => {
  cacheDir = await mkdtemp(join(tmpdir(), 'wiki-actions-vite-'))
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), cacheDir, server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch({ executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH || undefined })
})
after(async () => {
  await browser?.close()
  await server?.close()
  if (cacheDir) await rm(cacheDir, { recursive: true, force: true })
})

const cases = [
  { name: 'like', index: 0, activeClass: 'text-error' },
  { name: 'bookmark', index: 1, activeClass: 'text-primary' },
  { name: 'watch', index: 2, activeClass: 'text-success' },
]
async function setup(width) {
  const page = await browser.newPage({ viewport: { width, height: 900 } })
  page.setDefaultTimeout(15000)
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  const requests = []
  const held = []
  await page.route('**/api/**', async route => {
    const url = new URL(route.request().url())
    if (/\/api\/forum\/topics\/(like|bookmark|watch)$/.test(url.pathname)) {
      requests.push({ name: url.pathname.split('/').at(-1), ...route.request().postDataJSON() })
      held.push(route)
    } else {
      await route.fulfill({ json: { code: 0, result: { posts: [], replyTargets: [], hasBefore: false, hasAfter: false, total: 0, maxPostNo: 0 } } })
    }
  })
  await page.goto(`${origin}/assets/test/fixtures/browser/wiki-page-actions.html`)
  await page.getByRole('heading', { name: 'Wiki 互动状态测试' }).waitFor()
  const card = page.locator('section.gf-card').first()
  const button = index => card.locator('button:visible').nth(index)
  async function release(failed = false) {
    const deadline = Date.now() + 15000
    while (!held.length && Date.now() < deadline) await new Promise(resolve => setTimeout(resolve, 10))
    assert.ok(held.length, 'the mocked request must have reached the browser route')
    await held.shift().fulfill({ status: failed ? 403 : 200, json: failed ? { code: 1, message: 'Fixture request rejected' } : { code: 0, result: true } })
  }
  const settle = () => page.waitForFunction(() => !document.querySelector('section.gf-card button[disabled]'))
  return { page, card, button, requests, release, settle, errors }
}

for (const action of cases) {
  for (const startWidth of [375, 1440]) {
    test(`${action.name}: ${startWidth}px -> other breakpoint, success, cancellation failure and retry`, async () => {
      const { page, card, button, requests, release, settle, errors } = await setup(startWidth)
      try {
        await button(action.index).click()
        assert.equal(await card.locator('button[disabled]').count(), 2)
        await page.setViewportSize({ width: startWidth === 375 ? 1440 : 375, height: 900 })
        assert.equal(await button(action.index).isDisabled(), true)
        assert.match(await button(action.index).getAttribute('class'), new RegExp(action.activeClass))
        for (const other of cases.filter(other => other !== action)) assert.equal(await button(other.index).isEnabled(), true)
        await release()
        await settle()
        assert.equal(requests.length, 1)
        assert.match(await button(action.index).getAttribute('class'), new RegExp(action.activeClass))
        if (process.env.WIKI_ACTIONS_BROWSER_EVIDENCE_DIR) {
          await mkdir(process.env.WIKI_ACTIONS_BROWSER_EVIDENCE_DIR, { recursive: true })
          await page.screenshot({ path: join(process.env.WIKI_ACTIONS_BROWSER_EVIDENCE_DIR, `${action.name}-from-${startWidth}.png`), fullPage: true })
        }
        await button(action.index).click()
        await page.setViewportSize({ width: startWidth, height: 900 })
        await release(true)
        await settle()
        assert.match(await button(action.index).getAttribute('class'), new RegExp(action.activeClass), 'failed cancellation rolls back across the breakpoint')
        await button(action.index).click()
        await release()
        await settle()
        assert.doesNotMatch(await button(action.index).getAttribute('class'), new RegExp(action.activeClass))
        assert.deepEqual(requests, [1, 2, 2].map(value => ({ name: action.name, topicId: 42, action: value })))
        if (action.name === 'like') assert.equal((await button(0).innerText()).trim(), '4')
        assert.deepEqual(errors, [])
      } finally { await page.close() }
    })
  }
}
