import { createApp, h } from 'vue'
import PostStream from '../../../src/site/components/PostStream.vue'
import PostReplyRow from '../../../src/site/components/PostReplyRow.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { PostPayload, ViewerPayload } from '@gooseforum/client'

await setLocale('en')
const count = Number(new URLSearchParams(location.search).get('count') || 0)
const post: PostPayload = {
  id: 202, postNo: 2, topicId: 42, content: 'Reply', renderedContent: '<p>Reply</p>',
  processStatus: 0, isHidden: false, isAuthorDeleted: false, isModeratorRemoved: false,
  canModerate: false, author: { id: 2, username: 'alice', avatarUrl: '' },
  createdAt: '2026-09-16T00:00:00Z', likeCount: count, isLiked: false, isBookmarked: false,
}
const viewer = { id: 1, username: 'viewer', isAuthenticated: true } as ViewerPayload
const app = createApp({ render: () => h('div', [
  h('section', { id: 'flat' }, h(PostStream, {
    topicId: 42, topicTitle: 'Actions', contentType: 3, viewer, canPost: true,
    initialPostStream: { posts: [post], hasBefore: false, hasAfter: false },
  })),
  h('section', { id: 'tree' }, h(PostReplyRow, {
    post, authenticated: true, canPost: true,
    actionState: { likeCount: count, isLiked: false, isBookmarked: false, actingLike: false, actingBookmark: false },
  })),
]) }).use(i18n)
for (const name of ['code-copy', 'code-highlight', 'math-render', 'content-enhancements']) app.directive(name, {})
app.mount('#app')
