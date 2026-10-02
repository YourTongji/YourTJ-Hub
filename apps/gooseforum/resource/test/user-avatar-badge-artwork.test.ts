// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { mount } from '@vue/test-utils'
import UserAvatar from '../src/site/components/UserAvatar.vue'
import type { UserBadgePayload } from '@gooseforum/client'

function badge(iconUrl: string): UserBadgePayload {
  return {
    code: 'b', type: 'custom', grantMode: 'manual', name: 'b', description: '',
    iconType: 'asset', iconKey: '', iconUrl, color: 'blue', level: 'bronze',
    isEnabled: true, isWearable: true, sortOrder: 0,
    source: 'manual', reason: '', grantedAt: '2026-09-01T00:00:00Z',
  }
}

function chip(iconUrl: string) {
  const wrapper = mount(UserAvatar, { props: { src: '/static/pic/1.webp', alt: 'a', badge: badge(iconUrl) } })
  const span = wrapper.get('span.absolute')
  return { chip: span.classes(), icon: span.get('img').classes() }
}

// 系统徽章图形自带光学边距，图标区按比例占角标 88%；自定义徽章（如外链品牌 logo）
// 往往铺满画布，保持原来的 2px 内边距 + 填满，不能跟着放大到压住圆环。
describe('UserAvatar badge artwork framing', () => {
  test('system artwork fills 88% of the chip without padding', () => {
    for (const url of ['/static/badges/robot.svg?v=2', '/static/badges/robot.svg', '']) {
      const { chip: c, icon } = chip(url)
      expect(c).not.toContain('p-[2px]')
      expect(icon).toEqual(expect.arrayContaining(['h-[88%]', 'w-[88%]']))
    }
  })

  test('custom artwork keeps the 2px inset and fills the chip', () => {
    for (const url of ['https://thesvg.org/icons/deepseek/default.svg', '/file/img/2026/10/badge.png']) {
      const { chip: c, icon } = chip(url)
      expect(c).toContain('p-[2px]')
      expect(icon).toEqual(expect.arrayContaining(['h-full', 'w-full']))
    }
  })
})
