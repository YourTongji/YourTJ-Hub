import assert from 'node:assert/strict'
import { fileURLToPath } from 'node:url'
import { after, before, test } from 'node:test'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server, browser, origin
const layout = { viewer: { id: 12, isAuthenticated: true, requiresEmailVerification: false } }

function topic(id, title = `返回测试话题 ${id}`, feedTrace = `trace-${id}`) {
  return {
    id, title,
    description: '这是用于验证详情返回后分页列表与滚动位置的测试话题。'.repeat(2),
    url: `/p/post/${id}`,
    author: { id: 1000 + id, username: `作者 ${id}`, avatarUrl: '/assets/test/fixtures/browser/topic-feed-art.svg' },
    participants: [], categories: [{ id: 7, name: '校园生活', url: '/c/7', color: '#3b82f6' }],
    replyCount: 3, likeCount: 0, viewCount: 12, activityText: '刚刚', lastUpdateTime: '2026-10-07T00:00:00Z',
    pinWeight: 0, processStatus: 0, contentType: 0, liked: false, bookmarked: false,
    feedTrace, feedPosition: id - 1,
  }
}

function homePayload(ids, { page, nextCursor, titleUpdates = {}, sort = 'latest', viewerId = 12, tracePrefix = 'trace' } = {}) {
  return {
    component: 'home.index', url: `/?sort=${sort}`, version: '1.0', layout: { viewer: { ...layout.viewer, id: viewerId } },
    props: {
      sort, actualSort: sort,
      tabs: [{ key: 'latest', active: sort === 'latest', url: '/?sort=latest' }, { key: 'hot', active: sort === 'hot', url: '/?sort=hot' }],
      topics: ids.map((id) => topic(id, titleUpdates[id] ?? `返回测试话题 ${id}`, `${tracePrefix}-${id}`)),
      pagination: { page, nextPage: page + 1, hasNext: nextCursor !== null, nextUrl: nextCursor === null ? '' : `/?sort=${sort}&cursor=next-${nextCursor}` },
      announcement: { enabled: false, html: '', items: [] },
    },
  }
}

function categoryPayload(ids, { page, nextCursor } = {}) {
  return {
    component: 'category.index', url: '/c/7', version: '1.0', layout,
    props: {
      category: { id: 7, name: '校园生活', description: '分类返回验证', icon: '#', color: '#3b82f6', url: '/c/7' },
      sort: 'latest', tabs: [{ key: 'latest', active: true, url: '/c/7' }],
      topics: ids.map((id) => topic(id)),
      pagination: { page, nextPage: page + 1, hasNext: nextCursor !== null, nextUrl: nextCursor === null ? '' : `/c/7?cursor=next-${nextCursor}` },
    },
  }
}

before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })

