import assert from 'node:assert/strict'
import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

/**
 * 徽章图标的光学尺寸守卫：在个人主页头像的真实几何下，每个图标的墨迹
 * 都不能压到角标圆环上，所有图标的墨迹半径必须落在同一档（视觉等大），
 * 并且同一个图标在各头像尺寸下占角标的比例一致。
 *
 * 测量方式：页面里把 SVG 光栅化到 480×480（24 单位画布 → 20px/unit），
 * 扫描非透明像素得到墨迹最大半径（单位：viewBox 单位）；再读取真实 DOM 里
 * 角标圆与图标内容盒的像素尺寸，换算出圆内可用半径。
 *
 * 文件层不变量（第二个 test）：归一化缩放必须是以画布中心为基准的均匀缩放，
 * 且描边按 1/scale 补偿，使屏幕上的描边粗细与改前一致。
 */
let server, browser, page, origin

before(async () => {
  server = await createServer({
    root: fileURLToPath(new URL('../', import.meta.url)),
    server: { host: '127.0.0.1', port: 0, open: false },
    logLevel: 'error',
  })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
  page = await browser.newPage({ viewport: { width: 1280, height: 900 } })
  // 线上 /static/badges/ 由 Go 二进制提供；裸 Vite 只服务 /assets/，这里直接从磁盘回应
  await page.route((url) => url.pathname.startsWith('/static/badges/'), (route) => {
    const name = new URL(route.request().url()).pathname.slice('/static/badges/'.length)
    return route.fulfill({ path: fileURLToPath(new URL(`../static/badges/${name}`, import.meta.url)), contentType: 'image/svg+xml' })
  })
})

after(async () => {
  await browser?.close()
  await server?.close()
})

test('badge artwork stays inside the avatar chip and keeps one optical size', async () => {
  await page.goto(`${origin}/assets/test/fixtures/browser/avatar-badge-live.html`)
  await page.waitForFunction(() => document.querySelectorAll('[data-variant="after"]').length >= 60)

  const measured = await page.evaluate(async () => {
    const N = 480
    const unit = 24 / N
    async function inkRadius(src) {
      const img = new Image()
      img.src = src
      await img.decode()
      const canvas = document.createElement('canvas')
      canvas.width = N
      canvas.height = N
      const ctx = canvas.getContext('2d', { willReadFrequently: true })
      ctx.drawImage(img, 0, 0, N, N)
      const data = ctx.getImageData(0, 0, N, N).data
      let maxRadius = 0
      for (let y = 0; y < N; y++) {
        for (let x = 0; x < N; x++) {
          if (data[(y * N + x) * 4 + 3] > 24) {
            const r = Math.hypot((x + 0.5) * unit - 12, (y + 0.5) * unit - 12)
            if (r > maxRadius) maxRadius = r
          }
        }
      }
      return maxRadius
    }

    const cells = Array.from(document.querySelectorAll('[data-variant="after"]'))
    const seen = new Set()
    const rows = []
    for (const cell of cells) {
      const code = cell.dataset.badgeCode
      const site = cell.dataset.site
      if (!code || !site || seen.has(`${site}:${code}`)) continue
      seen.add(`${site}:${code}`)
      const chip = cell.querySelector('span.absolute')
      const icon = chip?.querySelector('img')
      if (!chip || !icon) continue
      const chipPx = chip.getBoundingClientRect().width
      const iconBoxPx = icon.getBoundingClientRect().width
      const avatarPx = cell.getBoundingClientRect().width
      const allowRunit = (chipPx / 2) / (iconBoxPx / 24)
      const maxR = await inkRadius(icon.getAttribute('src'))
      rows.push({ code, site, avatarPx: +avatarPx.toFixed(1), chipPx: +chipPx.toFixed(1), maxR: +maxR.toFixed(2), ratio: +(maxR / allowRunit).toFixed(3) })
    }
    return rows
  })

  // 角标图标区按比例缩放：同一个图标在任何头像尺寸下占角标的比例都一样，
  // 不会因为固定像素内边距在小头像上额外缩小。
  for (const code of new Set(measured.map((row) => row.code))) {
    const bySite = measured.filter((row) => row.code === code)
    assert.equal(bySite.length, 4, `${code}: 应在 profile / card / post / reply 四种尺寸下各测到一次`)
    const sizeSpread = Math.max(...bySite.map((row) => row.ratio)) - Math.min(...bySite.map((row) => row.ratio))
    assert.ok(
      sizeSpread <= 0.02,
      `${code}: 不同头像尺寸下占比不一致（${bySite.map((row) => `${row.site} ${Math.round(row.ratio * 100)}%`).join('、')}）`,
    )
  }

  const result = measured.filter((row) => row.site === 'profile')
  assert.equal(result.length, 15, '应测到全部 15 个系统徽章')
  for (const row of result) {
    assert.ok(row.avatarPx > 100, `${row.code}: 测量上下文应为个人主页尺寸（h-24/sm:h-28），实得 ${row.avatarPx}px`)
  }

  const crowded = result.filter((row) => row.ratio > 0.8)
  assert.deepEqual(
    crowded.map((row) => `${row.code}=${Math.round(row.ratio * 100)}%`),
    [],
    `徽章墨迹压到角标圆环（> 80% 可用半径）：${crowded.map((row) => `${row.code} ${row.maxR}单位`).join('、')}`,
  )

  const ratios = result.map((row) => row.ratio)
  const spread = Math.max(...ratios) - Math.min(...ratios)
  assert.ok(
    spread <= 0.05,
    `徽章图标视觉尺寸不一致：最大的 ${Math.round(Math.max(...ratios) * 100)}% 与最小的 ${Math.round(Math.min(...ratios) * 100)}% 相差 ${Math.round(spread * 100)} 个百分点`,
  )
})

test('badge artwork keeps a centred uniform scale and the compensated stroke width', () => {
  const dir = fileURLToPath(new URL('../static/badges/', import.meta.url))
  const files = readdirSync(dir).filter((name) => name.endsWith('.svg'))
  assert.ok(files.length >= 15, `系统徽章 SVG 不应少于 15 个，实得 ${files.length}`)

  const canonical = /<g transform="translate\(12 12\) scale\(([\d.]+)\) translate\(-12 -12\)" stroke-width="([\d.]+)">/
  const scaled = []
  for (const name of files) {
    const svg = readFileSync(join(dir, name), 'utf8')
    if (!/<g\b[^>]*transform=/.test(svg)) continue
    const group = svg.match(canonical)
    assert.ok(
      group,
      `${name}: 归一化必须是以画布中心为基准的均匀缩放，并把补偿后的描边写在同一个 <g> 上`,
    )
    const scale = Number(group[1])
    const strokeWidth = Number(group[2])
    assert.ok(scale > 0 && scale < 1, `${name}: 缩放系数应在 (0, 1)，实得 ${scale}`)
    assert.ok(
      Math.abs(strokeWidth - 2 / scale) < 5e-4,
      `${name}: stroke-width ${strokeWidth} 未按 1/scale 补偿（应为 ${(2 / scale).toFixed(4)}），缩小后描边会变细`,
    )
    scaled.push(name)
  }
  assert.ok(scaled.length >= 12, `应有至少 12 个图形做过缩放归一化，实得 ${scaled.length}`)
})
