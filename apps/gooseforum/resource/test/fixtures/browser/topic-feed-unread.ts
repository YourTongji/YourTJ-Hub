import { createApp, h } from 'vue'
import TopicFeedPreview from '../../../src/site/components/TopicFeedPreview.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { TopicPayload } from '@gooseforum/client'

await setLocale('zh')
const base: TopicPayload = {
  id: 101, title: '', description: '终于有时间画画了', url: '/p/101',
  author: { id: 1, username: 'alice', nickname: '小爱', avatarUrl: '/assets/test/fixtures/browser/topic-feed-art.svg' }, participants: [],
  categories: [{ id: 9, name: '瞬间', url: '/c/9', color: '#10b981' }],
  replyCount: 7, viewCount: 100, pinWeight: 0, processStatus: 0,
  activityText: '刚刚', lastUpdateTime: '2026-10-03T00:00:00Z', contentType: 2, unseen: true,
}
const scenarios: Array<[string, Partial<TopicPayload>]> = [
  ['unseen-with-image', { images: ['/assets/test/fixtures/browser/topic-feed-art.svg'] }],
  ['seen-without-image', { unseen: false, images: [] }],
  ['image-only-unseen', { description: '', images: ['/assets/test/fixtures/browser/topic-feed-art.svg'] }],
  ['empty-unseen', { description: '', images: [] }],
  ['empty-long-author-unseen', { description: '', images: [], author: { ...base.author, nickname: '这是一段在窄屏上会被截断的很长作者名称' } }],
  ['long-unseen', { description: '这是一个很长的瞬间摘要，用来验证未读标记在文字折成多行后依然保留在第一行，并且不会被两行截断规则隐藏。'.repeat(3), images: [] }],
  ['titled-unseen', { title: '有标题的瞬间', images: [] }],
]
const topic = (id: number, overrides: Partial<TopicPayload>): TopicPayload => ({ ...base, id, ...overrides })

function surface(compact: boolean, label: string) {
  return h('section', { class: 'min-w-0 space-y-3' }, [
    h('h2', { class: 'px-4 pt-4 text-sm font-semibold text-base-content' }, label),
    ...scenarios.map(([name, overrides], index) => h('article', {
      class: 'group mx-3 overflow-hidden rounded-xl border border-line bg-base-100 shadow-sm',
      'data-case': `${compact ? 'compact' : 'preview'}-${name}`,
    }, [h(TopicFeedPreview, {
      topic: topic((compact ? 200 : 300) + index, overrides),
      compact,
      showStats: false,
    })])),
  ])
}

createApp({
  render: () => h('div', { class: 'mx-auto grid min-w-0 max-w-5xl gap-5 p-3 md:grid-cols-2' }, [
    surface(true, '卡片列表'),
    surface(false, '桌面悬停预览'),
  ]),
}).use(i18n).mount('#app')
