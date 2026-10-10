import assert from 'node:assert/strict'
import { mkdir } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { before, after, test } from 'node:test'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server, browser, origin
const screenshots = join(tmpdir(), 'yourtj-for-you-session')
before(async () => {
  await mkdir(screenshots, { recursive: true })
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen(); origin = `http://127.0.0.1:${server.httpServer.address().port}`; browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })

const card = (id, likes = 0) => ({ id, title: `推荐帖子 ${id}`, description: '校园里的新讨论，读者可以继续原来的浏览位置。'.repeat(3),
  url: `/p/post/${id}`, author: { id: 1000 + id, username: `作者 ${id}`, avatarUrl: '/assets/test/fixtures/browser/topic-feed-art.svg' }, participants: [], categories: [], replyCount: 3,
  likeCount: likes, viewCount: 12, activityText: '刚刚', lastUpdateTime: '2026-10-07T00:00:00Z', pinWeight: 0, processStatus: 0, contentType: 0,
  liked: likes > 0, bookmarked: false, feedTrace: 'trace-original', feedPosition: (id - 1) % 20 })
const proof = (ids, token = `proof-${ids[0]}`) => ({ token, topicIds: ids, issuedAt: Date.now(), expiresAt: Date.now() + 1_800_000 })
async function setupPage(width, { failRefresh = false, expired = false, empty = false, delayedContinuation = false } = {}) {
  const page = await browser.newPage({ viewport: { width, height: 900 } })
  page.setDefaultTimeout(10000)
  const errors = []
  page.on("pageerror", (err) => errors.push(err.message))
  const state = { firstPages: 0, refreshes: [], reconciles: 0, seen: new Set(), tokens: new Map(), likes: 0, continuationStarted: false, removed: [] }
  state.tokens.set('proof-1', Array.from({ length: 20 }, (_, i) => i + 1))
  await page.addInitScript(() => {
    class Events {
      static instances = []
      constructor() { this.handlers = {}; Events.instances.push(this) }
      addEventListener(key, fn) { this.handlers[key] = fn; if (key === 'hello') setTimeout(fn, 20) }
      close() {}
    }
    window.EventSource = Events
    window.feedHello = () => Events.instances.at(-1)?.handlers.hello?.()
  })
  const applySeen = (patches = []) => { for (const patch of patches) { const ids = state.tokens.get(patch.proof) ?? []; for (const pos of Object.keys(patch.seen)) if (ids[Number(pos)]) state.seen.add(ids[Number(pos)]) } }
  const json = (route, body, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
  await page.route('**/api/forum/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    const data = route.request().postDataJSON() ?? {}
    if (path.startsWith('/api/forum/test-')) { state.likes++; return json(route, { code: 0, result: true }) }
    if (path.endsWith('/feed/events')) { applySeen(data.seenPatches); return json(route, { code: 0, result: data.seenPatches?.length ? { seenConfirmed: true } : true }) }
    if (path.endsWith('/feed/reconcile')) {
      state.reconciles++
      const ids = data.topicIds.filter((id) => !state.removed.includes(id))
      const proofs = []
      for (let i = 0; i < ids.length; i += 20) { const group = ids.slice(i, i + 20); const token = `reconciled-${state.reconciles}-${i}`; state.tokens.set(token, group); proofs.push(proof(group, token)) }
      return json(route, { code: 0, result: { viewerId: 12, topics: ids.map((id) => card(id, state.likes)), removedIds: data.topicIds.filter((id) => state.removed.includes(id)), seenProofs: proofs, snapshotId: '', available: true, seenConfirmed: false } })
    }
    if (path.endsWith('/feed/refresh')) {
      state.refreshes.push(data); applySeen(data.seenPatches)
      if (failRefresh) return json(route, { code: 1 }, 503)
      const ids = Array.from({ length: 60 }, (_, i) => i + 1).filter((id) => !state.seen.has(id)).slice(0, 20)
      return json(route, { code: 0, result: { viewerId: 12, topics: ids.map((id) => card(id)), removedIds: [], seenProofs: ids.length ? [proof(ids, 'fresh')] : [], snapshotId: 'new-batch', available: true, seenConfirmed: true, pagination: { page: 1, nextPage: 2, hasNext: false, nextUrl: '' } } })
    }
    return json(route, { code: 0, result: {} })
  })
  await page.route(/\/p\/post\/\d+/, (route) => json(route, { component: 'topic.detail', props: {}, layout: { viewer: { id: 12 } }, url: new URL(route.request().url()).pathname, version: '1.0' }))
  await page.route(/\/?\?sort=for_you/, async (route) => {
    if (!route.request().headers()['x-goose-page']) return route.continue()
    const url = new URL(route.request().url()); const offset = Number(url.searchParams.get('cursor')?.replace('next-', '') ?? 0)
    if (!offset) state.firstPages++
    if (offset && delayedContinuation) { state.continuationStarted = true; await new Promise((resolve) => setTimeout(resolve, 700)) }
    if (expired && offset) return json(route, { code: 1, errorCode: 'feed_snapshot_expired' }, 409)
    const ids = Array.from({ length: 20 }, (_, i) => offset + i + 1); const token = `proof-${offset + 1}`; state.tokens.set(token, ids)
    return json(route, { component: 'home.index', url: '/?sort=for_you', version: '1.0', layout: { viewer: { id: 12 } }, props: {
      sort: 'for_you', actualSort: 'for_you', topics: ids.map((id) => card(id)), seenProofs: [proof(ids, token)], snapshotId: 'original',
      tabs: [], announcement: { enabled: false, html: '', items: [] }, pagination: { page: offset / 20 + 1, nextPage: offset / 20 + 2, hasNext: offset < 40, nextUrl: `/?sort=for_you&cursor=next-${offset + 20}` },
    } })
  })
  await page.goto(`${origin}/assets/test/fixtures/browser/for-you-session.html${empty ? '?empty=1' : ''}`)
  try { await page.waitForFunction(() => document.documentElement.dataset.ready === 'true') }
  catch (err) { throw new Error(`browser fixture failed to initialize: ${errors.join('; ')}; ${err.message}`) }
  return { page, state }
}

