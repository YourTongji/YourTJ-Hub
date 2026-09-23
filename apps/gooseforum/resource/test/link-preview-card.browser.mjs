import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

/**
 * LinkPreviewCard 的真实布局回归。
 *
 * 这里断言的是几何，而不是 class 名：卡片曾经因为 `block` 与 `line-clamp-*`
 * 同时写 display 互相覆盖，描述完整展开到 26 行、卡片高 692px，而任何
 * happy-dom 单测都测不出来（它不做布局）。所以本文件只跑真实 Chromium，
 * 在多个视口下量行数、封面比例、贴边规则与卡片高度预算。
 */

let server, browser, origin

before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})

after(async () => { await browser?.close(); await server?.close() })

/** 16:9 横版封面：带可见网格，方便在截图里看出 object-cover 的裁切方向。 */
const wideCover = `<svg xmlns="http://www.w3.org/2000/svg" width="1120" height="630"><rect width="1120" height="630" fill="#1f6feb"/><g fill="none" stroke="#ffffff" stroke-opacity="0.35" stroke-width="6"><path d="M0 210h1120M0 420h1120M373 0v630M746 0v630"/></g><circle cx="560" cy="315" r="150" fill="#ffffff" fill-opacity="0.22"/></svg>`
/** 1:3 竖版封面：最容易把卡片撑高的输入。 */
const tallCover = `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="1200"><rect width="400" height="1200" fill="#b54708"/><g fill="none" stroke="#ffffff" stroke-opacity="0.3" stroke-width="6"><path d="M0 400h400M0 800h400"/></g></svg>`
const favicon = `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32"><rect width="32" height="32" rx="6" fill="#f472b6"/></svg>`

async function installAssetRoutes(page) {
  await page.route('**/cover-wide.png', route => route.fulfill({ contentType: 'image/svg+xml', body: wideCover }))
  await page.route('**/cover-tall.png', route => route.fulfill({ contentType: 'image/svg+xml', body: tallCover }))
  await page.route('**/icon.png', route => route.fulfill({ contentType: 'image/svg+xml', body: favicon }))
}

function measure() {
  const rect = (element) => {
    const box = element.getBoundingClientRect()
    return { x: box.x, y: box.y, width: box.width, height: box.height, right: box.right, bottom: box.bottom }
  }
  const lineHeight = (element) => (element ? Number.parseFloat(getComputedStyle(element).lineHeight) : 0)
  const cases = {}
  for (const id of ['long', 'dedup', 'internal', 'tall', 'campus', 'campus-named']) {
    const card = document.querySelector(`#case-${id} .gf-link-preview-card`)
    if (!card) { cases[id] = null; continue }
    const title = card.querySelector('.gf-link-preview-card__title')
    const descWrap = card.querySelector('.gf-link-preview-card__desc')
    const desc = card.querySelector('.gf-link-preview-card__desc-text')
    const host = card.querySelector('.gf-link-preview-card__host')
    const cover = card.querySelector('.gf-link-preview-card__cover')
    const coverImage = card.querySelector('.gf-link-preview-card__cover-image')
    const body = card.querySelector('.gf-link-preview-card__body')
    cases[id] = {
      card: rect(card),
      body: body ? rect(body) : null,
      source: card.querySelector('.gf-link-preview-card__source')
        ? rect(card.querySelector('.gf-link-preview-card__source'))
        : null,
      title: title ? { ...rect(title), lineHeight: lineHeight(title), text: title.textContent.trim() } : null,
      descWrap: descWrap ? { ...rect(descWrap), display: getComputedStyle(descWrap).display } : null,
      desc: desc ? { ...rect(desc), lineHeight: lineHeight(desc), text: desc.textContent.trim() } : null,
      host: host ? { ...rect(host), text: host.textContent.trim() } : null,
      cover: cover
        ? {
            ...rect(cover),
            objectFit: coverImage ? getComputedStyle(coverImage).objectFit : null,
            naturalWidth: coverImage?.naturalWidth ?? 0,
            naturalHeight: coverImage?.naturalHeight ?? 0,
            imageBorderTopWidth: coverImage ? getComputedStyle(coverImage).borderTopWidth : null,
            display: getComputedStyle(cover).display,
          }
        : null,
      labels: [...card.querySelectorAll('.gf-link-preview-card__source > span, .gf-link-preview-card__host')].map(element => element.textContent.trim()),
    }
  }
  return { viewport: innerWidth, scrollWidth: document.documentElement.scrollWidth, cases }
}

const widths = [320, 375, 480, 640, 934, 1440]