async function openFeed({ resync = false } = {}) {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
  page.setDefaultTimeout(10000)
  const state = { firstPageRequests: 0, categoryRequests: 0, cursors: [], viewerId: 12, removed: [] }
  await page.addInitScript(() => {
    class ControlledIntersectionObserver {
      observe() {}
      disconnect() {}
      unobserve() {}
      takeRecords() { return [] }
    }
    window.IntersectionObserver = ControlledIntersectionObserver
    class ControlledEventSource extends EventTarget {
      static current
      constructor() { super(); ControlledEventSource.current = this }
      close() {}
    }
    window.EventSource = ControlledEventSource
    window.dispatchFeedHello = () => ControlledEventSource.current?.dispatchEvent(new Event('hello'))
  })
  await page.route('**/*', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    if (url.pathname === '/api/forum/feed/reconcile') {
      const ids = request.postDataJSON().topicIds
      return route.fulfill({ json: { code: 0, result: {
        viewerId: state.viewerId, topics: [],
        removedIds: ids.filter((id) => state.removed.includes(id)), seenProofs: [], available: true,
      } } })
    }
    if (!request.headers()['x-goose-page']) return route.continue()
    if (/^\/p\/post\/\d+$/.test(url.pathname)) {
      return route.fulfill({ json: { component: 'topic.detail', props: {}, layout, url: url.pathname, version: '1.0' } })
    }
    if (url.pathname === '/c/7') {
      const cursor = url.searchParams.get('cursor')
      if (cursor) {
        const offset = Number(cursor.replace('next-', ''))
        state.cursors.push(cursor)
        return route.fulfill({ json: categoryPayload(Array.from({ length: 20 }, (_, index) => 200 + offset + index + 1), { page: offset / 20 + 1, nextCursor: offset < 40 ? offset + 20 : null }) })
      }
      state.categoryRequests++
      return route.fulfill({ json: categoryPayload(Array.from({ length: 20 }, (_, index) => 201 + index), { page: 1, nextCursor: 20 }) })
    }
    if (url.pathname !== '/') return route.fulfill({ status: 404, json: {} })
    const sort = url.searchParams.get('sort') ?? 'latest'
    const cursor = url.searchParams.get('cursor')
    if (cursor) {
      state.cursors.push(cursor)
      const offset = Number(cursor.replace('next-', ''))
      const baseID = sort === 'hot' ? 100 : 0
      const ids = Array.from({ length: 20 }, (_, index) => baseID + offset + index + 1)
      return route.fulfill({ json: homePayload(ids, { page: offset / 20 + 1, nextCursor: offset < 60 ? offset + 20 : null, sort, viewerId: state.viewerId }) })
    }
    if (sort === 'hot') return route.fulfill({ json: homePayload(Array.from({ length: 20 }, (_, index) => 101 + index), { page: 1, nextCursor: 20, sort, viewerId: state.viewerId }) })
    state.firstPageRequests++
    if (resync && state.firstPageRequests === 1) {
      return route.fulfill({ json: homePayload([100, ...Array.from({ length: 20 }, (_, index) => index + 1)], { page: 1, nextCursor: 20 }) })
    }
    if (state.viewerId !== 12) {
      return route.fulfill({ json: homePayload(Array.from({ length: 20 }, (_, index) => 501 + index), { page: 1, nextCursor: 20, viewerId: state.viewerId }) })
    }
    return route.fulfill({ json: homePayload(Array.from({ length: 20 }, (_, index) => index + 1), {
      page: 1, nextCursor: 20, titleUpdates: state.firstPageRequests > 0 ? { 1: '首页已更新' } : {},
      tracePrefix: state.firstPageRequests > 0 ? 'return' : 'trace',
    }) })
  })
  await page.goto(`${origin}/assets/test/fixtures/browser/home-return.html`)
  await page.waitForFunction(() => document.documentElement.dataset.ready === 'true')
  await page.locator('[data-feed-id="20"]').waitFor()
  return { page, state }
}

async function loadThreePages(page) {
  for (const lastID of [40, 60]) {
    await page.getByRole('button', { name: '加载更多', exact: true }).click()
    await page.locator(`[data-feed-id="${lastID}"]`).waitFor()
  }
  return page.locator('[data-feed-id]').evaluateAll((rows) => rows.map((row) => Number(row.dataset.feedId)))
}

