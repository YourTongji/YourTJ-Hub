import { createApp, h } from 'vue'
import TopicRow from '../../../src/site/components/TopicRow.vue'
import TopicFeedPreview from '../../../src/site/components/TopicFeedPreview.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { TopicPayload } from '@gooseforum/client'

await setLocale('zh')
const topic: TopicPayload = {
  id: 17, title: '校园散步', description: '今天的校园风景很好', url: '/p/post/17',
  author: { id: 1, username: 'author', nickname: '一位同学', avatarUrl: '/assets/test/fixtures/browser/topic-feed-art.svg' }, participants: [],
  categories: [], replyCount: 0, viewCount: 0, pinWeight: 0, processStatus: 0,
  activityText: '刚刚', lastUpdateTime: '2026-10-04T00:00:00Z', contentType: 2,
}
createApp({
  render: () => h('main', { class: 'mx-auto max-w-5xl space-y-4 p-3' },
    [TopicRow, TopicFeedPreview].flatMap((component, index) => [0, 2].map(status =>
      h('section', { 'data-case': `${index === 0 ? 'row' : 'card'}-${status}`, class: 'rounded border border-line bg-base-100' }, [
        h(component, { topic: { ...topic, processStatus: status }, compact: true }),
      ]),
    )),
  ),
}).use(i18n).mount('#app')