for (const width of [390, 1440]) {
  test(`For You preserves three loaded pages, order and anchor after detail actions at ${width}px`, async () => {
    const { page, state } = await setupPage(width)
    try {
      for (let i = 0; i < 3; i++) { await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight)); await page.waitForTimeout(180) }
      await page.locator('[data-feed-id="60"]').waitFor()
      const before = await page.locator('[data-feed-id]').evaluateAll((rows) => rows.map((r) => Number(r.dataset.feedId)))
      await page.locator('[data-feed-id="31"]').evaluate((row) => window.scrollBy(0, row.getBoundingClientRect().top - 80))
      const top = await page.locator('[data-feed-id="31"]').evaluate((row) => row.getBoundingClientRect().top)
      await page.locator('[data-feed-id="31"] a[href="/p/post/31"]').first().click()
      await page.locator('[data-testid="detail"]').waitFor()
      for (const action of ['like', 'bookmark', 'comment']) await page.locator(`[data-action="${action}"]`).click()
      await page.goBack(); await page.locator('[data-feed-id="60"]').waitFor(); await page.waitForTimeout(300)
      assert.deepEqual(await page.locator('[data-feed-id]').evaluateAll((rows) => rows.map((r) => Number(r.dataset.feedId))), before)
      assert.ok(Math.abs(await page.locator('[data-feed-id="31"]').evaluate((row) => row.getBoundingClientRect().top) - top) < 3)
      assert.equal(state.firstPages, 0); assert.equal(state.refreshes.length, 0)
      await page.evaluate(() => window.feedHello()); await page.waitForTimeout(250)
      assert.equal(state.refreshes.length, 0); assert.ok(state.reconciles >= 2)
      await page.evaluate(() => document.fonts.ready)
      await page.waitForFunction(() => document.querySelector('[data-feed-id="31"]')?.getBoundingClientRect().top < 200)
      assert.ok(await page.locator('[data-feed-id="31"]').evaluate((row) => document.elementsFromPoint(row.getBoundingClientRect().left + 20, row.getBoundingClientRect().top + 20).some((el) => el.closest('[data-feed-id="31"]'))), "restored anchor must remain painted and unobscured")
      await page.screenshot({ path: join(screenshots, `returned-${width}.png`), animations: "disabled" })
    } finally { await page.close() }
  })
}

