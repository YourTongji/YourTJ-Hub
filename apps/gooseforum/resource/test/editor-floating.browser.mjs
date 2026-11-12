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

const selectionBar = '[data-test="editor-selection-toolbar"]'
const imageBar = '[data-test="editor-image-toolbar"]'
const imageFrame = '[data-test="editor-image-frame"]'

async function openEditor(query = '', viewport = { width: 900, height: 900 }) {
  const page = await browser.newPage({ viewport })
  const errors = []
  page.on('pageerror', (err) => errors.push(err.message))
  await page.goto(`${origin}/assets/test/fixtures/browser/editor-floating.html${query}`)
  const editable = page.locator('.vditor-wysiwyg .vditor-reset')
  await editable.waitFor()
  await page.waitForFunction(() => document.querySelectorAll('.vditor-wysiwyg .vditor-reset img').length >= 3)
  return { page, editable, errors }
}

/** 程序选中某段落的前 n 个字（与键盘 Shift+方向键一样不经过指针） */
async function selectText(locator, length = 4) {
  await locator.evaluate((el, n) => {
    const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT, { acceptNode: (node) => (node.textContent.trim() ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_SKIP) })
    const text = walker.nextNode()
    const range = document.createRange()
    range.setStart(text, 0)
    range.setEnd(text, Math.min(n, text.length))
    el.closest('[contenteditable]').focus()
    getSelection().removeAllRanges()
    getSelection().addRange(range)
  }, length)
}

/** 浮层中心点命中的确是浮层自身：没有被顶栏、对话框或编辑区裁切遮挡 */
async function assertOnTop(page, selector) {
  const hit = await page.locator(selector).evaluate((el) => {
    const rect = el.getBoundingClientRect()
    const top = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2)
    return { inside: el.contains(top), within: rect.left >= 0 && rect.right <= innerWidth && rect.top >= 0 && rect.bottom <= innerHeight }
  })
  assert.deepEqual(hit, { inside: true, within: true }, `${selector} must be visible and on top`)
}

test('a text selection shows the format bar above it and applies bold', async () => {
  const { page, editable, errors } = await openEditor()
  try {
    const paragraph = editable.locator('p').first()
    await selectText(paragraph)
    const bar = page.locator(selectionBar)
    await bar.waitFor({ state: 'visible' })
    await assertOnTop(page, selectionBar)
    const [barBox, lineBox] = await Promise.all([bar.boundingBox(), paragraph.boundingBox()])
    assert.ok(barBox.y + barBox.height <= lineBox.y + 1 || barBox.y >= lineBox.y, 'the bar never covers the selected line')
    await bar.locator('[data-command="bold"]').click()
    await page.waitForFunction(() => window.testValue().startsWith('**第一段文**'))
    assert.deepEqual(errors, [])
  } finally {
    await page.close()
  }
})

test('the format bar hides Vditor block popovers instead of stacking on them', async () => {
  const { page, editable } = await openEditor(`?value=${encodeURIComponent('> 引用里的文字\n\n![a](/assets/test/fixtures/browser/topic-feed-art.svg)\n\n![b](/assets/test/fixtures/browser/topic-feed-art.svg) ![c](/assets/test/fixtures/browser/topic-feed-art.svg)')}`)
  try {
    const quote = editable.locator('blockquote p').first()
    await quote.click()
    const popover = page.locator('.vditor-wysiwyg > .vditor-panel').first()
    await page.waitForFunction(() => document.querySelector('.vditor-wysiwyg > .vditor-panel')?.style.display === 'block')
    assert.equal(await popover.evaluate((el) => getComputedStyle(el).zIndex), '30')
    await selectText(quote)
    await page.locator(selectionBar).waitFor({ state: 'visible' })
    await page.waitForFunction(() => getComputedStyle(document.querySelector('.vditor-wysiwyg > .vditor-panel')).opacity === '0')
    assert.equal(await popover.evaluate((el) => getComputedStyle(el).pointerEvents), 'none')
  } finally {
    await page.close()
  }
})

test('clicking an image selects it: width presets, alt text and delete', async () => {
  const { page, editable, errors } = await openEditor()
  try {
    const cover = editable.locator('img').first()
    await cover.click()
    const bar = page.locator(imageBar)
    await bar.waitFor({ state: 'visible' })
    await assertOnTop(page, imageBar)
    // Vditor's own address popover stays closed; our bar replaces it.
    assert.equal(await page.locator('.vditor-wysiwyg > .vditor-panel').first().evaluate((el) => el.style.display), 'none')

    await bar.locator('.gf-editor-float-segment', { hasText: '50%' }).click()
    await page.waitForFunction(() => window.testValue().includes('topic-feed-art.svg#w=50)'))
    const width = await cover.evaluate((img) => img.getBoundingClientRect().width / img.parentElement.getBoundingClientRect().width)
    assert.ok(Math.abs(width - 0.5) < 0.02, `50% renders at half the column, got ${width}`)
    assert.equal(await bar.locator('[aria-pressed="true"]').innerText(), '50%')

    await bar.getByRole('button', { name: '替代文字' }).click()
    await page.locator('.gf-editor-float-input').fill('封面图')
    await page.keyboard.press('Enter')
    await page.waitForFunction(() => window.testValue().includes('![封面图]('))

    await cover.click()
    await bar.waitFor({ state: 'visible' })
    await page.keyboard.press('Backspace')
    await page.waitForFunction(() => !window.testValue().includes('封面图'))
    await page.locator(imageBar).waitFor({ state: 'detached' })
    assert.deepEqual(errors, [])
  } finally {
    await page.close()
  }
})

