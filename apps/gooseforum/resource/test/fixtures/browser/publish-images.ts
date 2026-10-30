import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import { createMemoryHistory, createRouter } from 'vue-router'
import PublishPage from '../../../src/site/pages/PublishPage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import { applyThemeTemporary, type SiteTheme } from '../../../src/runtime/site-theme'
import '../../../src/styles/resource.css'
await setLocale((new URLSearchParams(location.search).get('lang') || 'zh') as 'zh' | 'en' | 'ja' | 'de')
applyThemeTemporary((new URLSearchParams(location.search).get('theme') || 'gf-light') as SiteTheme)
;(window as unknown as { testSetLocale: typeof setLocale }).testSetLocale = setLocale
;(window as unknown as { testToolbarLabels: () => Record<string,string> }).testToolbarLabels = () => Object.fromEntries(['code','link','list','quote','italic','bold','upload'].map(type => [type, i18n.global.t('editor.toolbar.' + type)]))
const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
await router.push('/')
await router.isReady()
createApp({
  render: () => h(PublishPage, {
    layout: { viewer: { id: 7, isAuthenticated: true }, posting: {} } as any,
    props: { topicId: 17, isEditing: true, categories: [{ id: 1, name: '校园生活', color: '#059669' }], topic: {
      title: '修改后重新提交', content: '修改正文和图片后，可以重新提交审核。', categoryIds: [1], contentType: new URLSearchParams(location.search).get('type') === '2' ? 2 : 3,
      images: ['/assets/test/fixtures/browser/topic-feed-art.svg'],
    } } as any,
  }),
}).use(i18n).use(router).mount('#app')
