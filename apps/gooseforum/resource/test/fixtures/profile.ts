import type { AnonymousProfileProps, LayoutPayload, TopicPayload, UserProfileProps } from '@gooseforum/client'
import card from '../../../../../packages/api-contract/fixtures/user-card-success.json'

export const persona = { kind: 'persona' as const, publicUid: 'a'.repeat(32), name: '躲进云里的猫', avatarUrl: `/a/${'a'.repeat(32)}/avatar.svg`, profileUrl: `/a/${'a'.repeat(32)}` }
export const topics: TopicPayload[] = ['分享一件最近让你开心的小事', '图书馆窗边的银杏开始变黄了', '下雨天适合抱着热茶看书'].map((title, index) => ({
  id: index + 1, title, description: '记录校园里的一点日常。', url: `/p/${index + 1}`,
  author: { kind: 'persona', id: 0, publicUid: persona.publicUid, username: persona.name, avatarUrl: persona.avatarUrl, profileUrl: persona.profileUrl },
  participants: [], categories: [{ id: 1, name: '校园生活', color: '#339f84', url: '/c/1' }], replyCount: index + 2, viewCount: 42 + index * 15,
  likeCount: 3, pinWeight: 0, processStatus: 0, activityText: '', lastUpdateTime: '2026-10-07T09:00:00Z', contentType: 0,
}))
export const identityState = {
  persona, nameSelectedAt: '2026-10-07T00:00:00Z', nameChangeAvailableAt: '2027-10-07T00:00:00Z',
  disabled: false, governanceDisabled: false, day: '2026-10-07', remaining: 10, resetsAt: '2026-10-07T16:00:00Z', batches: [], lexiconVersion: 'phrase6-v1',
}
export const layout: LayoutPayload = {
  site: { name: 'yourtj', description: '', logo: '', favicon: '', brandType: 'text', brandText: 'yourtj', brandImage: '' },
  viewer: { id: 1024, username: 'tongji_user', email: '', avatarUrl: '/static/pic/3.webp', isAuthenticated: true, canAccessAdmin: false, isModerator: false, requiresEmailVerification: false, adminPermissions: [] },
  sidebar: { categories: [], activeKey: 'user' }, footer: { links: [], primary: [] },
  unread: { notifications: false, messages: false }, posting: { maxTitleLength: 100 },
  theme: { enabled: false, current: 'gf-light', themeColor: '' }, umamiEnabled: false,
}
export const memberProfile: UserProfileProps = {
  user: { ...card.result, isSelf: true }, section: 'summary', activityTab: 'timeline',
  tabs: ['summary','activity','bookmarks','badges'].map(key => ({ key, url: `/u/1024/${key}`, active: key === 'summary' })),
  activityTabs: [], pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' },
  badges: [], topics: topics.map(topic => ({ ...topic, author: { id: card.result.userId, username: card.result.username, nickname: card.result.nickname, avatarUrl: card.result.avatarUrl } })), activities: [], likes: [], bookmarks: [], following: [], followers: [],
  isOwnProfile: true, canMessage: false, canFollow: false, messageUrl: '', settingsUrl: '/settings',
}
export const anonymousProfile: AnonymousProfileProps = {
  persona, topics, topicCount: topics.length, replyCount: 2, page: 1, hasNext: false,
  replies: [{ id: 1, url: '/p/1/2', excerpt: '路过图书馆时，发现窗边的银杏已经开始变黄了。' },
    { id: 2, url: '/p/2/3', excerpt: '抱着一杯热茶，看了一下午的雨。' }],
}
