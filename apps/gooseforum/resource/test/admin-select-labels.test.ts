// @vitest-environment happy-dom
import { afterEach, expect, test } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import AdminSelect from '../src/admin/components/AdminSelect.vue'
let wrapper: VueWrapper | undefined
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = '' })
test('closed select updates the selected label when options are translated', async () => {
  wrapper = mount(AdminSelect, { attachTo: document.body, props: { modelValue: '', options: [{ value: '', label: '全部来源' }] } })
  await flushPromises()
  expect(wrapper.get('[role="combobox"]').text()).toBe('全部来源')
  await wrapper.setProps({ options: [{ value: '', label: 'All sources' }] })
  await flushPromises()
  expect(wrapper.get('[role="combobox"]').text()).toBe('All sources')
  expect(wrapper.emitted('update:modelValue')).toBeUndefined()
})
test('numeric selection and an empty placeholder remain distinct', async () => {
  wrapper = mount(AdminSelect, { attachTo: document.body, props: { modelValue: undefined, options: [], placeholder: 'Choose' } })
  await flushPromises()
  expect(wrapper.get('[role="combobox"]').text()).toBe('Choose')
  await wrapper.setProps({ modelValue: 42, options: [{ value: 42, label: 'Forty-two' }] })
  await flushPromises()
  expect(wrapper.get('[role="combobox"]').text()).toBe('Forty-two')
})
