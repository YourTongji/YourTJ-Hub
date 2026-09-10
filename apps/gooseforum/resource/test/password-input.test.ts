// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import PasswordInput from '../src/site/components/PasswordInput.vue'

function mountInput() {
  return mount(PasswordInput, {
    props: { placeholder: '密码', label: '密码' },
    global: { plugins: [i18n] },
    attachTo: document.body,
  })
}

describe('PasswordInput 明文/密文切换（issue #375）', () => {
  test('默认密文输入，切换按钮 aria 状态为“显示密码”', () => {
    const wrapper = mountInput()
    const input = wrapper.find('input')
    expect(input.attributes('type')).toBe('password')
    const toggle = wrapper.find('button')
    expect(toggle.attributes('aria-label')).toBe(i18n.global.t('auth.showPassword'))
    expect(toggle.attributes('aria-pressed')).toBe('false')
  })

  test('点击切换按钮后输入框变为明文，图标状态同步为“隐藏密码”', async () => {
    const wrapper = mountInput()
    const toggle = wrapper.find('button')
    await toggle.trigger('click')
    expect(wrapper.find('input').attributes('type')).toBe('text')
    expect(toggle.attributes('aria-label')).toBe(i18n.global.t('auth.hidePassword'))
    expect(toggle.attributes('aria-pressed')).toBe('true')
    // 再点一次恢复密文
    await toggle.trigger('click')
    expect(wrapper.find('input').attributes('type')).toBe('password')
  })

  test('v-model 双向绑定输入值', async () => {
    const wrapper = mountInput()
    await wrapper.find('input').setValue('secret')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['secret'])
  })
})
