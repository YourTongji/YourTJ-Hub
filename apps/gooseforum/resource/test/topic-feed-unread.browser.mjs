import assert from 'node:assert/strict'
import { mkdir } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { after, before, test } from 'node:test'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server, browser, origin
const screenshotDir = process.env.TOPIC_FEED_UNREAD_SCREENSHOT_DIR ?? join(tmpdir(), 'topic-feed-unread')
before(async () => {
  await mkdir(screenshotDir, { recursive: true })
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })

for (const theme of ['gf-light', 'gf-dark']) {
  for (const width of [360, 1440]) {
    test(`feed unread layout renders at ${width}px in ${theme}`, async () => {
      const page = await browser.newPage({ viewport: { width, height: 1100 }, deviceScaleFactor: 1 })
      try {
        await page.goto(`${origin}/assets/test/fixtures/browser/topic-feed-unread.html`)
        await page.locator('[data-case="compact-unseen-with-image"] img[alt=""]').waitFor()
        await page.locator('[data-case="compact-unseen-with-image"] img[alt=""]').evaluate(img => img.decode())
        await page.evaluate(themeName => { document.documentElement.dataset.theme = themeName }, theme)

        for (const surface of ['compact', 'preview']) {
          const imageLayout = await page.locator(`[data-case="${surface}-unseen-with-image"] img[alt=""]`).evaluate(image => {
            const meta = image.parentElement.firstElementChild.getBoundingClientRect()
            const imageRect = image.getBoundingClientRect()
            return { metaBottom: meta.bottom, metaRight: meta.right, imageTop: imageRect.top, imageLeft: imageRect.left }
          })
          if (surface === 'compact' || width >= 640) {
            assert.ok(imageLayout.imageLeft >= imageLayout.metaRight,
              `compact/wide image should remain right of metadata: ${JSON.stringify(imageLayout)}`)
          } else {
            assert.ok(imageLayout.imageTop >= imageLayout.metaBottom,
              `narrow hover preview should stack its image below metadata: ${JSON.stringify(imageLayout)}`)
          }

          const bodyUnread = page.locator(`[data-case="${surface}-unseen-with-image"]`)
          assert.equal(await bodyUnread.locator('h3').count(), 0)
          assert.equal(await bodyUnread.locator('p').count(), 1)
          assert.equal(await bodyUnread.locator('p span[aria-hidden="true"]').count(), 1)
          const inlineLayout = await bodyUnread.locator('p').evaluate(paragraph => {
            const text = paragraph.querySelector('span')
            const dot = paragraph.querySelector('[aria-hidden="true"]')
            const range = document.createRange()
            range.selectNodeContents(text)
            const firstLine = range.getClientRects()[0]
            const dotRect = dot.getBoundingClientRect()
            return {
              firstLineCenter: firstLine.top + firstLine.height / 2,
              dotCenter: dotRect.top + dotRect.height / 2,
              paragraphWidth: paragraph.clientWidth,
              paragraphScrollWidth: paragraph.scrollWidth,
            }
          })
          assert.ok(Math.abs(inlineLayout.firstLineCenter - inlineLayout.dotCenter) <= 6,
            `unread dot should sit on the excerpt's first line: ${JSON.stringify(inlineLayout)}`)
          assert.ok(inlineLayout.paragraphScrollWidth <= inlineLayout.paragraphWidth,
            `excerpt should not overflow at ${width}px: ${JSON.stringify(inlineLayout)}`)

          const imageOnly = page.locator(`[data-case="${surface}-image-only-unseen"]`)
          assert.equal(await imageOnly.locator('h3, p').count(), 0)
          assert.equal(await imageOnly.locator('.unread-meta-dot').count(), 1)
          assert.match(await imageOnly.locator('.unread-meta-dot').evaluate(dot => dot.parentElement.textContent), /小爱/)

          const empty = page.locator(`[data-case="${surface}-empty-unseen"]`)
          assert.equal(await empty.locator('h3, p').count(), 0)
          assert.equal(await empty.locator('.unread-meta-dot').count(), 1)
          assert.match(await empty.locator('.unread-meta-dot').evaluate(dot => dot.parentElement.textContent), /小爱/)
          const fallbackLayout = await empty.locator('.unread-meta-dot').evaluate(dot => {
            const authorName = dot.previousElementSibling.getBoundingClientRect()
            const dotRect = dot.getBoundingClientRect()
            return {
              nameCenter: authorName.top + authorName.height / 2,
              dotCenter: dotRect.top + dotRect.height / 2,
              rightOfName: dotRect.left >= authorName.right,
            }
          })
          assert.ok(Math.abs(fallbackLayout.nameCenter - fallbackLayout.dotCenter) <= 2,
            `metadata fallback dot should share the author line: ${JSON.stringify(fallbackLayout)}`)
          assert.ok(fallbackLayout.rightOfName, `metadata fallback dot should follow the name: ${JSON.stringify(fallbackLayout)}`)

          const seen = page.locator(`[data-case="${surface}-seen-without-image"]`)
          assert.equal(await seen.locator('h3, span[aria-hidden="true"]').count(), 0)
          assert.ok((await seen.locator('p').getAttribute('class')).includes('line-clamp-2'))

          const titled = page.locator(`[data-case="${surface}-titled-unseen"]`)
          assert.equal(await titled.locator('h3 span[aria-hidden="true"]').count(), 1)
          assert.ok((await titled.locator('p').getAttribute('class')).includes('line-clamp-2'))
        }
        for (const surface of ['compact', 'preview']) {
          const longExcerpt = page.locator(`[data-case="${surface}-long-unseen"]`)
          const paragraph = longExcerpt.locator('p')
          const longLayout = await paragraph.evaluate(paragraph => {
            const text = paragraph.querySelector('span')
            const dot = paragraph.querySelector('[aria-hidden="true"]')
            const range = document.createRange()
            range.selectNodeContents(text)
            const firstLine = range.getClientRects()[0]
            const dotRect = dot.getBoundingClientRect()
            return {
              firstLineCenter: firstLine.top + firstLine.height / 2,
              dotCenter: dotRect.top + dotRect.height / 2,
              dotWidth: dotRect.width,
              paragraphWidth: paragraph.clientWidth,
              paragraphScrollWidth: paragraph.scrollWidth,
            }
          })
          assert.ok(longLayout.dotWidth > 0, `long excerpt unread dot should remain visible: ${JSON.stringify(longLayout)}`)
          assert.ok(Math.abs(longLayout.firstLineCenter - longLayout.dotCenter) <= 6,
            `long excerpt dot should stay on its first line: ${JSON.stringify(longLayout)}`)
          assert.ok(longLayout.paragraphScrollWidth <= longLayout.paragraphWidth,
            `long excerpt should not overflow: ${JSON.stringify(longLayout)}`)

          const longAuthor = page.locator(`[data-case="${surface}-empty-long-author-unseen"]`)
          const fallback = longAuthor.locator('.unread-meta-dot')
          assert.equal(await fallback.count(), 1)
          const fallbackLayout = await fallback.evaluate(dot => {
            const name = dot.previousElementSibling.getBoundingClientRect()
            const dotRect = dot.getBoundingClientRect()
            return {
              nameCenter: name.top + name.height / 2,
              dotCenter: dotRect.top + dotRect.height / 2,
              nameRight: name.right,
              dotLeft: dotRect.left,
              dotWidth: dotRect.width,
            }
          })
          assert.ok(fallbackLayout.dotWidth > 0, `metadata unread dot should remain visible: ${JSON.stringify(fallbackLayout)}`)
          assert.ok(Math.abs(fallbackLayout.nameCenter - fallbackLayout.dotCenter) <= 2,
            `metadata unread dot should stay beside a long author: ${JSON.stringify(fallbackLayout)}`)
          assert.ok(fallbackLayout.dotLeft >= fallbackLayout.nameRight,
            `metadata unread dot should follow the author: ${JSON.stringify(fallbackLayout)}`)
        }
        const pageOverflow = await page.evaluate(() => ({
          overflows: document.documentElement.scrollWidth > innerWidth,
          elements: [...document.querySelectorAll('*')]
            .map(element => ({ element: element.tagName, className: element.className, case: element.closest('[data-case]')?.getAttribute('data-case'), ...element.getBoundingClientRect().toJSON() }))
            .filter(box => box.right > innerWidth || box.left < 0)
            .slice(0, 30),
        }))
        assert.ok(!pageOverflow.overflows, `page should not overflow horizontally at ${width}px: ${JSON.stringify(pageOverflow)}`)

        await page.screenshot({ path: join(screenshotDir, `${theme}-${width}.png`), fullPage: true })
      } finally { await page.close() }
    })
  }
}
