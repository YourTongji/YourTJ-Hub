import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium, webkit } from 'playwright'
import { createServer } from 'vite'

let server
let browser
let origin
const browserType = process.env.TWEMOJI_BROWSER === 'webkit' ? webkit : chromium
const isWebKit = process.env.TWEMOJI_BROWSER === 'webkit'

before(async () => {
  server = await createServer({
    root: fileURLToPath(new URL('../', import.meta.url)),
    server: { host: '127.0.0.1', port: 0, open: false },
  })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await browserType.launch()
})

after(async () => {
  await browser?.close()
  await server?.close()
})

for (const theme of ['gf-light', 'gf-dark']) {
  test(`Twemoji stays inline, transparent, and outside image preview (${theme})`, async () => {
    const context = await browser.newContext({ viewport: { width: 720, height: 640 } })
    if (!isWebKit) await context.grantPermissions(['clipboard-read', 'clipboard-write'], { origin })
    const page = await context.newPage()
    try {
      await page.route('**/photo.png', (route) =>
        route.fulfill({
          contentType: 'image/png',
          body: Buffer.from(
            'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=',
            'base64',
          ),
        }),
      )
      await page.goto(`${origin}/assets/test/fixtures/browser/twemoji-preview.html?theme=${theme}`)
      await page.locator('#content img.twemoji').first().waitFor()
      await page.waitForFunction(() =>
        [...document.querySelectorAll('#content img.twemoji')].every((image) => image.complete && image.naturalWidth === 72),
      )

      const rendered = await page.evaluate(() => {
        const images = [...document.querySelectorAll('#content img.twemoji')]
        const photo = document.querySelector('#photo')
        return {
          emoji: images.map((image) => {
            const style = getComputedStyle(image)
            return {
              alt: image.alt,
              src: image.currentSrc,
              width: image.naturalWidth,
              display: style.display,
              borderWidth: style.borderWidth,
              borderRadius: style.borderRadius,
              padding: style.padding,
              backgroundColor: style.backgroundColor,
              boxShadow: style.boxShadow,
              filter: style.filter,
              objectFit: style.objectFit,
              cursor: style.cursor,
            }
          }),
          photo: photo && {
            borderWidth: getComputedStyle(photo).borderWidth,
            outlineWidth: getComputedStyle(photo).outlineWidth,
            borderRadius: getComputedStyle(photo).borderRadius,
            cursor: getComputedStyle(photo).cursor,
          },
          mixedParagraphAlignment: getComputedStyle(document.querySelector('#copy').parentElement).textAlign,
        }
      })

      assert.deepEqual(
        rendered.emoji.map(({ alt }) => alt),
        ['🇺🇸', '👩🏽‍🚀', '🪿', '🐦‍🔥', '🇺🇸', '👩🏽‍🚀', '🪿', '🐦‍🔥'],
      )
      assert.ok(rendered.emoji.every(({ src, width }) => src.includes('/twemoji/72x72/') && width === 72))
      for (const image of rendered.emoji) {
        assert.equal(image.display, 'inline-block')
        assert.equal(image.borderWidth, '0px')
        assert.equal(image.borderRadius, '0px')
        assert.equal(image.padding, '0px')
        assert.equal(image.backgroundColor, 'rgba(0, 0, 0, 0)')
        assert.equal(image.boxShadow, 'none')
        assert.equal(image.filter, 'none')
        assert.equal(image.objectFit, 'contain')
        assert.notEqual(image.cursor, 'zoom-in')
      }
      assert.equal(rendered.mixedParagraphAlignment, 'start')
      // 图片用 1px 低透明外描边（outline）代替带色相的边框，见 prose.css
      assert.equal(rendered.photo.borderWidth, '0px')
      assert.equal(rendered.photo.outlineWidth, '1px')
      assert.equal(rendered.photo.borderRadius, '8px')
      assert.equal(rendered.photo.cursor, 'zoom-in')

      await page.locator('#content img.twemoji').first().hover()
      const hoveredEmoji = await page.locator('#content img.twemoji').first().evaluate((image) => {
        const style = getComputedStyle(image)
        return {
          cursor: style.cursor,
          transform: style.transform,
          boxShadow: style.boxShadow,
          borderWidth: style.borderWidth,
          borderRadius: style.borderRadius,
          backgroundColor: style.backgroundColor,
        }
      })
      assert.notEqual(hoveredEmoji.cursor, 'zoom-in')
      assert.equal(hoveredEmoji.transform, 'none')
      assert.equal(hoveredEmoji.boxShadow, 'none')
      assert.equal(hoveredEmoji.borderWidth, '0px')
      assert.equal(hoveredEmoji.borderRadius, '0px')
      assert.equal(hoveredEmoji.backgroundColor, 'rgba(0, 0, 0, 0)')

      await page.locator('#content img.twemoji').first().click()
      assert.equal(await page.locator('[role="dialog"][aria-modal="true"]').count(), 0)

      await page.locator('#photo').click()
      const viewer = page.locator('[role="dialog"][aria-modal="true"]')
      await viewer.waitFor()
      assert.equal(await viewer.getAttribute('aria-label'), 'Photo attachment')
      assert.deepEqual(
        await viewer.locator('img').evaluateAll((images) => images.map((image) => image.alt)),
        ['Photo attachment'],
      )

      await page.keyboard.press('Escape')
      await viewer.waitFor({ state: 'detached' })
      await page.locator('#copy').evaluate((element) => {
        const range = document.createRange()
        range.selectNodeContents(element)
        const selection = window.getSelection()
        selection.removeAllRanges()
        selection.addRange(range)
      })
      if (isWebKit) {
        const copied = await page.evaluate(() => {
          const values = {}
          const event = new Event('copy', { bubbles: true, cancelable: true })
          Object.defineProperty(event, 'clipboardData', {
            value: { types: [], setData: (type, value) => (values[type] = value) },
          })
          document.dispatchEvent(event)
          return { values, defaultPrevented: event.defaultPrevented }
        })
        assert.equal(copied.defaultPrevented, true)
        assert.equal(copied.values['text/plain'], 'copy me 🇺🇸, 👩🏽‍🚀, 🪿, and 🐦‍🔥')
        assert.equal(copied.values['text/html'], 'copy me 🇺🇸, 👩🏽‍🚀, 🪿, and 🐦‍🔥')
      } else {
        await page.keyboard.press(`${process.platform === 'darwin' ? 'Meta' : 'Control'}+C`)
        assert.equal(await page.evaluate(() => navigator.clipboard.readText()), 'copy me 🇺🇸, 👩🏽‍🚀, 🪿, and 🐦‍🔥')
      }
    } finally {
      await page.close()
      await context.close()
    }
  })
}