test('dragging a frame handle snaps the width to a preset', async () => {
  const { page, editable } = await openEditor()
  try {
    const cover = editable.locator('img').first()
    await cover.click()
    const handle = page.locator(`${imageFrame} [data-edge="right"]`)
    await handle.waitFor({ state: 'visible' })
    const box = await handle.boundingBox()
    const column = await cover.evaluate((img) => img.parentElement.getBoundingClientRect().width)
    const start = { x: box.x + box.width / 2, y: box.y + box.height / 2 }
    const imgBox = await cover.boundingBox()
    // Symmetric resize: drag the right edge so the image spans ~70% of the column.
    const targetRight = imgBox.x + imgBox.width / 2 + column * 0.35
    await page.mouse.move(start.x, start.y)
    await page.mouse.down()
    await page.mouse.move(targetRight, start.y, { steps: 8 })
    assert.match(await page.locator('.gf-editor-image-badge').innerText(), /^(66|75)%$/)
    await page.mouse.up()
    await page.waitForFunction(() => /topic-feed-art\.svg#w=(66|75)\)/.test(window.testValue()))
    assert.equal(await cover.evaluate((img) => img.style.width), '', 'live drag styles are cleared on release')
  } finally {
    await page.close()
  }
})

test('adjacent sized images share one centred row', async () => {
  const { page, editable } = await openEditor()
  try {
    const row = await editable.locator('img[src*="#w=50"]').evaluateAll((images) => images.map((img) => {
      const rect = img.getBoundingClientRect()
      return { top: Math.round(rect.top), width: rect.width, column: img.parentElement.getBoundingClientRect().width }
    }))
    assert.equal(row.length, 2)
    assert.equal(row[0].top, row[1].top)
    for (const image of row) assert.ok(image.width > image.column * 0.45 && image.width < image.column * 0.5)
  } finally {
    await page.close()
  }
})

test('floating bars inside a transformed dialog stay attached to their anchor', async () => {
  const { page, editable, errors } = await openEditor('?host=dialog')
  try {
    const paragraph = editable.locator('p').first()
    await selectText(paragraph)
    const bar = page.locator(selectionBar)
    await bar.waitFor({ state: 'visible' })
    assert.equal(await bar.evaluate((el) => el.parentElement.getAttribute('role')), 'dialog')
    await assertOnTop(page, selectionBar)
    const [barBox, lineBox] = await Promise.all([bar.boundingBox(), paragraph.boundingBox()])
    const gap = barBox.y >= lineBox.y ? barBox.y - (lineBox.y + lineBox.height) : lineBox.y - (barBox.y + barBox.height)
    assert.ok(gap >= 0 && gap < 40, `bar sits next to the selection, gap ${gap}`)

    await editable.locator('img').first().click()
    await page.locator(imageBar).waitFor({ state: 'visible' })
    const [frame, image] = await Promise.all([page.locator(imageFrame).boundingBox(), editable.locator('img').first().boundingBox()])
    assert.ok(Math.abs(frame.x - image.x) < 1 && Math.abs(frame.y - image.y) < 1, 'selection frame covers the image')
    assert.deepEqual(errors, [])
  } finally {
    await page.close()
  }
})

test('on a 320px dark phone both bars fit the viewport and stay on top', async () => {
  const { page, editable, errors } = await openEditor('?theme=gf-dark', { width: 320, height: 640 })
  try {
    await selectText(editable.locator('p').first(), 2)
    await page.locator(selectionBar).waitFor({ state: 'visible' })
    await assertOnTop(page, selectionBar)
    await editable.locator('img').first().click()
    await page.locator(imageBar).waitFor({ state: 'visible' })
    await assertOnTop(page, imageBar)
    const surface = await page.locator(imageBar).evaluate((el) => getComputedStyle(el).backgroundColor)
    const editor = await editable.evaluate((el) => getComputedStyle(el.closest('.vditor')).backgroundColor)
    assert.equal(surface, editor, 'the bar uses the themed base surface')
    assert.deepEqual(errors, [])
  } finally {
    await page.close()
  }
})

test('in the fullscreen editor a click selects the image above the editor layer', async () => {
  const { page, editable, errors } = await openEditor()
  try {
    await page.evaluate(() => document.querySelector('.vditor-toolbar [data-type="fullscreen"]').dispatchEvent(new Event('click', { bubbles: true })))
    await page.waitForFunction(() => document.querySelector('.vditor')?.classList.contains('vditor--fullscreen'))
    await editable.locator('img').first().click()
    await page.locator(imageBar).waitFor({ state: 'visible' })
    await assertOnTop(page, imageBar)
    const handle = page.locator(`${imageFrame} [data-edge="right"]`)
    const box = await handle.boundingBox()
    assert.ok(await handle.evaluate((el, point) => el.contains(document.elementFromPoint(point.x, point.y)), { x: box.x + box.width / 2, y: box.y + box.height / 2 }), 'the resize handle is reachable')
    assert.equal(await page.locator('[role="dialog"][aria-modal="true"]').count(), 0, 'a single click never opens the viewer')
    assert.deepEqual(errors, [])
  } finally {
    await page.close()
  }
})

test('a partly scrolled-out image keeps its frame inside the visible editor area', async () => {
  const { page, editable } = await openEditor('?host=scroll')
  try {
    const cover = editable.locator('img').first()
    await cover.click()
    await page.locator(imageFrame).waitFor({ state: 'visible' })
    // Scroll the image's own scroll container until its top half is hidden.
    await page.evaluate(() => {
      const img = document.querySelector('.vditor-wysiwyg .vditor-reset img')
      let scroller = img.parentElement
      while (scroller && !(scroller.scrollHeight > scroller.clientHeight && getComputedStyle(scroller).overflowY !== 'visible')) scroller = scroller.parentElement
      scroller.dataset.testScroller = ''
      scroller.scrollTop += img.getBoundingClientRect().top - scroller.getBoundingClientRect().top + img.getBoundingClientRect().height / 2
    })
    const edge = () => page.evaluate(() => {
      const content = document.querySelector('.vditor-content').getBoundingClientRect()
      const scroller = document.querySelector('[data-test-scroller]').getBoundingClientRect()
      const toolbar = document.querySelector('.vditor-toolbar').getBoundingClientRect()
      let top = Math.max(content.top, scroller.top, 0)
      if (toolbar.top <= top + 1 && toolbar.bottom > top) top = toolbar.bottom
      return top
    })
    const visibleEdge = await edge()
    await page.waitForFunction((top) => {
      const frame = document.querySelector('[data-test="editor-image-frame"]')
      const clipTop = parseFloat(frame.style.clipPath.slice('inset('.length))
      return clipTop > 0 && Math.abs(frame.getBoundingClientRect().top + clipTop - top) < 2
    }, visibleEdge)
    const handle = await page.locator(`${imageFrame} [data-edge="right"]`).boundingBox()
    assert.ok(handle.y >= visibleEdge, `the handle moves into the visible part (${handle.y} >= ${visibleEdge})`)
  } finally {
    await page.close()
  }
})

test('dragging files over the editor shows a drop hint until they are dropped or leave', async () => {
  const { page, editable } = await openEditor()
  try {
    const state = () => page.evaluate(() => document.querySelector('.vditor-official').classList.contains('gf-drop-active'))
    const fire = (type) => editable.evaluate((el, eventType) => {
      const data = new DataTransfer()
      data.items.add(new File(['x'], 'a.png', { type: 'image/png' }))
      el.dispatchEvent(new DragEvent(eventType, { bubbles: true, cancelable: true, dataTransfer: data }))
    }, type)
    await fire('dragenter')
    assert.equal(await state(), true)
    assert.notEqual(await page.evaluate(() => getComputedStyle(document.querySelector('.vditor-content')).outlineColor), 'rgba(0, 0, 0, 0)')
    await fire('dragleave')
    assert.equal(await state(), false)
    await fire('dragenter')
    await fire('drop')
    assert.equal(await state(), false)
  } finally {
    await page.close()
  }
})

test('the image viewer zooms around the pointer and pans when zoomed', async () => {
  const { page, editable } = await openEditor()
  try {
    await editable.locator('img').first().dblclick()
    const viewer = page.locator('[role="dialog"][aria-modal="true"]')
    await viewer.waitFor({ state: 'visible' })
    const picture = viewer.locator('img').first()
    const box = await picture.boundingBox()
    await page.mouse.move(box.x + box.width * 0.75, box.y + box.height / 2)
    await page.mouse.wheel(0, -400)
    await page.waitForFunction(() => /scale\((?!1\))/.test(document.querySelector('[role="dialog"][aria-modal="true"] img').style.transform))
    assert.match(await viewer.innerText(), /\d{3}%/)
    const before = await picture.evaluate((img) => img.style.transform)
    await page.mouse.down()
    await page.mouse.move(box.x + box.width * 0.5, box.y + box.height / 2, { steps: 6 })
    await page.mouse.up()
    assert.notEqual(await picture.evaluate((img) => img.style.transform), before)
    assert.equal(await viewer.count(), 1, 'a pan never closes the viewer')
    await page.keyboard.press('Escape')
    await viewer.waitFor({ state: 'detached' })
  } finally {
    await page.close()
  }
})
