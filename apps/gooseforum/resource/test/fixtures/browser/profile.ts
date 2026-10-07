import { createApp, h } from 'vue'
import { createRouter, createMemoryHistory } from 'vue-router'
import type { PostPayload } from '@gooseforum/client'
import { layout, memberProfile, anonymousProfile, persona, identityState } from '../profile'
import AppShell from '../../../src/site/components/AppShell.vue'
import UserPage from '../../../src/site/pages/UserPage.vue'
import AnonymousProfilePage from '../../../src/site/pages/AnonymousProfilePage.vue'
import PostReplyRow from '../../../src/site/components/PostReplyRow.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
const params = new URLSearchParams(location.search)
document.documentElement.dataset.theme = params.get('theme') === 'dark' ? 'gf-dark' : 'gf-light'
await setLocale('zh')
if (params.get('guest') === '1') { layout.viewer.isAuthenticated = false; layout.viewer.id = 0 }
// Synthetic private API for testing the actual profile/dialog/CSS. The URL flag
// stands in for a server-persisted preference across the production page reload.
const fixtureState = { ...identityState, showContent: params.get('hidden') !== '1' }
anonymousProfile.showContent = fixtureState.showContent
if (!fixtureState.showContent) {
  anonymousProfile.topics = []; anonymousProfile.replies = []
  anonymousProfile.topicCount = 0; anonymousProfile.replyCount = 0
}
const nativeFetch = window.fetch.bind(window)
window.fetch = async (input, init) => {
  const path = String(input)
  if (path === '/api/forum/anonymous/state')
    return Response.json({ code: 0, result: fixtureState })
  if (path === '/api/forum/anonymous/privacy') {
    fixtureState.showContent = JSON.parse(String(init?.body)).showContent
    const url = new URL(location.href)
    if (fixtureState.showContent) url.searchParams.delete('hidden')
    else url.searchParams.set('hidden', '1')
    history.replaceState(null, '', url)
    return Response.json({ code: 0, result: true })
  }
  return nativeFetch(input, init)
}
persona.avatarUrl = '/assets/test/fixtures/browser/anonymous-admin-avatar-0.svg'
layout.viewer.avatarUrl = '/assets/test/fixtures/browser/anonymous-admin-avatar-1.svg'
const member = params.get('member') === '1'
const reply: PostPayload = {
  id: 202, postNo: 2, topicId: 1, isAnonymous: true,
  content: '抱着一杯热茶，看了一下午的雨。', renderedContent: '<p>抱着一杯热茶，看了一下午的雨。</p>',
  processStatus: 0, isHidden: false, isAuthorDeleted: false, isModeratorRemoved: false,
  canModerate: false, author: { ...persona, id: 0, username: persona.name },
  createdAt: '2026-10-07T09:00:00Z', likeCount: 3, isLiked: false, isBookmarked: false,
}
const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:pathMatch(.*)*', component: { render: () => null } }] })
const app = createApp({ render: () => h(AppShell, { layout }, { default: () => params.has('comment')
  ? h('section', { class: 'gf-card p-4' }, [h('h1', { class: 'mb-4 text-xl font-bold' }, '下雨天适合抱着热茶看书'), h(PostReplyRow, {
    post: reply, authenticated: true, canPost: true,
    actionState: { likeCount: 3, isLiked: false, isBookmarked: false, actingLike: false, actingBookmark: false },
  })])
  : h(member ? UserPage : AnonymousProfilePage, { layout, props: member ? memberProfile : anonymousProfile }) }) }).use(i18n).use(router)
for (const name of ['code-copy', 'code-highlight', 'math-render', 'content-enhancements']) app.directive(name, {})
app.mount('#app')
