import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server
let browser
let origin

before(async () => {
  server = await createServer({
    root: fileURLToPath(new URL('../', import.meta.url)),
    server: { host: '127.0.0.1', port: 0, open: false },
  })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})

after(async () => {
  await browser?.close()
  await server?.close()
})

test('Twemoji plain copy matches native nested block, list and table copy', async () => {
  const context = await browser.newContext()
  await context.grantPermissions(['clipboard-read', 'clipboard-write'], { origin })
  const page = await context.newPage()
  try {
    await page.route('**/twemoji/72x72/**', (route) =>
      route.fulfill({
        contentType: 'image/png',
        body: Buffer.from(
          'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=',
          'base64',
        ),
      }),
    )
    await page.goto(`${origin}/assets/test/fixtures/browser/twemoji-copy.html`)
    await page.locator('#copy-root img.twemoji').nth(4).waitFor()

    await page.evaluate(() => {
      const source = document.querySelector('#copy-root')
      const nativeCopy = source.cloneNode(true)
      nativeCopy.id = 'native-copy'
      for (const image of nativeCopy.querySelectorAll('img.twemoji')) {
        image.replaceWith(document.createTextNode(image.dataset.emoji ?? image.alt))
      }
      document.body.append(nativeCopy)
    })

    const copyText = async (selector) => {
      await page.locator(selector).evaluate((element) => {
        const range = document.createRange()
        range.selectNodeContents(element)
        const selection = window.getSelection()
        selection.removeAllRanges()
        selection.addRange(range)
      })
      await page.keyboard.press(`${process.platform === 'darwin' ? 'Meta' : 'Control'}+C`)
      return page.evaluate(() => navigator.clipboard.readText())
    }

    const nativeText = await copyText('#native-copy')
    const twemojiText = await copyText('#copy-root')
    assert.equal(twemojiText, nativeText)
    assert.match(twemojiText, /段落一 🐦‍🔥/)
    assert.match(twemojiText, /列表二 🇨🇳/)
    assert.match(twemojiText, /表格甲\s+表格乙 👩🏽‍💻/)
  } finally {
    await page.close()
    await context.close()
  }
})

test('multiple copied ranges join native range text without extra block breaks', async () => {
  const context = await browser.newContext()
  await context.grantPermissions(['clipboard-read', 'clipboard-write'], { origin })
  const page = await context.newPage()
  try {
    await page.route('**/twemoji/72x72/**', (route) =>
      route.fulfill({
        contentType: 'image/png',
        body: Buffer.from(
          'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=',
          'base64',
        ),
      }),
    )
    await page.goto(`${origin}/assets/test/fixtures/browser/twemoji-copy.html`)
    await page.locator('#copy-root img.twemoji').nth(4).waitFor()

    const result = await page.evaluate(() => {
      const root = document.querySelector('#copy-root')
      const ranges = [root.querySelector('p'), root.querySelector('li:last-child')].map((node) => {
        const range = document.createRange()
        range.selectNodeContents(node)
        return range
      })
      const nativeText = ranges.map((range) => {
        const fragment = range.cloneContents()
        for (const image of fragment.querySelectorAll('img.twemoji')) {
          image.replaceWith(document.createTextNode(image.dataset.emoji ?? image.alt))
        }
        const host = document.createElement('div')
        host.append(fragment)
        document.body.append(host)
        const text = host.innerText
        host.remove()
        return text
      }).join('')
      const actualSelection = window.getSelection
      Object.defineProperty(window, 'getSelection', {
        configurable: true,
        value: () => ({
          isCollapsed: false,
          rangeCount: ranges.length,
          getRangeAt: (index) => ranges[index],
        }),
      })
      const values = {}
      const event = new Event('copy', { bubbles: true, cancelable: true })
      Object.defineProperty(event, 'clipboardData', {
        value: { types: [], setData: (type, value) => (values[type] = value) },
      })
      document.dispatchEvent(event)
      delete window.getSelection
      Object.defineProperty(window, 'getSelection', { value: actualSelection })
      return { nativeText, copiedText: values['text/plain'], defaultPrevented: event.defaultPrevented }
    })

    assert.equal(result.defaultPrevented, true)
    assert.equal(result.copiedText, result.nativeText)
    assert.match(result.copiedText, /段落一 🐦‍🔥/)
    assert.match(result.copiedText, /列表二 🇨🇳/)
  } finally {
    await page.close()
    await context.close()
  }
})
