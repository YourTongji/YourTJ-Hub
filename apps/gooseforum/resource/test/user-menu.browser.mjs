import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

let server, browser, origin

before(async () => {
  server = await createServer({
    root: fileURLToPath(new URL('../', import.meta.url)),
    optimizeDeps: { entries: ['test/fixtures/browser/user-menu.html'] },
    server: { host: '127.0.0.1', port: 0, open: false },
    logLevel: 'error',
  })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})

after(async () => {
  await browser?.close()
  await server?.close()
})

const fixtureUrl = (theme) => `${origin}/assets/test/fixtures/browser/user-menu.html?theme=${theme}`

async function openPage(page, theme) {
  await page.route('**/api/forum/anonymous/state', route => route.fulfill({ json: { code: 0, result: { persona: null } } }))
  await page.route('**/api/forum/unread-status', route => route.fulfill({ json: { code: 0, result: { notifications: false, messages: false } } }))
  await page.goto(fixtureUrl(theme))
  await page.waitForFunction(() => document.documentElement.dataset.ready === 'true')
  const trigger = page.locator('header button[aria-haspopup="menu"][aria-label="用户菜单"]')
  await trigger.click()
  await page.waitForFunction(() => document.querySelector('header button[aria-haspopup="menu"][aria-label="用户菜单"]')?.getAttribute('aria-expanded') === 'true')
  await page.waitForFunction(() => {
    const menu = document.querySelector('[data-user-menu]')
    return menu?.parentElement && Number(getComputedStyle(menu.parentElement).opacity) > 0.99
  })
  return page.locator('[data-user-menu]')
}

test('menu groups fit mobile and desktop viewports and follow light and dark theme tokens', async () => {
  const surfaces = new Map()
  for (const theme of ['light', 'dark']) {
    for (const width of [360, 1440]) {
      const page = await browser.newPage({ viewport: { width, height: 900 } })
      try {
        const menu = await openPage(page, theme)
        const box = await menu.boundingBox()
        assert.ok(box, `${theme} menu at ${width}px should be visible`)
        assert.ok(box.x >= 0 && box.x + box.width <= width, `menu should fit ${width}px viewport: ${JSON.stringify(box)}`)
        assert.ok(await menu.locator('[data-user-menu-group="personal"] a[href="/drafts"]').isVisible(), 'drafts should be in the personal group')
        assert.ok(await menu.locator('[data-user-menu-group="settings"] a[href="/settings"]').isVisible(), 'settings entry should remain visible')

        const groupState = await menu.evaluate((element) => {
          const preferences = element.querySelector('[data-user-menu-group="preferences"]')
          const settings = element.querySelector('[data-user-menu-group="settings"]')
          return {
            surface: getComputedStyle(element).backgroundColor,
            preferencesBorderWidth: getComputedStyle(preferences).borderTopWidth,
            preferencesDisplay: getComputedStyle(preferences).display,
            settingsBorderWidth: getComputedStyle(settings).borderTopWidth,
          }
        })
        assert.equal(groupState.preferencesBorderWidth, '1px')
        assert.equal(groupState.settingsBorderWidth, '1px')
        assert.equal(groupState.preferencesDisplay !== 'none', width < 640, 'theme and language controls keep their mobile-only visibility')
        assert.notEqual(groupState.surface, 'rgba(0, 0, 0, 0)', 'menu surface should use the active theme background')
        if (width === 360) {
          assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'mobile page should not overflow horizontally')
          if (process.env.YOURTJ_USER_MENU_SCREENSHOT_DIR) {
            await page.screenshot({ path: `${process.env.YOURTJ_USER_MENU_SCREENSHOT_DIR}/user-menu-${width}-${theme}.png` })
          }
        }
        surfaces.set(`${theme}-${width}`, groupState.surface)
      } finally {
        await page.close()
      }
    }
  }
  assert.notEqual(surfaces.get('light-360'), surfaces.get('dark-360'), 'menu surface should change with the theme')
})

test('keyboard tab reaches personal menu links in order without closing the menu', async () => {
  const page = await browser.newPage({ viewport: { width: 360, height: 800 } })
  try {
    await page.route('**/api/forum/anonymous/state', route => route.fulfill({ json: { code: 0, result: { persona: null } } }))
    await page.route('**/api/forum/unread-status', route => route.fulfill({ json: { code: 0, result: { notifications: false, messages: false } } }))
    await page.goto(fixtureUrl('light'))
    await page.waitForFunction(() => document.documentElement.dataset.ready === 'true')
    const trigger = page.locator('header button[aria-haspopup="menu"][aria-label="用户菜单"]')
    await trigger.focus()
    await page.waitForFunction(() => document.querySelector('header button[aria-haspopup="menu"][aria-label="用户菜单"]')?.getAttribute('aria-expanded') === 'true')

    const expectedHrefs = ['/u/42', '/settings?tab=privacy', '/notifications', '/messages', '/drafts']
    for (const href of expectedHrefs) {
      await page.keyboard.press('Tab')
      await page.waitForFunction((expected) => document.activeElement?.getAttribute('href') === expected, href)
      assert.equal(await trigger.getAttribute('aria-expanded'), 'true', `tabbing to ${href} should keep the menu open`)
    }
  } finally {
    await page.close()
  }
})

test('touch opens the avatar menu and activates the drafts link', async () => {
  const context = await browser.newContext({ viewport: { width: 360, height: 800 }, isMobile: true, hasTouch: true })
  const page = await context.newPage()
  try {
    await page.addInitScript(() => {
      document.addEventListener('click', (event) => {
        if (event.target instanceof Element && event.target.closest('a[href="/drafts"]')) {
          event.preventDefault()
          document.documentElement.dataset.draftActivated = 'true'
        }
      }, true)
    })
    await page.route('**/api/forum/anonymous/state', route => route.fulfill({ json: { code: 0, result: { persona: null } } }))
    await page.route('**/api/forum/unread-status', route => route.fulfill({ json: { code: 0, result: { notifications: false, messages: false } } }))
    await page.goto(fixtureUrl('light'))
    await page.waitForFunction(() => document.documentElement.dataset.ready === 'true')

    const trigger = page.locator('header button[aria-haspopup="menu"][aria-label="用户菜单"]')
    const triggerBox = await trigger.boundingBox()
    assert.ok(triggerBox)
    await page.touchscreen.tap(triggerBox.x + triggerBox.width / 2, triggerBox.y + triggerBox.height / 2)
    await page.waitForFunction(() => document.querySelector('header button[aria-haspopup="menu"][aria-label="用户菜单"]')?.getAttribute('aria-expanded') === 'true')

    const drafts = page.locator('[data-user-menu-group="personal"] a[href="/drafts"]')
    const draftsBox = await drafts.boundingBox()
    assert.ok(draftsBox)
    // Chromium can report an exact CSS row as 35.99998474121094px after
    // transforming fractional viewport coordinates. Keep the 36px minimum
    // with a hundredth-pixel geometry tolerance, below any visible regression.
    assert.ok(draftsBox.height >= 36 - 0.01, `drafts touch target should retain its 36px menu row, got ${draftsBox.height}px`)
    await page.touchscreen.tap(draftsBox.x + draftsBox.width / 2, draftsBox.y + draftsBox.height / 2)
    await page.waitForFunction(() => document.documentElement.dataset.draftActivated === 'true')
    assert.equal(await page.locator('[data-user-menu]').count(), 1)
  } finally {
    await context.close()
  }
})
