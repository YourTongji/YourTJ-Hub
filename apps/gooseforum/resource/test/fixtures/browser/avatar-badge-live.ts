import { createApp, h } from 'vue'
import UserAvatar from '../../../src/site/components/UserAvatar.vue'
import '../../../src/styles/resource.css'
import type { UserBadgePayload } from '@gooseforum/client'

// 全部系统徽章 × 四种头像尺寸，用真实 UserAvatar.vue 渲染，
// 供 test/badge-icon-optical-size.browser.mjs 测量（data-variant="after" 的单元格）。
// 与 app/service/badgeservice/definitions.go 一致
const BADGES: [code: string, color: string, level: string][] = [
  ['first_post', 'blue', 'bronze'], ['first_comment', 'teal', 'bronze'], ['first_like_given', 'rose', 'bronze'],
  ['first_follower', 'violet', 'bronze'], ['writer_10', 'sky', 'silver'], ['commenter_50', 'emerald', 'silver'],
  ['liked_10', 'amber', 'silver'], ['popular_100', 'orange', 'gold'], ['social_10', 'purple', 'silver'],
  ['early_member', 'cyan', 'special'], ['contributor', 'fuchsia', 'special'], ['moderator', 'emerald', 'special'],
  ['sponsor', 'yellow', 'special'], ['king', 'amber', 'special'], ['robot', 'slate', 'special'],
]

// 个人主页 h-24 / sm:h-28、用户卡片 h-14、帖子楼层 h-9 / sm:h-10、回复行 h-6
const SITES = [
  ['profile', 'large', 'h-24 w-24 sm:h-28 sm:w-28'],
  ['card', 'medium', 'h-14 w-14'],
  ['post', 'medium', 'h-9 w-9 sm:h-10 sm:w-10'],
  ['reply', 'medium', 'h-6 w-6'],
] as const

function payload(code: string, color: string, level: string): UserBadgePayload {
  return {
    code, type: 'system', grantMode: 'auto', name: code, description: '',
    // 与线上接口返回的地址一致；测试用 Playwright 路由把 /static/badges/ 映射到 static/badges/
    iconType: 'asset', iconKey: '', iconUrl: `/static/badges/${code.replace(/_/g, '-')}.svg?v=2`, color, level,
    isEnabled: true, isWearable: false, sortOrder: 0,
    source: 'auto', reason: '', grantedAt: '2026-09-01T00:00:00Z',
  }
}

createApp({
  render: () => h('div', { style: { padding: '24px' } }, SITES.map(([site, size, cls]) =>
    h('div', { style: { display: 'flex', flexWrap: 'wrap', gap: '24px', marginBottom: '24px' } }, BADGES.map(([code, color, level]) =>
      h(UserAvatar, {
        src: '/assets/static/pic/1.webp', alt: code, badge: payload(code, color, level), size,
        class: `${cls} rounded-full`, imgClass: 'rounded-full',
        'data-badge-code': code, 'data-variant': 'after', 'data-site': site,
      }))))),
}).mount('#app')
