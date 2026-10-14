// issue #1063 回归：首页信息流内容类型徽章必须在长标题、窄内容列下保持单行完整尺寸，
// 不允许被标题挤压缩窄（文字折行竖排溢出胶囊）。happy-dom 没有布局引擎，只有真浏览器能量出几何。
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

// de/en/ja 标签更长（Artikel/Question/モーメント），一并验证 nowrap 后不会撑破行。
const LABELS = {
  zh: { 1: '提问', 2: '瞬间', 3: '文章' },
  en: { 1: 'Question', 2: 'Moment', 3: 'Article' },
  de: { 1: 'Frage', 2: 'Moment', 3: 'Artikel' },
  ja: { 1: '質問', 2: 'モーメント', 3: '記事' },
}
const TOLERANCE = 0.5

// 量出胶囊（rounded-full 类型徽章）与其中文字行的几何；文字行数 > 1 即报告中的「折行竖排」。
async function measureBadge(page, caseName, label) {
  return page.evaluate(({ caseName, label }) => {
    const scope = document.querySelector(`[data-case="${caseName}"]`)
    if (!scope) return { error: `missing case ${caseName}` }
    const pill = [...scope.querySelectorAll('span.rounded-full')].find(el => el.textContent.trim() === label)
    if (!pill) return { error: `missing badge ${label} in ${caseName}` }
    const textEl = pill.querySelector('span') ?? pill
    const range = document.createRange()
    range.selectNodeContents(textEl)
    const lines = [...range.getClientRects()].map(r => ({ width: r.width, height: r.height, top: r.top, bottom: r.bottom }))
    const box = pill.getBoundingClientRect()
    return { box: { width: box.width, height: box.height, top: box.top, bottom: box.bottom }, lines }
  }, { caseName, label })
}

function assertInsidePill(measured, context) {
  assert.equal(measured.error, undefined, `${context}: ${measured.error}`)
  const { box, lines } = measured
  assert.equal(lines.length, 1, `${context}: 徽章文字必须单行显示，实际 ${lines.length} 行（折行竖排）`)
  const line = lines[0]
  assert.ok(line.width <= box.width + TOLERANCE, `${context}: 徽章文字宽 ${line.width} 溢出胶囊宽 ${box.width}`)
  assert.ok(line.height <= box.height + TOLERANCE, `${context}: 徽章文字高 ${line.height} 溢出胶囊高 ${box.height}`)
  assert.ok(line.top >= box.top - TOLERANCE && line.bottom <= box.bottom + TOLERANCE, `${context}: 徽章文字纵向溢出胶囊`)
}

async function badgeWidth(page, caseName, label) {
  const measured = await measureBadge(page, caseName, label)
  assert.equal(measured.error, undefined, `${caseName}: ${measured.error}`)
  return measured.box.width
}

// 徽章不收缩后，被挤走的空间必须由标题截断吸收：行本身不得横向溢出，标题要真的出现省略号。
async function assertTitleAbsorbsSqueeze(page, caseName, context) {
  const row = await page.evaluate((caseName) => {
    const scope = document.querySelector(`[data-case="${caseName}"]`)
    const article = scope.querySelector('article')
    const title = article.querySelector('a.truncate')
    return {
      overflowX: article.scrollWidth - article.clientWidth,
      titleOverflow: title ? title.scrollWidth - title.clientWidth : null,
    }
  }, caseName)
  assert.ok(row.overflowX <= 1, `${context}: 行横向溢出 ${row.overflowX}px（徽章未让位给标题截断）`)
  assert.ok(row.titleOverflow !== null && row.titleOverflow > 1, `${context}: 长标题应被截断吸收挤压，实际未截断`)
}

// 徽章既不能被挤扁（宽度与无压力基准一致），也不能把标题挤出行外。
async function assertBadgeStaysIntact(page, { label, naturalCase, nestedCase, context }) {
  assertInsidePill(await measureBadge(page, nestedCase, label), context)
  const natural = await badgeWidth(page, naturalCase, label)
  const squeezed = await badgeWidth(page, nestedCase, label)
  assert.ok(
    Math.abs(squeezed - natural) <= TOLERANCE,
    `${context}: 长标题把徽章从 ${natural}px 挤到 ${squeezed}px`,
  )
  await assertTitleAbsorbsSqueeze(page, nestedCase, context)
}

for (const locale of ['zh', 'en', 'de', 'ja']) {
  test(`宽屏（1512px）列表视图 · ${locale}：长标题窄内容列下内容类型徽章保持完整尺寸不收缩`, async () => {
    const page = await browser.newPage({ viewport: { width: 1512, height: 900 } })
    try {
      await page.goto(`${origin}/assets/test/fixtures/browser/topic-badges.html?locale=${locale}`)
      await page.locator('[data-case="narrow-lg-3"]').waitFor()
      for (const type of [1, 2, 3]) {
        await assertBadgeStaysIntact(page, {
          label: LABELS[locale][type],
          naturalCase: `natural-${type}`,
          nestedCase: `narrow-lg-${type}`,
          context: `宽屏列表 ${locale} 类型${type}`,
        })
      }
    } finally { await page.close() }
  })
}

for (const locale of ['zh', 'de']) {
  for (const width of [375, 320]) {
    test(`窄屏（${width}px）列表视图 · ${locale}：内容类型徽章单行完整，不被标题挤压`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 900 } })
      try {
        await page.goto(`${origin}/assets/test/fixtures/browser/topic-badges.html?locale=${locale}`)
        await page.locator('[data-case="narrow-sm-3"]').waitFor()
        for (const type of [1, 2, 3]) {
          await assertBadgeStaysIntact(page, {
            label: LABELS[locale][type],
            naturalCase: `natural-sm-${type}`,
            nestedCase: `narrow-sm-${type}`,
            context: `窄屏列表 ${width}px ${locale} 类型${type}`,
          })
        }
      } finally { await page.close() }
    })
  }
}

test('卡片视图（320px）：内容类型徽章同样单行完整', async () => {
  const page = await browser.newPage({ viewport: { width: 320, height: 900 } })
  try {
    await page.goto(`${origin}/assets/test/fixtures/browser/topic-badges.html`)
    await page.locator('[data-case="card"]').waitFor()
    assertInsidePill(await measureBadge(page, 'card', LABELS.zh[3]), '卡片视图 类型3')
  } finally { await page.close() }
})
