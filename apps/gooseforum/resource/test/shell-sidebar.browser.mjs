import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

// 前台外壳侧栏折叠（issue #798）：折叠的实质是网格轨道与裁切，
// happy-dom 测不到真实宽度，因此这里渲染生产组件与样式测量布局，
// 覆盖「桌面限定可见」「记忆状态」「rail 三列」「键盘不可达」「动效与 reduced-motion」。
let server, browser, origin
const collapsedStorageKey = 'goose:shell-sidebar-collapsed'

before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })

const fixtureUrl = (query = '') => `${origin}/assets/test/fixtures/browser/shell-sidebar.html${query}`

async function openShell(page, query = '') {
  await page.goto(fixtureUrl(query))
  await page.waitForFunction(() => document.documentElement.dataset.ready === 'true')
  return page.locator('button[aria-controls="goose-shell-sidebar"]')
}

const boxWidth = (page, selector) => page.locator(selector).evaluate((el) => el.getBoundingClientRect().width)
const sidebarWidth = (page) => boxWidth(page, '#goose-shell-sidebar')
const feedWidth = (page) => boxWidth(page, '[data-testid="feed"]')

// 旋转态由 transform 矩阵读出：rotate(180deg) 的 a = cos180 = -1，未旋转为 none。
// 用矩阵而不是类名断言，顺带证明组件层样式真的加载并生效。
const iconRotation = (page) => page.locator('.gf-shell-sidebar-toggle-icon').evaluate((el) => {
  const value = getComputedStyle(el).transform
  if (value === 'none') return null
  const match = value.match(/matrix\(([-\d.]+)/)
  return match ? Number(match[1]) : null
})

test('桌面端：折叠收起侧栏、内容列占满，刷新后保持', async () => {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
  try {
    const toggle = await openShell(page)
    const expandedSidebar = await sidebarWidth(page)
    const expandedFeed = await feedWidth(page)
    assert.ok(expandedSidebar > 200, `展开时侧栏应占宽（实测 ${expandedSidebar}px）`)
    assert.equal(await toggle.getAttribute('aria-expanded'), 'true')
    assert.equal(await iconRotation(page), null, '展开时图标不应带旋转')

    await toggle.click()
    await page.waitForFunction(() => document.querySelector('#goose-shell-sidebar').getBoundingClientRect().width < 1)
    // 图标旋转与轨道同为 200ms 但缓动不同，宽度归零时旋转可能仍在中途，
    // 先等它落定再断言，避免读到 -0.99997 这类过程值。
    await page.waitForFunction(() => {
      const match = getComputedStyle(document.querySelector('.gf-shell-sidebar-toggle-icon')).transform.match(/matrix\(([-\d.]+)/)
      return match ? Number(match[1]) <= -0.999 : false
    }, null, { timeout: 5000 })
    const rotation = await iconRotation(page)
    assert.ok(Math.abs(rotation + 1) < 0.01, `收起后图标应旋过半圈（matrix a≈cos180=-1，实测 ${rotation}）`)
    const collapsedFeed = await feedWidth(page)
    assert.ok(
      collapsedFeed > expandedFeed + expandedSidebar - 40,
      `折叠后内容列应获得侧栏让出的宽度（展开 ${expandedFeed}px → 折叠 ${collapsedFeed}px）`,
    )
    assert.equal(await page.locator('#goose-shell-sidebar').getAttribute('aria-hidden'), 'true')
    assert.equal(await toggle.getAttribute('aria-expanded'), 'false')
    assert.equal(await page.evaluate((key) => localStorage.getItem(key), collapsedStorageKey), '1')
    assert.ok(
      await page.locator('#goose-shell-sidebar nav').evaluate((el) => el.getBoundingClientRect().width) > 200,
      '折叠时侧栏内容应保持展开宽度被裁切，而不是逐帧重排换行',
    )

    await page.reload()
    await page.waitForFunction(() => document.documentElement.dataset.ready === 'true')
    await page.waitForFunction(() => document.querySelector('#goose-shell-sidebar').getBoundingClientRect().width < 1)
    const reloadedToggle = page.locator('button[aria-controls="goose-shell-sidebar"]')
    assert.equal(await reloadedToggle.getAttribute('aria-expanded'), 'false', '刷新后应保持收起')

    await reloadedToggle.click()
    await page.waitForFunction((width) => document.querySelector('#goose-shell-sidebar').getBoundingClientRect().width > width - 1, expandedSidebar)
    assert.equal(await page.evaluate((key) => localStorage.getItem(key), collapsedStorageKey), '0')
  } finally { await page.close() }
})

test('收起后侧栏对键盘不可达（inert 生效）', async () => {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
  try {
    const toggle = await openShell(page)
    await toggle.click()
    await page.waitForFunction(() => document.querySelector('#goose-shell-sidebar').matches('[inert]'))
    // 起点固定在侧栏之外：inert 子树不可聚焦，activeElement 必须留在原地
    await toggle.focus()
    await page.locator('#goose-shell-sidebar a').first().focus()
    assert.equal(
      await page.evaluate(() => document.activeElement?.closest('#goose-shell-sidebar') === null),
      true,
      '收起后侧栏内链接不应获得焦点',
    )
  } finally { await page.close() }
})

test('带右侧 rail 的三列布局：折叠只收侧栏列', async () => {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
  try {
    const toggle = await openShell(page, '?rail=1')
    const railBefore = await boxWidth(page, '[data-testid="rail"]')
    assert.ok(railBefore > 260, `xl rail 应为 280px（实测 ${railBefore}px）`)

    await toggle.click()
    await page.waitForFunction(() => document.querySelector('#goose-shell-sidebar').getBoundingClientRect().width < 1)
    const railAfter = await boxWidth(page, '[data-testid="rail"]')
    assert.ok(Math.abs(railAfter - railBefore) < 1, '折叠不得改变 rail 列宽')
    assert.ok(await feedWidth(page) > 600, '内容列仍应占满中间列')
  } finally { await page.close() }
})

test('窄屏（<lg）：不渲染折叠开关，抽屉与内容列不受影响', async () => {
  const page = await browser.newPage({ viewport: { width: 390, height: 844 } })
  try {
    // 已收起偏好在窄屏同样继承：抽屉模式下侧栏列本就隐藏，内容列不得被压缩
    await page.addInitScript((key) => localStorage.setItem(key, '1'), collapsedStorageKey)
    const toggle = await openShell(page)
    assert.equal(await toggle.isVisible(), false)
    assert.equal(await page.locator('#goose-shell-sidebar').isVisible(), false)
    assert.ok(await feedWidth(page) > 300, '窄屏内容列应占满可用宽度')
  } finally { await page.close() }
})

test('过渡动效：常规下轨道与图标旋转过渡，prefers-reduced-motion 下取消', async () => {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
  try {
    await openShell(page)
    const main = page.locator('.gf-shell-main')
    const aside = page.locator('#goose-shell-sidebar')
    const icon = page.locator('.gf-shell-sidebar-toggle-icon')
    assert.match(await main.evaluate((el) => getComputedStyle(el).transitionProperty), /grid-template-columns/)
    assert.equal(await main.evaluate((el) => getComputedStyle(el).transitionDuration), '0.2s')
    assert.match(await aside.evaluate((el) => getComputedStyle(el).transitionProperty), /opacity/)
    assert.match(await icon.evaluate((el) => getComputedStyle(el).transitionProperty), /transform/)
    assert.equal(await icon.evaluate((el) => getComputedStyle(el).transitionDuration), '0.2s')

    await page.emulateMedia({ reducedMotion: 'reduce' })
    assert.equal(await main.evaluate((el) => getComputedStyle(el).transitionDuration), '0s')
    assert.equal(await aside.evaluate((el) => getComputedStyle(el).transitionDuration), '0s')
    assert.equal(await icon.evaluate((el) => getComputedStyle(el).transitionDuration), '0s')
  } finally { await page.close() }
})
