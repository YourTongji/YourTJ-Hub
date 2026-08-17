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
