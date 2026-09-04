// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import PublishMenu from '../src/site/components/PublishMenu.vue'
import { i18n } from '../src/runtime/i18n'
import { useQuickPublish } from '../src/site/composables/useQuickPublish'
import { useShellState, resetShellState } from '../src/runtime/shell-state'

describe('PublishMenu 组件', () => {
  test('渲染桌面端 (navbar) 模式的原版 gf-button 触发按钮', () => {
    const wrapper = mount(PublishMenu, {
      props: { variant: 'navbar' },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    const trigger = wrapper.find('button')
    expect(trigger.exists()).toBe(true)
    expect(trigger.classes()).toContain('gf-button')
    expect(trigger.classes()).toContain('gf-button-primary')
    expect(trigger.text()).toContain(i18n.global.t('shell.publish'))
    expect(trigger.attributes('aria-haspopup')).toBe('true')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    wrapper.unmount()
  })

  test('渲染移动端 (fab) 模式的悬浮按钮', () => {
    const wrapper = mount(PublishMenu, {
      props: { variant: 'fab' },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    const trigger = wrapper.find('button')
    expect(trigger.exists()).toBe(true)
    expect(trigger.classes()).toContain('fixed')
    expect(trigger.classes()).toContain('h-14')
    expect(trigger.classes()).toContain('w-14')
    wrapper.unmount()
  })

  test('点击触发按钮展开菜单，点击非文章类型触发弹层发布', async () => {
    const { quickPublishOpen, quickPublishType } = useQuickPublish()
    quickPublishOpen.value = false

    const wrapper = mount(PublishMenu, {
      props: { variant: 'navbar' },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    const trigger = wrapper.get('button')
    await trigger.trigger('click')
    await flushPromises()

    expect(trigger.attributes('aria-expanded')).toBe('true')

    // 检查浮层内 3 个发布类型链接（发瞬间、提问题、写文章）
    const links = document.body.querySelectorAll<HTMLAnchorElement>('a[role="menuitem"]')
    expect(links.length).toBe(3)

    // 点击「发瞬间」（type=2），应触发弹层发布
    const thoughtLink = Array.from(links).find((l) => l.getAttribute('href')?.includes('thought'))
    expect(thoughtLink).toBeDefined()
    thoughtLink?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    await flushPromises()

    expect(quickPublishOpen.value).toBe(true)
    expect(quickPublishType.value).toBe(2)

    // 点击「提问题」（type=1），应触发弹层发布
    const questionLink = Array.from(links).find((l) => l.getAttribute('href')?.includes('question'))
    expect(questionLink).toBeDefined()
    questionLink?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    await flushPromises()

    expect(quickPublishOpen.value).toBe(true)
    expect(quickPublishType.value).toBe(1)

    wrapper.unmount()
  })

  test('通过 shellState.isTopicPage 标记帖子详情页状态以控制移动端 FAB 显示', () => {
    const shellState = useShellState()
    expect(shellState.isTopicPage).toBe(false)

    shellState.isTopicPage = true
    expect(shellState.isTopicPage).toBe(true)

    resetShellState()
    expect(shellState.isTopicPage).toBe(false)
  })
})
