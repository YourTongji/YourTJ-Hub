// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test } from 'vitest'
import { mount } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import GlobalFlash from '../src/site/components/GlobalFlash.vue'
import { dismiss, useFlashMessages } from '../src/runtime/flash-message'

describe('GlobalFlash lightweight motion and DOM cleanup', () => {
  beforeEach(async () => {
    await setLocale('zh')
  })

  afterEach(() => {
    const { messages } = useFlashMessages()
    for (const m of messages.value) {
      dismiss(m.id)
    }
    document.body.innerHTML = ''
  })

  test('renders toast banner and dismisses cleanly without particle DOM clutter', async () => {
    const wrapper = mount(GlobalFlash, {
      global: { plugins: [i18n] },
      attachTo: document.body,
    })

    const { push } = useFlashMessages()
    push('测试通知', 'success')
    await wrapper.vm.$nextTick()

    const banner = wrapper.find('.gf-flash-banner')
    expect(banner.exists()).toBe(true)
    expect(banner.text()).toContain('测试通知')

    // Click close
    const closeBtn = wrapper.find('.gf-flash-banner__close')
    expect(closeBtn.exists()).toBe(true)
    await closeBtn.trigger('click')
    await wrapper.vm.$nextTick()

    // Ensure no particle elements are created on document.body
    expect(document.querySelectorAll('.gf-flash-dissolve-particle')).toHaveLength(0)
    wrapper.unmount()
  })
})
