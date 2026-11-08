import { h } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { UserBookmarkPayload, UserProfileProps } from '@gooseforum/client'
import { layout, memberProfile } from '../profile'
import { createBrowserApp } from './create-app'
import AppShell from '../../../src/site/components/AppShell.vue'
import UserPage from '../../../src/site/pages/UserPage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

// #1639: real profile components and CSS, synthetic bookmarks only. The
// activity variant covers the retained render branch, not a current server tab.
const params = new URLSearchParams(location.search)
const activity = params.get('section') === 'activity'
document.documentElement.dataset.theme = params.get('theme') === 'dark' ? 'gf-dark' : 'gf-light'
await setLocale('zh')
layout.viewer.avatarUrl = '/assets/test/fixtures/browser/anonymous-admin-avatar-1.svg'

const longExcerpt = '这是用于检查回复收藏摘要截断的合成内容。窄屏与宽屏都应该最多显示两行，同时保留标题、楼层和收藏时间。'.repeat(10)
const bookmarks: UserBookmarkPayload[] = [
  { id: 1, type: 'post', topicId: 200, postId: 201, postNo: 3, url: '/p/200?postNo=3',
    title: '长摘要回复收藏：图书馆窗边的银杏与期末复习计划，窄屏长标题仍应省略', excerpt: longExcerpt, bookmarkedAt: '2026-11-01T09:00:00Z' },
  { id: 2, type: 'post', topicId: 200, postId: 202, postNo: 4, url: '/p/200?postNo=4',
    title: '短摘要回复收藏', excerpt: '简短摘要。', bookmarkedAt: '2026-11-01T08:00:00Z' },
  { id: 3, type: 'post', topicId: 200, postId: 203, postNo: 5, url: '/p/200?postNo=5',
    title: '无摘要回复收藏', excerpt: '', bookmarkedAt: '2026-11-01T07:00:00Z' },
  { id: 4, type: 'topic', topicId: 100, url: '/p/100',
    title: '主题收藏', bookmarkedAt: '2026-11-01T06:00:00Z' },
]
const profile: UserProfileProps = {
  ...memberProfile,
  user: { ...memberProfile.user, nickname: '收藏摘要验收（合成数据）', profileCoverUrl: '', avatarUrl: '/assets/test/fixtures/browser/anonymous-admin-avatar-0.svg' },
  section: activity ? 'activity' : 'bookmarks', activityTab: 'bookmarks', bookmarks,
  tabs: memberProfile.tabs.map(tab => ({ ...tab, active: tab.key === (activity ? 'activity' : 'bookmarks') })),
  activityTabs: activity ? [{ key: 'bookmarks', url: '/u/1024/activity/bookmarks', active: true }] : [],
}
const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:pathMatch(.*)*', component: { render: () => null } }] })
// Do not leave the memory router at '/': AppShell would render the home-only FAB.
await router.push(activity ? '/u/1024/activity/bookmarks' : '/u/1024/bookmarks')
const app = createBrowserApp({ render: () => h(AppShell, { layout }, {
  default: () => h(UserPage, { layout, props: profile }),
}) }).use(i18n).use(router)
for (const name of ['code-copy', 'code-highlight', 'math-render', 'content-enhancements']) app.directive(name, {})
app.mount('#app')
