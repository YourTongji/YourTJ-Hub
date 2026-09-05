// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import LinksPage from '../src/site/pages/LinksPage.vue'
import type { LinksPageProps } from '@gooseforum/client'

function makeProps(overrides: Partial<LinksPageProps> = {}): LinksPageProps {
  return {
    groups: [
      {
        name: '社区',
        emoji: '👥',
        color: '',
        links: [
          {
            name: 'GooseForum',
            desc: '简单的社区构建软件，YourTJHub框架基座来源',
            url: 'https://gooseforum.online',
            logoUrl: '/static/pic/default-avatar.webp',
          },
          {
            name: '无描述链接',
            desc: '',
            url: 'https://example.com/very-long-url-path',
            logoUrl: '',
          },
        ],
      },
    ],
    totalCount: 2,
    ...overrides,
  }
}

function mountPage(props: LinksPageProps) {
  return mount(LinksPage, {
    props: { layout: {}, props },
    global: { plugins: [i18n] },
  })
}

// 回归测试（issue #471）：链接卡片描述被 truncate 单行截断后，
// hover/键盘 focus 必须能通过 gf-tooltip 看到完整描述。
describe('LinksPage 链接卡片 hover card', () => {
  test('有描述的卡片渲染完整描述 tooltip，hover/focus 时显示', () => {
    const wrapper = mountPage(makeProps())
    const card = wrapper.findAll('a').find((a) => a.text().includes('GooseForum'))
    expect(card).toBeTruthy()

    const tooltip = card!.find('.gf-tooltip')
    expect(tooltip.exists()).toBe(true)
    expect(tooltip.text()).toBe('简单的社区构建软件，YourTJHub框架基座来源')

    const classes = tooltip.classes()
    expect(classes).toContain('group-hover:opacity-100')
    expect(classes).toContain('group-focus-visible:opacity-100')
    // 不干扰外链点击与读屏（完整文本本就在卡片 DOM 中）
    expect(classes).toContain('pointer-events-none')
    expect(tooltip.attributes('aria-hidden')).toBe('true')
  })

  test('tooltip 定位锚定卡片（group relative），深浅色由 gf-tooltip 主题变量承担', () => {
    const wrapper = mountPage(makeProps())
    const card = wrapper.findAll('a').find((a) => a.text().includes('GooseForum'))
    expect(card!.classes()).toContain('relative')
    expect(card!.find('.gf-tooltip').classes()).toContain('gf-tooltip')
  })

  test('无描述（仅 URL）的卡片不渲染重复 tooltip', () => {
    const wrapper = mountPage(makeProps())
    const card = wrapper.findAll('a').find((a) => a.text().includes('无描述链接'))
    expect(card).toBeTruthy()
    expect(card!.find('.gf-tooltip').exists()).toBe(false)
  })
})
