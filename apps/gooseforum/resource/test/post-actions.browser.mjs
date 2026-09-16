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

for (const width of [320, 1280]) {
  for (const count of [0, 12]) {
    test(`reply actions stay centered at width ${width}, likes ${count}`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 900 } })
      try {
        await page.goto(`${origin}/assets/test/fixtures/browser/post-actions.html?count=${count}`)
        for (const mode of ['flat', 'tree']) {
          for (const title of ['Like', 'Bookmark', 'Share', 'Report']) {
            const button = page.locator(`#${mode} button[title="${title}"]`).first()
            await button.waitFor()
            await button.hover()
            const geometry = await button.evaluate(el => {
              const b = el.getBoundingClientRect()
              const visible = [...el.children].filter(child => {
                const style = getComputedStyle(child)
                return style.display !== 'none' && style.position !== 'absolute'
              }).map(child => child.getBoundingClientRect())
              const left = Math.min(...visible.map(r => r.left))
              const right = Math.max(...visible.map(r => r.right))
              return { delta: (left + right - b.left - b.right) / 2, width: b.width }
            })
            assert.ok(Math.abs(geometry.delta) < 1, `${mode} ${title}: ${JSON.stringify(geometry)}`)
            assert.ok(geometry.width >= 28, `${mode} ${title} has a smaller target than adjacent actions`)
          }
        }
      } finally { await page.close() }
    })
  }
}
