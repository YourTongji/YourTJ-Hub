// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import { closePanel, openPanel } from '../src/runtime/use-wiki-search'
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