test('Web browser back from topic detail preserves three loaded feed pages and scroll position', async () => {
  const { page, state } = await openFeed()
  try {
    const loadedIDs = await loadThreePages(page)
    await page.locator('[data-feed-id="51"]').evaluate((row) => window.scrollBy(0, row.getBoundingClientRect().top - 100))
    const before = await page.locator('[data-feed-id="51"]').evaluate((row) => ({ top: row.getBoundingClientRect().top, scrollY: window.scrollY }))
    const originalTrace = await page.locator('[data-feed-id="1"]').getAttribute('data-feed-trace')

    await page.locator('[data-feed-id="51"] a[href="/p/post/51"]').first().click()
    await page.getByTestId('detail').waitFor()
    await page.goBack()
    await page.locator('[data-feed-id="60"]').waitFor()
    await page.locator('[data-feed-view="table"] a[href="/p/post/1"]', { hasText: '首页已更新' }).waitFor()

    assert.deepEqual(await page.locator('[data-feed-id]').evaluateAll((rows) => rows.map((row) => Number(row.dataset.feedId))), loadedIDs)
    assert.equal(await page.locator('[data-feed-id="1"]').getAttribute('data-feed-trace'), originalTrace)
    const after = await page.locator('[data-feed-id="51"]').evaluate((row) => ({ top: row.getBoundingClientRect().top, scrollY: window.scrollY }))
    assert.ok(Math.abs(after.top - before.top) < 3, `topic anchor moved from ${before.top}px to ${after.top}px`)
    assert.ok(Math.abs(after.scrollY - before.scrollY) < 3, `scrollY moved from ${before.scrollY}px to ${after.scrollY}px`)
    assert.equal(state.firstPageRequests, 1)
    await page.getByRole('button', { name: '加载更多', exact: true }).click()
    await page.locator('[data-feed-id="80"]').waitFor()
    assert.deepEqual(state.cursors, ['next-20', 'next-40', 'next-60'])
  } finally { await page.close() }
})

test('SSE hello resync updates the first page without dropping loaded rows or the cursor', async () => {
  const { page, state } = await openFeed({ resync: true })
  try {
    await loadThreePages(page)
    await page.locator('[data-feed-id="51"]').evaluate((row) => window.scrollBy(0, row.getBoundingClientRect().top - 100))
    const beforeTop = await page.locator('[data-feed-id="51"]').evaluate((row) => row.getBoundingClientRect().top)

    await page.evaluate(() => window.dispatchFeedHello())
    await page.locator('[data-feed-id="100"]').waitFor()
    await page.locator('[data-feed-id="60"]').waitFor()
    assert.equal(await page.locator('[data-feed-id]').count(), 61)
    assert.ok(Math.abs((await page.locator('[data-feed-id="51"]').evaluate((row) => row.getBoundingClientRect().top)) - beforeTop) < 3)

    await page.getByRole('button', { name: '加载更多', exact: true }).click()
    await page.locator('[data-feed-id="80"]').waitFor()
    assert.deepEqual(state.cursors, ['next-20', 'next-40', 'next-60'])
  } finally { await page.close() }
})

test('category detail return keeps its cached pages and scroll position', async () => {
  const { page, state } = await openFeed()
  try {
    await page.locator('[data-feed-view="table"] a[href="/c/7"]').first().click()
    await page.waitForURL('**/c/7')
    await page.locator('[data-feed-id="220"]').waitFor()
    for (const lastID of [240, 260]) {
      await page.getByRole('button', { name: '加载更多', exact: true }).click()
      await page.locator(`[data-feed-id="${lastID}"]`).waitFor()
    }
    const loadedIDs = await page.locator('[data-feed-id]').evaluateAll((rows) => rows.map((row) => Number(row.dataset.feedId)))
    await page.locator('[data-feed-id="251"]').evaluate((row) => window.scrollBy(0, row.getBoundingClientRect().top - 100))
    const before = await page.locator('[data-feed-id="251"]').evaluate((row) => ({ top: row.getBoundingClientRect().top, scrollY: window.scrollY }))

    await page.locator('[data-feed-id="251"] a[href="/p/post/251"]').first().click()
    await page.getByTestId('detail').waitFor()
    await page.goBack()
    await page.locator('[data-feed-id="260"]').waitFor()

    assert.deepEqual(await page.locator('[data-feed-id]').evaluateAll((rows) => rows.map((row) => Number(row.dataset.feedId))), loadedIDs)
    const after = await page.locator('[data-feed-id="251"]').evaluate((row) => ({ top: row.getBoundingClientRect().top, scrollY: window.scrollY }))
    assert.ok(Math.abs(after.top - before.top) < 3)
    assert.ok(Math.abs(after.scrollY - before.scrollY) < 3)
    assert.equal(state.categoryRequests, 2, 'category payload was requested again on browser back')
  } finally { await page.close() }
})