for (const theme of ['dark', 'light']) {
  for (const width of widths) {
    test(`link preview card geometry fits ${theme} ${width}px`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 900 } })
      try {
        await installAssetRoutes(page)
        await page.goto(`${origin}/assets/test/fixtures/browser/link-preview-card.html?theme=${theme}`)
        await page.waitForFunction(() => [...document.querySelectorAll('img')].every(image => image.complete))
        const state = await page.evaluate(measure)

        // 1. 卡片永远不能把正文撑出横向滚动。
        assert.ok(state.scrollWidth <= width, `page overflows horizontally: ${state.scrollWidth} > ${width}`)

        const long = state.cases.long
        assert.ok(long, 'long case must render a card')

        const coverIsRail = width >= 640
        const descriptionLines = width < 480 ? 0 : (coverIsRail ? 2 : 1)
        // 高度预算由行高推导，而不是拍一个魔法常数：上下边框 2px + 上下内边距
        // + 来源行 + 标题 2 行 + 描述（按档位）+ 目标行，再加 12px 亚像素余量。
        // 修复前实际值是 256px（桌面）/ 692px（375px），这个预算足以拦住回退。
        const rows = (long.source?.height ?? 16) + 4 + 2 * long.title.lineHeight
          + (descriptionLines === 0 ? 0 : 12 + descriptionLines * long.desc.lineHeight + 12)
          + (long.host?.height ?? 16)
        const cardBudget = rows + (coverIsRail ? 32 : 28) + 2 + 12

        console.log(`      ${theme} ${width}px  card=${long.card.height.toFixed(1)} budget=${cardBudget.toFixed(1)} source=${long.source.height.toFixed(0)} titleCol=${long.title.width.toFixed(0)} desc=${long.desc.height.toFixed(0)} cover=${long.cover?.width.toFixed(0)}x${long.cover?.height.toFixed(0)}`)

        // 2. 来源行必须紧：prose.css 的 `.gf-prose img` 会给 favicon 加外边距与
        //    边框，把 16px 的行顶到 28px，所以这里钉住行高上限。
        assert.ok(long.source.height <= 20, `source row is bloated: ${long.source.height}px, prose img chrome leaked in`)

        // 3. 标题与描述的截断必须真的生效（-webkit-line-clamp 只在 -webkit-box 上工作）。
        assert.ok(
          long.title.height <= 2 * long.title.lineHeight + 1,
          `title is not clamped: ${long.title.height}px > 2x${long.title.lineHeight}px`,
        )
        if (descriptionLines === 0) {
          assert.equal(long.descWrap.display, 'none', 'description must be hidden below 480px')
          assert.equal(long.desc.height, 0, 'hidden description must not occupy layout')
        } else {
          assert.equal(long.descWrap.display, 'block', 'description wrapper must be visible from 480px up')
          assert.ok(
            long.desc.height <= descriptionLines * long.desc.lineHeight + 1,
            `description is not clamped to ${descriptionLines} lines: ${long.desc.height}px`,
          )
        }

        // 4. 卡片高度预算（含竖图输入）。竖图曾经把卡片从 164px 撑到 362px。
        assert.ok(long.card.height <= cardBudget, `long card height ${long.card.height}px exceeds ${cardBudget}px`)
        assert.ok(state.cases.tall.card.height <= cardBudget, `tall card height ${state.cases.tall.card.height}px exceeds ${cardBudget}px`)

        // 5. 来源行与目标 host 重复时不再多打印一行。
        assert.deepEqual(state.cases.dedup.labels, ['example.com'], 'duplicate host row must be omitted')
        assert.equal(long.host.text, 'www.bilibili.com')

        // 6. 封面：小屏是内缩的方形缩略图（居中裁切），大屏按封面自身比例定高、
        //    垂直居中并完整显示。修复前大屏导轨钉死成 168×134.7，配 object-cover
        //    会把 2:1 的 OG 图横向裁掉 37.6%（用户报的「封面没显示全」）。
        assert.ok(long.cover, 'long case must render a cover')
        assert.equal(long.cover.imageBorderTopWidth, '0px', 'cover must not inherit the prose img border')
        assert.ok(long.cover.naturalWidth > 0 && long.cover.naturalHeight > 0, 'cover image must be decoded before measuring')
        const sourceAspect = long.cover.naturalWidth / long.cover.naturalHeight
        const clampedAspect = Math.min(2.4, Math.max(1.2, sourceAspect))
        const boxAspect = long.cover.width / long.cover.height
        if (coverIsRail) {
          // contain 从结构上不可能裁切；再钉住盒子的比例等于钳制后的源比例，
          // 保证它不是靠 letterbox 兜住而实际缩得很小。
          assert.equal(long.cover.objectFit, 'contain', 'desktop cover must never crop')
          assert.ok(
            Math.abs(boxAspect - clampedAspect) <= clampedAspect * 0.02,
            `desktop cover box must follow the source aspect (source=${sourceAspect.toFixed(3)} box=${boxAspect.toFixed(3)} want=${clampedAspect.toFixed(3)})`,
          )
          assert.ok(long.cover.width >= 120, `desktop cover too small: ${long.cover.width}px`)
          assert.ok(
            Math.abs((long.card.right - long.cover.right) - 16) <= 1,
            `desktop cover must be inset 16px from the card edge (gap=${(long.card.right - long.cover.right).toFixed(1)}px)`,
          )
          const coverCenter = long.cover.y + long.cover.height / 2
          const cardCenter = long.card.y + long.card.height / 2
          assert.ok(
            Math.abs(coverCenter - cardCenter) <= 2,
            `desktop cover must be vertically centered (cover=${coverCenter.toFixed(1)} card=${cardCenter.toFixed(1)})`,
          )
          assert.ok(
            long.cover.height <= long.card.height + 1,
            `desktop cover must not exceed the card (cover=${long.cover.height.toFixed(1)} card=${long.card.height.toFixed(1)})`,
          )
        } else {
          assert.equal(long.cover.objectFit, 'cover', 'mobile thumbnail keeps its square crop')
          assert.ok(Math.abs(long.cover.width - long.cover.height) <= 1, 'mobile thumbnail must be square')
          assert.ok(long.cover.width <= 80, `mobile thumbnail too large: ${long.cover.width}px`)
          assert.ok(long.card.right - long.cover.right >= 8, 'mobile thumbnail must be inset from the card edge')
          // 缩略图不能把标题压到不可读：320px 下正文列仍需 >= 190px。
          assert.ok(long.title.width >= 190, `text column too narrow: ${long.title.width}px`)
        }

        // 7. 竖版封面走比例下限（仅大屏）：源图 1:3，若原样贴边会缩成一条细缝，
        //    所以钳到 1.2（盒高 ≤140），再由 max-h-full 保证不超出卡片。
        //    小屏是固定 56px 方形缩略图，比例下限不参与。
        const tall = state.cases.tall
        assert.ok(tall.cover, 'tall case must render a cover')
        if (coverIsRail) {
          assert.equal(tall.cover.objectFit, 'contain', 'tall desktop cover must never crop')
          const tallBoxAspect = tall.cover.width / tall.cover.height
          assert.ok(
            tallBoxAspect >= 1.2 - 0.02,
            `tall cover must respect the 1.2 aspect floor (box=${tallBoxAspect.toFixed(3)})`,
          )
        } else {
          assert.equal(tall.cover.objectFit, 'cover', 'tall mobile thumbnail keeps its square crop')
          assert.ok(Math.abs(tall.cover.width - tall.cover.height) <= 1, 'tall mobile thumbnail must be square')
        }

        // 8. 校园网卡片：服务端只给 campus 标记（展示名来自部署配置），配置没给
        //    名字时 title / description 都为空——兜底文案必须由客户端 i18n 渲染，
        //    服务端不得留任何中文（夹具把语言固定成 zh）。
        const campus = state.cases.campus
        const campusNamed = state.cases['campus-named']
        assert.ok(campus && campusNamed, 'campus cases must render cards')
        assert.equal(campus.title.text, '校园网', 'nameless campus card must use the localized fallback title')
        assert.equal(campus.desc.text, '需校园网络访问', 'nameless campus card must use the localized fallback description')
        assert.equal(campus.host, null, 'campus card must collapse the duplicate host row')
        // 配置给了名字时以服务端值为准，客户端不得用兜底文案覆盖。
        assert.equal(campusNamed.title.text, '同济大学教学管理系统', 'configured campus name must win over the fallback')
        assert.equal(campusNamed.desc.text, '需校园网络访问', 'named campus card still gets the localized hint')
        // 校园网卡片没有封面/图标，右侧内边距必须回到无封面档。
        assert.equal(campus.cover, null, 'campus card must not reserve cover space')
        assert.equal(campus.body.width, campus.card.width - 2, 'body must use the full card width without a cover')

        if (process.env.LINK_PREVIEW_CARD_SCREENSHOT) {
          await page.screenshot({ path: `${process.env.LINK_PREVIEW_CARD_SCREENSHOT}-${theme}-${width}.png`, fullPage: true })
        }
      } finally {
        await page.close()
      }
    })
  }
}
