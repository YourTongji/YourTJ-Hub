import { createApp, h } from 'vue'
import TopicList from '../../../src/site/components/TopicList.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { TopicPayload } from '@gooseforum/client'
await setLocale('zh')
const topic: TopicPayload = {
  id: 42, title: '卡片互动', description: '卡片阅读与快捷互动', url: '/p/42',
  author: { id: 1, username: 'alice', avatarUrl: '' }, participants: [],
  categories: [{ id: 9, name: '学习', url: '/c/9', color: '#10b981' }],
  replyCount: 7, viewCount: 100, likeCount: 5, pinWeight: 0, processStatus: 0,
  activityText: '刚刚', lastUpdateTime: '2026-09-11T00:00:00Z', contentType: 3,
  liked: false, bookmarked: false,
}
createApp({ render: () => h(TopicList, { topics: [topic], viewerId: 1, feedMode: 'card' }) }).use(i18n).mount('#app')