test('changing home sort replaces the old feed instead of carrying its loaded rows', async () => {
  const { page } = await openFeed()
  try {
    await loadThreePages(page)
    await page.locator('a[href="/?sort=hot"]').first().click()
    await page.waitForURL('**/?sort=hot')
    await page.locator('[data-feed-id="120"]').waitFor()
    assert.equal(await page.locator('[data-feed-id]').count(), 20)
    assert.equal(await page.locator('[data-feed-id="60"]').count(), 0)
  } finally { await page.close() }
})

test('return after account change discards the previous account feed', async () => {
  const { page, state } = await openFeed()
  try {
    await loadThreePages(page)
    await page.locator('[data-feed-id="51"] a[href="/p/post/51"]').first().click()
    await page.getByTestId('detail').waitFor()
    state.viewerId = 13
    await page.goBack()
    await page.locator('[data-feed-id="520"]').waitFor()
    assert.equal(await page.locator('[data-feed-id]').count(), 20)
    assert.equal(await page.locator('[data-feed-id="60"]').count(), 0)
  } finally { await page.close() }
})

test('SSE after an account change replaces rows and resets the old cursor', async () => {
  const { page, state } = await openFeed()
  try {
    await loadThreePages(page)
    state.viewerId = 13
    await page.evaluate(() => window.dispatchFeedHello())
    await page.locator('[data-feed-id="520"]').waitFor()
    assert.equal(await page.locator('[data-feed-id]').count(), 20)
    assert.equal(await page.locator('[data-feed-id="60"]').count(), 0)
    await page.getByRole('button', { name: '加载更多', exact: true }).click()
    await page.waitForFunction(() => document.querySelectorAll('[data-feed-id]').length > 20)
    assert.equal(state.cursors.at(-1), 'next-20')
  } finally { await page.close() }
})

test('continuation after account change reloads the new account first page', async () => {
  const { page, state } = await openFeed()
  try {
    await loadThreePages(page)
    state.viewerId = 13
    await page.getByRole('button', { name: '加载更多', exact: true }).click()
    await page.locator('[data-feed-id="520"]').waitFor()
    assert.equal(await page.locator('[data-feed-id]').count(), 20)
    assert.equal(await page.locator('[data-feed-id="60"]').count(), 0)
  } finally { await page.close() }
})

test('an account change observed by a background probe cannot merge old rows on SSE', async () => {
  const { page, state } = await openFeed()
  try {
    await loadThreePages(page)
    state.viewerId = 13
    await page.evaluate(async () => {
      const { fetchPage } = await import('/assets/src/runtime/router.ts')
      await fetchPage(new URL('/?sort=latest', window.location.origin), true)
    })
    await page.evaluate(() => window.dispatchFeedHello())
    await page.locator('[data-feed-id="520"]').waitFor()
    assert.equal(await page.locator('[data-feed-id]').count(), 20)
    assert.equal(await page.locator('[data-feed-id="60"]').count(), 0)
  } finally { await page.close() }
})

for (const trigger of ['back', 'SSE']) {
  test(`a removed cached topic disappears on ${trigger} without discarding later pages`, async () => {
    const { page, state } = await openFeed()
    try {
      await loadThreePages(page)
      if (trigger === 'back') {
        await page.locator('[data-feed-id="51"] a[href="/p/post/51"]').first().click()
        await page.getByTestId('detail').waitFor()
      }
      state.removed = [51]
      if (trigger === 'back') await page.goBack()
      else await page.evaluate(() => window.dispatchFeedHello())
      await page.locator('[data-feed-id="60"]').waitFor()
      await page.waitForFunction(() => !document.querySelector('[data-feed-id="51"]'))
      assert.equal(await page.locator('[data-feed-id="60"]').count(), 1)
      assert.equal(await page.locator('[data-feed-id]').count(), 59)
    } finally { await page.close() }
  })
}
