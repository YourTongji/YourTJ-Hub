// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SiteSelect from '../src/site/components/SiteSelect.vue'

const options = [
  { value: 'zh', label: '中文' },
  { value: 'en', label: 'English' },
]

// reka-ui SelectRoot 是 renderless 组件（inheritAttrs: false），调用方 class
// 必须经 $attrs 透传到 SelectTrigger（issue: SettingsPage 语言选择/字号预设回归）。
describe('SiteSelect attrs 透传', () => {
  test('调用方 class 完整落到 trigger 上', () => {
    const wrapper = mount(SiteSelect, {
      props: { modelValue: 'zh', options },
      attrs: { class: 'mt-1 shrink-0' },
      attachTo: document.body,
    })
    const trigger = wrapper.get('[role="combobox"]')
    expect(trigger.classes()).toContain('mt-1')
    expect(trigger.classes()).toContain('shrink-0')
    wrapper.unmount()
  })

  test('无调用方 attrs 时不影响默认样式', () => {
    const wrapper = mount(SiteSelect, {
      props: { modelValue: 'zh', options },
      attachTo: document.body,
    })
    const trigger = wrapper.get('[role="combobox"]')
    expect(trigger.classes()).toContain('gf-input')
    expect(trigger.classes()).toContain('w-full')
    wrapper.unmount()
  })
})
// P1-1（review #320 第二轮）：$attrs.class 与静态 w-full 同时传给 trigger 时，
// Tailwind 产物顺序会让 w-full 覆盖调用方 w-44。twMerge 必须让调用方宽度胜出。
describe('SiteSelect twMerge class 合并', () => {
  test('调用方 w-44 覆盖默认 w-full（SettingsPage 字号预设）', () => {
    const wrapper = mount(SiteSelect, {
      props: { modelValue: 'zh', options },
      attrs: { class: 'w-44 shrink-0' },
      attachTo: document.body,
    })
    const trigger = wrapper.get('[role="combobox"]')
    const cls = trigger.classes()
    expect(cls).toContain('w-44')
    // twMerge 把冲突的 w-full 移除，避免 CSS 产物顺序导致 width:100% 覆盖
    expect(cls).not.toContain('w-full')
    expect(cls).toContain('shrink-0')
    wrapper.unmount()
  })

  test('调用方 mt-1 与默认类共存（SettingsPage 语言选择）', () => {
    const wrapper = mount(SiteSelect, {
      props: { modelValue: 'zh', options },
      attrs: { class: 'mt-1' },
      attachTo: document.body,
    })
    const trigger = wrapper.get('[role="combobox"]')
    expect(trigger.classes()).toContain('mt-1')
    expect(trigger.classes()).toContain('gf-input')
    wrapper.unmount()
  })

  // P2-1（review #320 第三轮）：$attrs.class 可能来自 :class 对象/数组形式，
  // 必须先经 clsx 规范化再交 twMerge，否则对象内容被忽略、w-full 仍生效。
  test('调用方对象形式 class（:class="{ \'w-44\': true }"）生效且覆盖 w-full', () => {
    const wrapper = mount(SiteSelect, {
      props: { modelValue: 'zh', options },
      attrs: { class: { 'w-44': true, shrink: false } },
      attachTo: document.body,
    })
    const trigger = wrapper.get('[role="combobox"]')
    const cls = trigger.classes()
    expect(cls).toContain('w-44')
    expect(cls).not.toContain('w-full')
    // 值为 false 的键不渲染
    expect(cls).not.toContain('shrink')
    wrapper.unmount()
  })

  test('调用方数组形式 class（:class="[\'w-44\', \'mt-1\']"）生效', () => {
    const wrapper = mount(SiteSelect, {
      props: { modelValue: 'zh', options },
      attrs: { class: ['w-44', 'mt-1'] },
      attachTo: document.body,
    })
    const trigger = wrapper.get('[role="combobox"]')
    expect(trigger.classes()).toContain('w-44')
    expect(trigger.classes()).toContain('mt-1')
    expect(trigger.classes()).not.toContain('w-full')
    wrapper.unmount()
  })
})

