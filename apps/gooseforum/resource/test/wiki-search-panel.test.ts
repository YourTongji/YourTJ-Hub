// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import { closePanel, consumeWikiJumpState, openPanel, type WikiSearchItem } from '../src/runtime/use-wiki-search'
import WikiSearchPanel from '../src/site/components/WikiSearchPanel.vue'

let wrapper: VueWrapper | null = null

beforeEach(() => {
  i18n.global.locale.value = 'zh'
})

afterEach(async () => {
  wrapper?.unmount()
  wrapper = null
  closePanel()
  await flushPromises()
  document.body.innerHTML = ''
  sessionStorage.clear()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

function searchItem(namespace: string, page: string, anchors: string[]): WikiSearchItem {
  return {
    namespace, path: `${namespace}/${page}`, title: page, anchors,
    titleHit: true, snippet: page, score: 1, hitType: 'title',
  }
}

const interleavedResults = [
  searchItem('A', 'first', ['s-1']),
  searchItem('B', 'first', ['s-2']),
  searchItem('A', '选课 指南', ['s-3', 's-4']),
]

describe('WikiSearchPanel grouped result navigation', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    vi.spyOn(window.location, 'assign').mockImplementation(() => {})
  })

  async function search(items = interleavedResults) {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      json: async () => ({ result: { query: '选课', total: items.length, items } }),
    }))
    mountPanel()
    await flushPromises()
    const input = new DOMWrapper(document.querySelector<HTMLInputElement>('[role="combobox"]')!)
    await input.setValue('选课')
    await vi.advanceTimersByTimeAsync(120)
    await flushPromises()
    return input
  }

  function options() {
    return Array.from(document.querySelectorAll<HTMLButtonElement>('[role="option"]'))
  }

  function expectActive(input: DOMWrapper<HTMLInputElement>, index: number) {
    const selected = options().filter((option) => option.getAttribute('aria-selected') === 'true')
    expect(selected).toEqual([options()[index]])
    expect(input.attributes('aria-activedescendant')).toBe(selected[0]!.id)
  }

  test('Enter opens the highlighted grouped result and preserves its matching anchors', async () => {
    const input = await search()
    expect(options().map((option) => option.textContent)).toEqual([
      expect.stringContaining('A/first'),
      expect.stringContaining('A/选课 指南'),
      expect.stringContaining('B/first'),
    ])
    await input.trigger('keydown', { key: 'ArrowDown' })
    expectActive(input, 1)
    await input.trigger('keydown', { key: 'Enter' })
    expect(window.location.assign).toHaveBeenCalledWith('/wiki/A/%E9%80%89%E8%AF%BE%20%E6%8C%87%E5%8D%97#s-3')
    expect(consumeWikiJumpState()).toEqual({ query: '选课', anchors: ['s-3', 's-4'] })
  })

  test('up and down wrap in visible order across namespace boundaries', async () => {
    const input = await search()
    expectActive(input, 0)
    await input.trigger('keydown', { key: 'ArrowUp' })
    expectActive(input, 2)
    await input.trigger('keydown', { key: 'ArrowDown' })
    expectActive(input, 0)
    await input.trigger('keydown', { key: 'ArrowDown' })
    expectActive(input, 1)
    await input.trigger('keydown', { key: 'ArrowDown' })
    expectActive(input, 2)
    await input.trigger('keydown', { key: 'Enter' })
    expect(window.location.assign).toHaveBeenCalledWith('/wiki/B/first#s-2')
  })

  test.each(['mouseenter then Enter', 'click'])('%s opens the displayed item', async (action) => {
    const input = await search()
    const second = new DOMWrapper(options()[1]!)
    if (action === 'click') {
      await second.trigger('click')
    } else {
      await second.trigger('mouseenter')
      expectActive(input, 1)
      await input.trigger('keydown', { key: 'Enter' })
    }
    expect(window.location.assign).toHaveBeenCalledWith('/wiki/A/%E9%80%89%E8%AF%BE%20%E6%8C%87%E5%8D%97#s-3')
    expect(consumeWikiJumpState()).toEqual({ query: '选课', anchors: ['s-3', 's-4'] })
  })

  test('single-namespace results without anchors retain their navigation behavior', async () => {
    const input = await search([searchItem('A', 'first', []), searchItem('A', 'second', [])])
    await input.trigger('keydown', { key: 'ArrowDown' })
    expectActive(input, 1)
    await input.trigger('keydown', { key: 'Enter' })
    expect(window.location.assign).toHaveBeenCalledWith('/wiki/A/second')
    expect(consumeWikiJumpState()).toBeNull()
  })

  test('empty results ignore navigation keys', async () => {
    const input = await search([])
    for (const key of ['ArrowDown', 'ArrowUp', 'Enter']) await input.trigger('keydown', { key })
    expect(options()).toHaveLength(0)
    expect(input.attributes('aria-activedescendant')).toBeUndefined()
    expect(window.location.assign).not.toHaveBeenCalled()
  })

  test('clearing the query removes the previous selection', async () => {
    const input = await search()
    await input.trigger('keydown', { key: 'ArrowDown' })
    await input.setValue('')
    await input.trigger('keydown', { key: 'Enter' })
    expect(options()).toHaveLength(0)
    expect(input.attributes('aria-activedescendant')).toBeUndefined()
    expect(window.location.assign).not.toHaveBeenCalled()
  })

  test('closing and reopening the panel removes the previous selection', async () => {
    const input = await search()
    await input.trigger('keydown', { key: 'ArrowDown' })
    expectActive(input, 1)
    closePanel()
    await flushPromises()
    openPanel()
    await flushPromises()
    const reopened = new DOMWrapper(document.querySelector<HTMLInputElement>('[role="combobox"]')!)
    expect(reopened.element.value).toBe('')
    expect(reopened.attributes('aria-activedescendant')).toBeUndefined()
    await reopened.trigger('keydown', { key: 'Enter' })
    expect(window.location.assign).not.toHaveBeenCalled()
  })
})

