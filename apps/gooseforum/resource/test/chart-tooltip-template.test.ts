// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import { describe, expect, test } from 'vitest'
import { defineComponent, h } from 'vue'
import { ChartTooltipContent, componentToString } from '../src/admin/components/ui/chart'
import { twemojiVuePlugin } from '../src/runtime/twemoji'

describe('chart crosshair tooltip template', () => {
  // 模板文本插值被编译为全局 TwemojiText；离屏渲染若丢失应用上下文，悬浮卡片会只剩空的 <twemojitext>。
  test('renders label, series and value text through the app context', () => {
    let template: ((data: Record<string, number>, x: number | Date) => string) | undefined
    mount(defineComponent({
      setup() {
        template = componentToString({ visits: { label: '访问' } }, ChartTooltipContent, { labelFormatter: () => 'Oct 8, 2026' })
        return () => h('div')
      },
    }), { global: { plugins: [twemojiVuePlugin] } })

    const card = document.createElement('div')
    card.innerHTML = template!({ visits: 12 }, new Date('2026-10-08T00:00:00Z'))

    expect(card.querySelector('twemojitext')).toBeNull()
    expect(card.textContent?.replace(/\s+/g, ' ')).toContain('Oct 8, 2026')
    expect(card.textContent).toContain('访问')
    expect(card.textContent).toContain('12')
  })
})