// SelectContent teleport 到 body，与 site 弹窗（z-[2000]/z-[2100]）同层；
// 列表必须高于弹窗层级，否则 ScheduleCoursePicker Tab3 下拉不可见。
// reka-ui SelectTrigger 用 pointerdown 打开（SelectTrigger.js handlePointerOpen）。
describe('SiteSelect 下拉层级', () => {
  test('打开后列表 z-index 为 z-[2100]（高于弹窗 z-[2000]）', async () => {
    const wrapper = mount(SiteSelect, {
      props: { modelValue: 'zh', options },
      attachTo: document.body,
    })
    await wrapper.get('[role="combobox"]').trigger('pointerdown', { button: 0, pageX: 10, pageY: 10 })
    await flushPromises()
    const listbox = document.querySelector('[role="listbox"]')
    expect(listbox).not.toBeNull()
    expect(listbox?.classList.contains('z-[2100]')).toBe(true)
    wrapper.unmount()
  })
})

// P1（review #320 第四轮）：reka-ui@2.9.8 SelectContentImpl 默认 bodyLock=true
// 且 disableOutsidePointerEvents=true，会把普通下拉变成页面级模态层
// （body.pointerEvents=none + overflow 锁定）。SiteSelect 传
// :body-lock="false" 与 :disable-outside-pointer-events="false"，
// 打开下拉后 body 必须保持可交互。
describe('SiteSelect 非模态行为', () => {
  test('打开下拉后 body 不被锁定（pointerEvents/overflow 不变）', async () => {
    const wrapper = mount(SiteSelect, {
      props: { modelValue: 'zh', options },
      attachTo: document.body,
    })
    await wrapper.get('[role="combobox"]').trigger('pointerdown', { button: 0, pageX: 10, pageY: 10 })
    await flushPromises()
    expect(document.querySelector('[role="listbox"]')).not.toBeNull()
    // 下拉打开时页面不应被模态锁定
    expect(document.body.style.pointerEvents).toBe('')
    expect(document.body.style.overflow).toBe('')
    wrapper.unmount()
  })
})

// P1-2（review #320 第二轮）：reka-ui@2.9.8 SelectContentImpl 对 Tab 无条件
// preventDefault 且不关闭 Select，用户会被困在列表中。补 Tab 关闭 + 焦点回 trigger。
describe('SiteSelect Tab 键盘行为', () => {
  async function openSelect(wrapper: ReturnType<typeof mount>) {
    await wrapper.get('[role="combobox"]').trigger('pointerdown', { button: 0, pageX: 10, pageY: 10 })
    await flushPromises()
    expect(document.querySelector('[role="listbox"]')).not.toBeNull()
  }

  test('在列表中按 Tab 关闭列表并把焦点移回 trigger', async () => {
    const wrapper = mount(SiteSelect, {
      props: { modelValue: 'zh', options },
      attachTo: document.body,
    })
    const trigger = wrapper.get('[role="combobox"]')
    await openSelect(wrapper)

    const listbox = document.querySelector('[role="listbox"]')!
    listbox.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true }))
    await flushPromises()

    expect(document.querySelector('[role="listbox"]')).toBeNull()
    await new Promise((r) => setTimeout(r, 0)) // nextTick 聚焦
    expect(document.activeElement).toBe(trigger.element)
    wrapper.unmount()
  })

  test('按 Shift+Tab 同样关闭列表', async () => {
    const wrapper = mount(SiteSelect, {
      props: { modelValue: 'zh', options },
      attachTo: document.body,
    })
    await openSelect(wrapper)

    const listbox = document.querySelector('[role="listbox"]')!
    listbox.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true }))
    await flushPromises()

    expect(document.querySelector('[role="listbox"]')).toBeNull()
    wrapper.unmount()
  })
})

