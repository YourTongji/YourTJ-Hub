import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server, browser, origin

before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })

// 量公告面板的真实几何：面板高度、面板下方第一个兄弟元素的顶边、
// 各公告层的左上角（同格叠放判据）与当前可见层。
// 面板高度稳定 = 下方话题列表不随轮播上下移动（issue #1102）。
function readGeometry(page) {
  return page.evaluate(() => {
    const panel = document.querySelector('.gf-announcement-panel')
    const below = panel.nextElementSibling
    const layers = [...panel.querySelectorAll('[data-testid="announcement-item"]')]
    const visible = layers.filter((el) => !el.classList.contains('invisible'))
    return {
      panelHeight: panel.getBoundingClientRect().height,
      belowTop: below.getBoundingClientRect().top,
      layerCount: layers.length,
      visibleCount: visible.length,
      layerRects: layers.map((el) => {
        const rect = el.getBoundingClientRect()
        return { top: rect.top, left: rect.left }
      }),
      visibleText: visible.map((el) => el.textContent.replace(/\s+/g, '')).join(''),
    }
  })
}

test('公告轮播与圆点切换时面板高度稳定，下方内容不移动（issue #1102）', async () => {
  const page = await browser.newPage({ viewport: { width: 880, height: 900 } })
  try {
    await page.emulateMedia({ reducedMotion: 'no-preference' })
    await page.route('**/api/forum/events', (route) => route.fulfill({ status: 200, contentType: 'text/event-stream', body: '' }))
    // 假时钟：同一套定时器逻辑下把 6 秒自动轮播推进到触发点，不真等。
    await page.clock.install()
    await page.goto(`${origin}/assets/test/fixtures/browser/home-announcement-rotation.html`)
    await page.locator('.gf-announcement-panel').waitFor()

    const initial = await readGeometry(page)
    assert.equal(initial.layerCount, 3, '所有公告条目都应参与高度预留（同格叠放）')
    assert.equal(initial.visibleCount, 1, '同一时刻只应展示一条公告')
    assert.ok(initial.visibleText.includes('一行短公告'), `初始应展示第一条公告，实际：${initial.visibleText}`)

    // 同格叠放：所有条目层共享同一网格单元，行高才由最高条目决定
    const [firstLayer, ...otherLayers] = initial.layerRects
    for (const rect of otherLayers) {
      assert.equal(rect.top, firstLayer.top, '所有条目层必须叠放在同一网格单元（top）')
      assert.equal(rect.left, firstLayer.left, '所有条目层必须叠放在同一网格单元（left）')
    }

    // 非激活条目必须排除在无障碍树与 Tab 焦点序之外（对齐折叠态 v-if 先例）
    assert.equal(await page.getByRole('link', { name: '说明链接' }).count(), 0, '隐藏公告内的链接不得进入无障碍树')
    assert.equal(
      await page.evaluate(() => {
        const link = document.querySelector('[data-testid="announcement-item"][inert] a')
        link.focus()
        return document.activeElement === link
      }),
      false,
      '隐藏公告内的链接不得可获得焦点',
    )

    // 自动轮播（6000ms）切到多行的第二条
    await page.clock.runFor(6200)
    const afterAuto = await readGeometry(page)
    assert.equal(afterAuto.visibleCount, 1)
    assert.ok(afterAuto.visibleText.includes('第二条公告的第一段说明文字。'), `自动轮播后应展示第二条公告，实际：${afterAuto.visibleText}`)
    assert.equal(afterAuto.panelHeight, initial.panelHeight, '自动轮播后面板高度必须不变')
    assert.equal(afterAuto.belowTop, initial.belowTop, '自动轮播后下方内容不得移动')

    // 手动圆点切换：多行 → 两行、两行 → 一行，高度与下方位置同样不变
    await page.getByRole('tab').nth(2).click()
    const afterManual = await readGeometry(page)
    assert.ok(afterManual.visibleText.includes('中等公告第一行'), `圆点切换后应展示第三条公告，实际：${afterManual.visibleText}`)
    assert.equal(afterManual.panelHeight, initial.panelHeight, '圆点切换后面板高度必须不变')
    assert.equal(afterManual.belowTop, initial.belowTop, '圆点切换后下方内容不得移动')

    await page.getByRole('tab').nth(1).click()
    const afterBack = await readGeometry(page)
    assert.ok(afterBack.visibleText.includes('第二条公告的第一段说明文字。'))
    assert.equal(afterBack.panelHeight, initial.panelHeight)
    assert.equal(afterBack.belowTop, initial.belowTop)

    // 该条目此时可见：链接回到无障碍树
    assert.equal(await page.getByRole('link', { name: '说明链接' }).count(), 1)
  } finally { await page.close() }
})

// reduced-motion 下不启动自动轮播，但圆点手动切换仍会换条目；
// 固定高度方案不含动效，这批用户同样不受整页跳动影响（issue #1102）。
test('prefers-reduced-motion 下圆点切换同样不改变面板高度（issue #1102）', async () => {
  const page = await browser.newPage({ viewport: { width: 880, height: 900 }, reducedMotion: 'reduce' })
  try {
    await page.route('**/api/forum/events', (route) => route.fulfill({ status: 200, contentType: 'text/event-stream', body: '' }))
    await page.goto(`${origin}/assets/test/fixtures/browser/home-announcement-rotation.html`)
    await page.locator('.gf-announcement-panel').waitFor()

    const initial = await readGeometry(page)
    assert.equal(initial.visibleCount, 1)
    assert.ok(initial.visibleText.includes('一行短公告'), `初始应展示第一条公告，实际：${initial.visibleText}`)

    await page.getByRole('tab').nth(1).click()
    const after = await readGeometry(page)
    assert.ok(after.visibleText.includes('第二条公告的第一段说明文字。'), `圆点切换后应展示第二条公告，实际：${after.visibleText}`)
    assert.equal(after.panelHeight, initial.panelHeight, 'reduced-motion 下圆点切换面板高度必须不变')
    assert.equal(after.belowTop, initial.belowTop, 'reduced-motion 下下方内容不得移动')
  } finally { await page.close() }
})
