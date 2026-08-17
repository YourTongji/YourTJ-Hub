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
      attrs: { class: 'mt-1 w-44 shrink-0' },
      attachTo: document.body,
    })
    const trigger = wrapper.get('[role="combobox"]')
    expect(trigger.classes()).toContain('mt-1')
    expect(trigger.classes()).toContain('w-44')
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
