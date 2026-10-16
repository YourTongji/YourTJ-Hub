import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'

// issue #1124: profile privacy settings and hidden profile sections must stay
// usable at the narrowest and a wide viewport, for owner, member and guest.
let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })

const fixture = query => `${origin}/assets/test/fixtures/browser/profile-privacy.html?${query}`
const fits = page => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)

test('privacy fixture renders dynamically compiled Emoji and text through the shared app plugin', async () => {
  const { page } = await open(320, 'viewer=owner&twemoji=1', {})
  try {
    const heading = page.locator('h1')
    await heading.getByText('同济用户').waitFor()
    assert.equal(await heading.evaluate(element => element.textContent), '同济用户')
    await heading.locator('img.twemoji[alt="🐦‍🔥"]').waitFor()
  } finally {
    await page.close()
  }
})

async function open(width, query, privacy) {
  const page = await browser.newPage({ viewport: { width, height: 900 } })
  const patches = []
  let failNext = false
  await page.route('**/api/**', route => route.fulfill({ status: 404, json: { code: 1, message: 'not in fixture' } }))
  await page.route('**/api/forum/unread-status', route => route.fulfill({ json: { code: 0, result: { notifications: false, messages: false } } }))
  await page.route('**/api/user/privacy-settings', async route => {
    if (route.request().method() === 'PATCH') {
      const body = route.request().postDataJSON()
      patches.push(body)
      if (failNext) { failNext = false; return route.fulfill({ status: 503, json: { code: 1, message: 'unavailable' } }) }
      Object.assign(privacy, body)
    }
    await route.fulfill({ json: { code: 0, result: privacy } })
  })
  await page.goto(fixture(query))
  return { page, patches, failNext: () => { failNext = true } }
}

async function pick(page, row, option) {
  await page.locator(`[data-test="profile-privacy-row-${row}"] [role="combobox"]`).click()
  await page.getByRole('listbox').getByRole('option', { name: option, exact: true }).click()
  // Let the menu finish its exit animation before the next interaction. A CSS
  // locator still sees it once the confirmation marks the page aria-hidden.
  await page.locator('[role="listbox"]').waitFor({ state: 'detached' })
}

for (const width of [320, 1280]) {
  test(`owner changes profile visibility from settings at ${width}px`, async () => {
    const privacy = { profileTopics: 'everyone', profileActivity: 'everyone', profileLikes: 'everyone', followingList: 'self', followersList: 'everyone' }
    const { page, patches, failNext } = await open(width, 'screen=settings&tab=privacy', privacy)
    try {
      const row = page.locator('[data-test="profile-privacy-row-profileLikes"]')
      await row.waitFor()
      assert.ok(await fits(page), 'privacy settings must not overflow horizontally')
      assert.equal(await page.getByRole('combobox', { name: '谁能看到关注列表' }).textContent().then(t => t.trim()), '仅自己')

      // Bookmarks are a fixed row; on wide screens its value lines up with the pickers' values.
      const textBox = selector => page.locator(selector).evaluate(el => {
        const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT, { acceptNode: n => (n.textContent.trim() ? 1 : 3) })
        const range = document.createRange()
        range.selectNodeContents(walker.nextNode())
        const { left, top } = range.getBoundingClientRect()
        return { left, top }
      })
      const fixed = await textBox('[data-test="profile-privacy-bookmarks-audience"]')
      const picker = await textBox('[data-test="profile-privacy-row-followingList"] [role="combobox"]')
      if (width >= 640) assert.ok(Math.abs(fixed.left - picker.left) <= 2, `bookmarks value must line up with picker values (${fixed.left} vs ${picker.left})`)
      assert.equal(await page.locator('[data-test="profile-privacy-row-bookmarks"] [role="combobox"]').count(), 0)

      // Narrowing saves straight away and confirms on the row without moving it.
      const top = () => row.evaluate(el => el.getBoundingClientRect().top + window.scrollY)
      const before = await top()
      await pick(page, 'profileLikes', '仅自己')
      await row.locator('[data-test="profile-privacy-saved"]').waitFor()
      assert.deepEqual(patches.at(-1), { profileLikes: 'self' })
      assert.equal(await top(), before, 'saving must not shift the settings layout')
      assert.equal(await page.locator('[data-test="profile-privacy-announcement"]').textContent(), '可见范围已保存到你的账号')

      // A failed save keeps the confirmed value and explains next to the row.
      failNext()
      await pick(page, 'profileTopics', '登录用户')
      await page.locator('[data-test="profile-privacy-row-profileTopics"] [role="alert"]').waitFor()
      assert.match(await page.locator('[data-test="profile-privacy-row-profileTopics"] [role="combobox"]').textContent(), /所有人/)

      // Widening asks first; Escape leaves the setting untouched.
      const count = patches.length
      await pick(page, 'followingList', '所有人')
      const dialog = page.locator('[data-test="profile-privacy-widen"]')
      await dialog.waitFor()
      assert.equal(await page.evaluate(() => document.activeElement?.textContent?.trim()), '取消')
      const box = await dialog.boundingBox()
      assert.ok(box && box.x >= 0 && box.x + box.width <= width, 'confirmation must fit the viewport')
      await page.keyboard.press('Escape')
      await dialog.waitFor({ state: 'detached' })
      assert.equal(patches.length, count)
      assert.equal(await page.evaluate(() => document.activeElement?.getAttribute('aria-label')), '谁能看到关注列表', 'focus returns to the row')
      await pick(page, 'followingList', '所有人')
      await dialog.getByRole('button', { name: '扩大可见范围' }).click()
      await dialog.waitFor({ state: 'detached' })
      assert.deepEqual(patches.at(-1), { followingList: 'everyone' })
      if (process.env.YOURTJ_PRIVACY_SCREENSHOT_DIR) await page.screenshot({ path: `${process.env.YOURTJ_PRIVACY_SCREENSHOT_DIR}/settings-${width}.png`, fullPage: true })
    } finally {
      await page.close()
    }
  })

  for (const viewer of ['owner', 'other', 'guest']) {
    test(`${viewer} sees the right likes state at ${width}px`, async () => {
      const { page } = await open(width, `viewer=${viewer}`, {})
      try {
        await page.getByText('同济用户', { exact: true }).first().waitFor()
        assert.ok(await fits(page), 'profile must not overflow horizontally')
        const hidden = page.locator('[data-test="profile-section-hidden"]')
        const login = hidden.getByRole('link', { name: '登录', exact: true })
        if (viewer === 'owner') {
          await page.locator('[data-test="profile-section-owner-note"]').getByText('只有关注你的人能看到这部分内容。').waitFor()
          assert.equal(await page.getByRole('link', { name: '修改可见范围' }).getAttribute('href'), '/settings?tab=privacy')
          assert.equal(await hidden.count(), 0)
        } else if (viewer === 'other') {
          await hidden.getByText('同济用户 只向关注者展示这部分内容，关注后即可查看。').waitFor()
          assert.equal(await login.count(), 0)
        } else {
          await hidden.getByText('同济用户 只向登录用户展示这部分内容。').waitFor()
          assert.match(await login.getAttribute('href'), /^\/login\?redirect=/)
        }
        if (viewer !== 'owner') assert.equal(await page.getByText('分享一件最近让你开心的小事').count(), 0)
        if (process.env.YOURTJ_PRIVACY_SCREENSHOT_DIR) await page.screenshot({ path: `${process.env.YOURTJ_PRIVACY_SCREENSHOT_DIR}/profile-${viewer}-${width}.png`, fullPage: true })
      } finally {
        await page.close()
      }
    })
  }
}
