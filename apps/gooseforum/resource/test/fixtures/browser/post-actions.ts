import { createApp, h } from 'vue'
import PostStream from '../../../src/site/components/PostStream.vue'
import PostReplyRow from '../../../src/site/components/PostReplyRow.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { PostPayload, ViewerPayload } from '@gooseforum/client'

await setLocale('en')
const params = new URLSearchParams(location.search)
const count = Number(params.get('count') || 0)
const topic = params.has('topic')
const guest = params.has('guest')
const own = params.has('own')
const post: PostPayload = {
  id: 202, postNo: 2, topicId: 42, content: 'Reply', renderedContent: '<p>Reply</p>',
  processStatus: 0, isHidden: false, isAuthorDeleted: false, isModeratorRemoved: false,
  canModerate: false, author: { id: 2, username: 'alice', avatarUrl: '' },
  createdAt: '2026-09-16T00:00:00Z', likeCount: count, isLiked: false, isBookmarked: false,
}
const viewer = { id: 1, username: 'viewer', isAuthenticated: !guest } as ViewerPayload
const app = createApp({ render: () => h('div', [
  h('section', { id: 'flat' }, h(PostStream, {
    topicId: 42, topicTitle: 'Actions', contentType: 3, viewer, canPost: true,
    initialPostStream: { posts: [post], hasBefore: topic, hasAfter: false },
    topicActions: topic ? {
      likeCount: 8, isLiked: false, isBookmarked: false, isWatched: false,
      processStatus: 0, authorDeleted: false, moderatorRemoved: false,
      isOwnTopic: own, canModerateTopic: own, createdAt: post.createdAt, updatedAt: post.createdAt,
      replyCount: 1, viewCount: 12, maxPostNo: 2, participants: [], author: post.author, description: 'Topic body',
    } : undefined,
  })),
  h('section', { id: 'tree' }, h(PostReplyRow, {
    post, authenticated: true, canPost: true,
    actionState: { likeCount: count, isLiked: false, isBookmarked: false, actingLike: false, actingBookmark: false },
  })),
]) }).use(i18n)
for (const name of ['code-copy', 'code-highlight', 'math-render', 'content-enhancements']) app.directive(name, {})
app.mount('#app')