test('immediate refresh carries visible claims and filters them; failure preserves the batch', async () => {
  for (const failRefresh of [false, true]) {
    const { page, state } = await setupPage(390, { failRefresh })
    try {
      await page.waitForTimeout(1500)
      const before = await page.locator('[data-feed-id]').evaluateAll((rows) => rows.map((r) => Number(r.dataset.feedId)))
      await page.locator('.gf-home-refresh-button').click(); await page.waitForTimeout(300)
      assert.equal(state.refreshes.length, 1)
      assert.ok(state.refreshes[0].seenPatches.length > 0, 'one-second exposure must reach immediate refresh before five-second flush')
      const after = await page.locator('[data-feed-id]').evaluateAll((rows) => rows.map((r) => Number(r.dataset.feedId)))
      if (failRefresh) assert.deepEqual(after, before)
      else for (const id of after) assert.ok(!state.seen.has(id), `seen topic ${id} resurrected`)
    } finally { await page.close() }
  }
})

test('expired cursor retains loaded cards and asks for explicit refresh', async () => {
  const { page, state } = await setupPage(390, { expired: true })
  try {
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    const expiredNotice = page.getByText('这批推荐已过期', { exact: false })
    await expiredNotice.waitFor()
    await expiredNotice.scrollIntoViewIfNeeded()
    assert.equal(await page.locator('[data-feed-id]').count(), 20)
    assert.equal(state.refreshes.length, 0)
    assert.equal(await page.getByText('加载更多', { exact: true }).count(), 0)
    assert.ok(await expiredNotice.evaluate((el) => el.getBoundingClientRect().top) < 900)
    await page.screenshot({ path: join(screenshots, 'expired-390.png') })
  } finally { await page.close() }
})

test('empty recommendation pool has bounded wording and a usable refresh', async () => {
  const { page } = await setupPage(390, { empty: true })
  try {
    await page.getByText('这一批已经看完了').waitFor()
    assert.equal(await page.locator('.gf-home-refresh-button').count(), 1)
    await page.screenshot({ path: join(screenshots, 'empty-390.png') })
  } finally { await page.close() }
})

test('a fresh batch cannot receive a delayed old continuation', async () => {
  const { page, state } = await setupPage(390, { delayedContinuation: true })
  try {
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    await page.waitForFunction(() => document.querySelector('.gf-home-refresh-button'))
    while (!state.continuationStarted) await page.waitForTimeout(20)
    await page.locator('.gf-home-refresh-button').evaluate((button) => button.click())
    await page.waitForTimeout(900)
    assert.equal(state.refreshes.length, 1)
    assert.equal(await page.locator('[data-feed-id]').count(), 20)
    assert.equal(await page.locator('[data-feed-id="40"]').count(), 0)
  } finally { await page.close() }
})

test('a removed scroll anchor advances to the next surviving card without reranking', async () => {
  const { page, state } = await setupPage(390)
  try {
    await page.locator('[data-feed-id="10"]').evaluate((row) => window.scrollBy(0, row.getBoundingClientRect().top - 80))
    await page.locator('[data-feed-id="10"] a[href="/p/post/10"]').first().click()
    await page.locator('[data-testid="detail"]').waitFor()
    state.removed = [10]
    await page.goBack()
    await page.waitForFunction(() => !document.querySelector('[data-feed-id="10"]'))
    assert.ok(Math.abs(await page.locator('[data-feed-id="11"]').evaluate((row) => row.getBoundingClientRect().top) - 80) < 3)
    assert.equal(state.firstPages, 0); assert.equal(state.refreshes.length, 0)
  } finally { await page.close() }
})

test('return after more than thirty seconds without a detail action preserves the batch', async () => {
  const { page, state } = await setupPage(390)
  try {
    await page.locator('[data-feed-id="10"]').evaluate((row) => window.scrollBy(0, row.getBoundingClientRect().top - 80))
    await page.locator('[data-feed-id="10"] a[href="/p/post/10"]').first().click()
    await page.locator('[data-testid="detail"]').waitFor()
    await page.waitForTimeout(31000)
    await page.goBack(); await page.locator('[data-feed-id="20"]').waitFor(); await page.waitForTimeout(250)
    assert.equal(await page.locator('[data-feed-id]').count(), 20)
    assert.ok(Math.abs(await page.locator('[data-feed-id="10"]').evaluate((row) => row.getBoundingClientRect().top) - 80) < 3)
    assert.equal(state.firstPages, 0); assert.equal(state.refreshes.length, 0)
  } finally { await page.close() }
})