// 排课器的专业、校区和开课院系可包含较多选项。可搜索模式应在当前下拉内
// 提供本地标签筛选，且不改变普通 SiteSelect 的默认交互。
describe('SiteSelect 可搜索模式', () => {
  async function openSearchableSelect(wrapper: ReturnType<typeof mount>) {
    await wrapper.get('[role="combobox"]').trigger('pointerdown', { button: 0, pageX: 10, pageY: 10 })
    await flushPromises()
    const input = document.querySelector<HTMLInputElement>('[data-testid="site-select-search-input"]')
    expect(input).not.toBeNull()
    return input!
  }

  test('按标签关键字过滤选项，且匹配不区分大小写', async () => {
    const wrapper = mount(SiteSelect, {
      props: {
        modelValue: '',
        options,
        searchable: true,
        searchPlaceholder: '搜索选项',
        emptyText: '没有匹配项',
      },
      attachTo: document.body,
    })

    const input = await openSearchableSelect(wrapper)
    expect(input.placeholder).toBe('搜索选项')
    input.value = 'EN'
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await flushPromises()

    const visibleOptions = [...document.querySelectorAll('[role="option"]')].map((option) => option.textContent)
    expect(visibleOptions).toEqual([expect.stringContaining('English')])
    expect(document.body.textContent).not.toContain('中文')
    wrapper.unmount()
  })

  test('输入关键词时保留输入焦点，不触发 Select 的类型导航', async () => {
    const wrapper = mount(SiteSelect, {
      props: { modelValue: '', options, searchable: true },
      attachTo: document.body,
    })

    const input = await openSearchableSelect(wrapper)
    input.focus()
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'E', bubbles: true }))
    await flushPromises()

    expect(document.activeElement).toBe(input)
    wrapper.unmount()
  })

  test('无匹配时显示调用方提供的空状态', async () => {
    const wrapper = mount(SiteSelect, {
      props: {
        modelValue: '',
        options,
        searchable: true,
        emptyText: '没有匹配项',
      },
      attachTo: document.body,
    })

    const input = await openSearchableSelect(wrapper)
    input.value = '不存在'
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await flushPromises()

    expect(document.querySelectorAll('[role="option"]')).toHaveLength(0)
    expect(document.querySelector('[data-testid="site-select-empty"]')?.textContent).toContain('没有匹配项')
    wrapper.unmount()
  })

  test('过滤后仍可用键盘选择选项', async () => {
    const wrapper = mount(SiteSelect, {
      props: {
        modelValue: '',
        options,
        searchable: true,
      },
      attachTo: document.body,
    })

    const input = await openSearchableSelect(wrapper)
    input.value = 'english'
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await flushPromises()

    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))
    await flushPromises()
    expect(document.activeElement?.textContent).toContain('English')

    document.activeElement?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await flushPromises()
    expect(wrapper.emitted('update:modelValue')).toEqual([['en']])
    wrapper.unmount()
  })
})

// 排课器学期选择：学期名可能很长（如「2026-2027 学年第一学期」），
// SelectValue 根 span 是 trigger(flex) 的子项，默认 min-width:auto 拒绝收缩，
// 长文本会把 trigger 撑破溢出；内层文本 span 必须是 block，truncate 才生效。
describe('SiteSelect 长文本溢出', () => {
  test('长标签下 SelectValue 容器可收缩(min-w-0)且文本 span 为 block', () => {
    const longLabel = '2026-2027 学年第一学期（同济大学一系统排课数据）'
    const wrapper = mount(SiteSelect, {
      props: { modelValue: '2026-1', options: [{ value: '2026-1', label: longLabel }] },
      attachTo: document.body,
    })
    const trigger = wrapper.get('[role="combobox"]')
    const labelSpan = trigger.element.querySelector('span.truncate')
    expect(labelSpan).not.toBeNull()
    // 文本 span 的父级 = SelectValue 根 span，必须是可收缩的 flex 子项
    expect(labelSpan!.parentElement!.classList.contains('min-w-0')).toBe(true)
    // inline 元素上 text-overflow: ellipsis 不生效，文本 span 必须为 block
    expect(labelSpan!.classList.contains('block')).toBe(true)
    wrapper.unmount()
  })
})

describe('SiteSelect clearable 模式', () => {
  test('启用 clearable 且有值时渲染清除按钮，点击触发清空', async () => {
    const wrapper = mount(SiteSelect, {
      props: {
        modelValue: '2026-1',
        options: [{ value: '2026-1', label: '2026学年' }],
        clearable: true,
        clearLabel: '清除',
      },
      attachTo: document.body,
    })
    const clearBtn = wrapper.find('button[aria-label="清除"]')
    expect(clearBtn.exists()).toBe(true)
    await clearBtn.trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([''])
    wrapper.unmount()
  })

  test('启用 clearable 但无值时不渲染清除按钮', () => {
    const wrapper = mount(SiteSelect, {
      props: {
        modelValue: '',
        options: [{ value: '2026-1', label: '2026学年' }],
        clearable: true,
        clearLabel: '清除',
      },
      attachTo: document.body,
    })
    const clearBtn = wrapper.find('button[aria-label="清除"]')
    expect(clearBtn.exists()).toBe(false)
    wrapper.unmount()
  })
})