function mountPanel() {
  openPanel()
  wrapper = mount(WikiSearchPanel, {
    global: { plugins: [i18n] },
    attachTo: document.body,
  })
  return wrapper
}

describe('WikiSearchPanel Dialog 可访问性', () => {
  test('aria-describedby 指向存在的描述元素（P2-5 建议项）', async () => {
    mountPanel()
    await flushPromises()

    const dialog = document.querySelector('[role="dialog"]')
    expect(dialog).not.toBeNull()
    const describedBy = dialog?.getAttribute('aria-describedby')
    expect(describedBy).toBeTruthy()
    // reka-ui 自动把 aria-describedby 指向 DialogDescription 的 id
    expect(document.getElementById(describedBy!)).not.toBeNull()
    expect(document.getElementById(describedBy!)?.textContent).toContain('搜索')
  })

  test('点击遮罩（Content 自身区域）关闭面板（P1-3 建议项）', async () => {
    mountPanel()
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')).not.toBeNull()

    // Content 是 fixed inset-0，卡片外点击命中 Content 自身 → @pointerdown.self 关闭
    const dialog = document.querySelector('[role="dialog"]')!
    dialog.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    await flushPromises()

    expect(document.querySelector('[role="dialog"]')).toBeNull()
  })

  test('点击卡片内部不关闭面板', async () => {
    mountPanel()
    await flushPromises()

    // 内容经 Teleport 到 body，须用 document 查询；卡片内冒泡到 Content 时
    // target 非 Content 自身，@pointerdown.self 不触发，面板保持打开。
    const input = document.querySelector<HTMLInputElement>('input[role="combobox"]')
    expect(input).not.toBeNull()
    input!.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    await flushPromises()

    expect(document.querySelector('[role="dialog"]')).not.toBeNull()
  })
})
