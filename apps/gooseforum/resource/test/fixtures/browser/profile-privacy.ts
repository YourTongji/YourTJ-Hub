import { h } from 'vue'
import { createRouter, createMemoryHistory } from 'vue-router'
import type { SettingsPageProps, UserBookmarkPayload, UserProfileProps } from '@gooseforum/client'
import { layout, memberProfile } from '../profile'
import { createBrowserApp } from './create-app'
import AppShell from '../../../src/site/components/AppShell.vue'
import UserPage from '../../../src/site/pages/UserPage.vue'
import SettingsPage from '../../../src/site/pages/SettingsPage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

// issue #1124: real components and CSS; the acceptance test serves the
// privacy API and picks the viewer (owner, other member, guest) by URL flag.
const params = new URLSearchParams(location.search)
const fixtureNickname = params.get('twemoji') === '1' ? '同济用户🐦‍🔥' : '同济用户'
document.documentElement.dataset.theme = params.get('theme') === 'dark' ? 'gf-dark' : 'gf-light'
await setLocale('zh')
const viewer = params.get('viewer') ?? 'owner'
if (viewer === 'guest') { layout.viewer.isAuthenticated = false; layout.viewer.id = 0 }
if (viewer === 'other') layout.viewer.id = 2048
layout.viewer.avatarUrl = '/assets/test/fixtures/browser/anonymous-admin-avatar-1.svg'

const own = viewer === 'owner'
const reason = viewer === 'guest' ? 'signin' : 'followers'
const profile: UserProfileProps = {
  ...memberProfile,
  user: { ...memberProfile.user, nickname: fixtureNickname, isSelf: own, avatarUrl: '/assets/test/fixtures/browser/anonymous-admin-avatar-0.svg' },
  isOwnProfile: own, canFollow: !own && viewer !== 'guest',
  section: 'activity', activityTab: 'likes',
  activityTabs: ['timeline', 'topics', 'likes', 'following', 'followers'].map(key => ({ key, url: `/u/1024/activity?tab=${key}`, active: key === 'likes' })),
  likes: own ? memberProfile.topics : [],
  visibility: {
    ...memberProfile.visibility,
    profileLikes: own ? { visible: true, audience: 'followers' } : { visible: false, reason },
  },
}
// #1624: reuse the real profile shell for mixed-bookmark browser acceptance.
// Only synthetic data is exposed to the test's intercepted pagination response.
if (params.get('screen') === 'bookmarks') {
  const firstType = params.get('first') === 'post' ? 'post' : 'topic'
  const topic: UserBookmarkPayload = {
    id: 7, type: 'topic', topicId: 100, url: '/p/100',
    title: '主题收藏：图书馆窗边的银杏与期末复习计划，这是用于检查窄屏长标题截断的合成内容',
    bookmarkedAt: '2026-11-06T09:00:00Z',
  }
  const post: UserBookmarkPayload = {
    id: 7, type: 'post', topicId: 200, postId: 555, postNo: 3, url: '/p/200?postNo=3',
    title: '回复收藏：校园生活讨论中的第三层回复，同号收藏不应覆盖另一种类型',
    excerpt: '这是一条合成回复摘要。检查楼层标记、两行摘要和收藏时间在浅色与深色主题中都清晰，加载后原先的收藏仍然保留。',
    bookmarkedAt: '2026-11-06T08:00:00Z',
  }
  const [first, next] = firstType === 'topic' ? [topic, post] : [post, topic]
  first.bookmarkedAt = '2026-11-06T09:00:00Z'
  next.bookmarkedAt = '2026-11-06T08:00:00Z'
  profile.section = params.get('section') === 'activity' ? 'activity' : 'bookmarks'
  profile.activityTab = 'bookmarks'
  profile.tabs = profile.tabs.map(tab => ({ ...tab, active: tab.key === profile.section }))
  profile.activityTabs = ['timeline', 'topics', 'likes', 'bookmarks'].map(key => ({
    key, url: `/u/1024/activity?tab=${key}`, active: key === 'bookmarks',
  }))
  profile.user = { ...profile.user, nickname: '收藏分页验收（合成数据）', profileCoverUrl: '', collectionCount: 2 }
  profile.bookmarks = [first]
  profile.pagination = { page: 1, nextPage: 2, hasNext: true, nextUrl: '/u/1024/bookmarks?cursor=identity' }
  Object.assign(window, { bookmarkAcceptancePage: {
    component: 'user.profile', version: '1.0', url: profile.pagination.nextUrl, layout,
    props: { ...profile, bookmarks: [first, next, next],
      pagination: { page: 2, nextPage: 0, hasNext: false, nextUrl: '' } },
  } })
}

const settings: SettingsPageProps = {
  user: {
    id: 1024, username: 'tongji_user', nickname: fixtureNickname, email: 'demo@example.test', locale: 'zh',
    avatarUrl: layout.viewer.avatarUrl, profileCoverUrl: '', bio: '', signature: '', websiteName: '', website: '',
    prestige: 0, createdAt: '2026-08-06T00:00:00Z', externalInformation: {}, wornBadgeCode: '', badges: [], wearableBadges: [], wornBadge: null,
  },
  googleOAuthReady: false, canSetPassword: false,
  stats: { topicCount: 3, replyCount: 2, followerCount: 5, followingCount: 9, likeReceivedCount: 18, likeGivenCount: 23, collectionCount: 3, createdAt: '2026-08-06T00:00:00Z' },
  tabs: ['profile', 'account', 'privacy', 'notifications', 'binding', 'security', 'content', 'deleted', 'general']
    .map(key => ({ key, url: `/settings?tab=${key}`, active: key === 'privacy' })),
}

const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:pathMatch(.*)*', component: { render: () => null } }] })
const screen = params.get('screen') ?? 'profile'
if (screen === 'bookmarks') {
  // The shell only renders its mobile publish button on the home route.
  // Match the profile entry being exercised rather than the memory default '/'.
  await router.push(profile.section === 'bookmarks' ? '/u/1024/bookmarks' : '/u/1024/activity?tab=bookmarks')
}
const app = createBrowserApp({ render: () => h(AppShell, { layout }, { default: () => screen === 'settings'
  ? h(SettingsPage, { layout, props: settings })
  : h(UserPage, { layout, props: profile }) }) }).use(i18n).use(router)
for (const name of ['code-copy', 'code-highlight', 'math-render', 'content-enhancements']) app.directive(name, {})
app.mount('#app')
