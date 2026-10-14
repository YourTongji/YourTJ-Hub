// issue #1063：信息流内容类型徽章（提问/瞬间/文章）在长标题、窄内容列下被挤压折行。
// 这里铺「宽屏内容列（约 634px）/ 窄屏满宽」两组、每组各含无压力基准行与长标题行，另加一张卡片，
// 供 topic-badges.browser.mjs 量测徽章几何。
import { createApp, h } from 'vue'
import TopicList from '../../../src/site/components/TopicList.vue'
import TopicRow from '../../../src/site/components/TopicRow.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { TopicPayload } from '@gooseforum/client'

// ?locale=<code> 让同一个 fixture 覆盖 en/de/ja 的长标签（Question/Artikel/モーメント…）。
await setLocale(new URLSearchParams(window.location.search).get('locale') ?? 'zh')

// 长标题吃满内容列，与 issue 截图中的「TJ Class Schedule 更新！妈妈再也不用担心我忘记提交作业了……」同类。
const LONG_TITLE = 'TJ Class Schedule 更新！妈妈再也不用担心我忘记提交作业了——课表同步、教室变更与考试周提醒全都帮你记着，标题再长也要完整阅读'
const SHORT_TITLE = '短标题'

// 列表行在 lg 断点走 5 列网格：内容列 minmax(0,1fr) + 112/72/72/88 + gap/px。
// 容器 1040px 时内容列约 634px，与 issue 报告的「宽屏侧栏占位后约 634px」一致。
const LG_ROW_WIDTH = 1040

function topic(contentType: 0 | 1 | 2 | 3, id: number, title: string): TopicPayload {
  return {
    id,
    title,
    description: '内容描述',
    url: `/p/${id}`,
    author: { id: 1, username: 'tester', nickname: '测试者', avatarUrl: '' },
    participants: [{ id: 1, username: 'tester', nickname: '测试者', avatarUrl: '' }],
    categories: [{ id: 7, name: '校园生活', url: '/c/7', color: '#3b82f6' }],
    replyCount: 12,
    viewCount: 640,
    pinWeight: 0,
    processStatus: 0,
    activityText: '刚刚',
    lastUpdateTime: '2026-10-05T00:00:00Z',
    contentType,
  }
}

const listRows = ([1, 2, 3] as const).flatMap((type, index) => [
  // 自然宽度基准：短标题无收缩压力，徽章保持完整尺寸。
  h('div', { 'data-case': `natural-${type}`, style: `width:${LG_ROW_WIDTH}px` }, [
    h(TopicRow, { topic: topic(type, index * 10 + 1, SHORT_TITLE), home: true }),
  ]),
  h('div', { 'data-case': `narrow-lg-${type}`, style: `width:${LG_ROW_WIDTH}px` }, [
    h(TopicRow, { topic: topic(type, index * 10 + 2, LONG_TITLE), home: true }),
  ]),
  // 窄屏行跟随视口宽度（width:100%），测试用不同 viewport 反复量测。
  h('div', { 'data-case': `natural-sm-${type}`, style: 'width:100%' }, [
    h(TopicRow, { topic: topic(type, index * 10 + 3, SHORT_TITLE), home: true }),
  ]),
  h('div', { 'data-case': `narrow-sm-${type}`, style: 'width:100%' }, [
    h(TopicRow, { topic: topic(type, index * 10 + 4, LONG_TITLE), home: true }),
  ]),
])

createApp({
  render: () => h('div', { class: 'flex flex-col gap-4 p-4' }, [
    ...listRows,
    h('div', { 'data-case': 'card', style: 'width:320px' }, [
      h(TopicList, { topics: [topic(3, 99, LONG_TITLE)], viewerId: 1, feedMode: 'card' }),
    ]),
  ]),
}).use(i18n).mount('#app')
