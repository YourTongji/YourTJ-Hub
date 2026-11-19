import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import WikiPage from '../../../src/site/pages/WikiPage.vue'
import { i18n } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { LayoutPayload, WikiDetailProps } from '@gooseforum/client'

i18n.global.locale.value = 'zh'
const layout = { viewer: { id: 7, username: 'reader', isAuthenticated: true, roles: [] } } as unknown as LayoutPayload
const detail: WikiDetailProps = {
  page: {
    id: 42, topicId: 42, namespace: 'guide', path: 'guide/start', title: 'Wiki 互动状态测试',
    content: '<p>本页面使用真实 Wiki 组件和合成数据，所有 API 均由浏览器测试拦截。</p>',
    toc: [], updatedAt: '2026-11-06T00:00:00Z', likeCount: 4, viewCount: 10, postCount: 0,
    liked: false, bookmarked: false, watched: false, canEdit: true, publishedRevisionNo: 1,
    editUrl: 'https://github.com/YourTongji/YourTJ-Wiki/edit/main/guide/start.md',
    historyUrl: 'https://github.com/YourTongji/YourTJ-Wiki/commits/main/guide/start.md',
  }, contributors: [], hotTopics: [],
}
const app = createApp({ render: () => h('main', { class: 'mx-auto w-full max-w-[1440px] p-4 xl:pr-[308px]' }, [h(WikiPage, { layout, props: detail })]) })
for (const directive of ['code-copy', 'code-highlight', 'math-render', 'content-enhancements']) app.directive(directive, {})
app.use(i18n).mount('#app')
