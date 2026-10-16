import { h } from 'vue'
import { createRouter, createMemoryHistory } from 'vue-router'
import type { SettingsPageProps, UserProfileProps } from '@gooseforum/client'
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
const app = createBrowserApp({ render: () => h(AppShell, { layout }, { default: () => screen === 'settings'
  ? h(SettingsPage, { layout, props: settings })
  : h(UserPage, { layout, props: profile }) }) }).use(i18n).use(router)
for (const name of ['code-copy', 'code-highlight', 'math-render', 'content-enhancements']) app.directive(name, {})
app.mount('#app')
