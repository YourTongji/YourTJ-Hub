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

for (const width of [320, 1280]) {
  for (const role of ['reader', 'guest', 'own']) {
    test(`deep-link topic capsule stays usable at width ${width}, ${role}`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 900 } })
      try {
        await page.goto(`${origin}/assets/test/fixtures/browser/post-actions.html?topic&${role}`)
        const pill = page.locator('[data-test="floating-controls"] .gf-floating-surface')
        await pill.waitFor()
        const like = pill.locator('button[title="Like"]')
        await like.waitFor({ state: 'visible' })
        assert.equal(await pill.locator('[data-test="mobile-topic-reply"]').isVisible(), true)
        assert.equal(await pill.locator('button[title="Report"]').count(), role === 'reader' ? 1 : 0)
        assert.equal(await pill.locator('button[title="Share"]').count(), role === 'guest' ? 0 : 1)
        if (role !== 'guest') {
          await page.evaluate(() => {
            navigator.clipboard.writeText = async (text) => { window.sharedTopicUrl = text }
          })
          await pill.locator('button[title="Share"]').click()
          assert.equal(await page.evaluate(() => window.sharedTopicUrl), `${origin}/p/post/42`)
        }
        if (role === 'reader') {
          await pill.locator('button[title="Report"]').click()
          await page.getByRole('dialog').waitFor({ state: 'visible' })
        }
        const bounds = await pill.evaluate(el => {
          const box = el.getBoundingClientRect()
          return { left: box.left, right: box.right, width: innerWidth }
        })
        assert.ok(bounds.left >= 0 && bounds.right <= bounds.width, `capsule overflows: ${JSON.stringify(bounds)}`)
      } finally { await page.close() }
    })
  }
}
