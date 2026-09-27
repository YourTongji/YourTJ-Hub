import { createApp, h } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import AppShell from '../../../src/site/components/AppShell.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import type { LayoutPayload } from '@gooseforum/client'
import '../../../src/styles/resource.css'

// 前台外壳侧栏折叠夹具（issue #798）：渲染生产 AppShell 与样式，
// 由 test/shell-sidebar.browser.mjs 测量真实轨道宽度、裁切与持久化。
// ?rail=1 复现 xl 三列布局（话题详情/wiki 详情页带右侧 rail）。
await setLocale('zh')

const withRail = new URLSearchParams(location.search).get('rail') === '1'

const layout = {
  site: {
    name: 'yourtj',
    description: '侧栏折叠夹具',
    logo: '',
    favicon: '',
    brandType: 'text',
    brandText: 'yourtj',
    brandImage: '',
  },
  viewer: {
    id: 0,
    username: '',
    email: '',
    avatarUrl: '',
    isAuthenticated: false,
    canAccessAdmin: false,
    isModerator: false,
    requiresEmailVerification: false,
    adminPermissions: [],
  },
  sidebar: {
    main: [],
    resources: [{ key: 'status', label: '运行状态', url: 'https://status.yourtj.de' }],
    groups: [{ key: 'custom', title: '自定义', items: [{ key: 'custom-a', label: '自定义入口', url: '/custom' }] }],
    categories: Array.from({ length: 8 }, (_, index) => ({
      id: index + 1,
      label: `分类 ${index + 1}`,
      url: `/c/${index + 1}`,
      color: '#10b981',
    })),
    activeKey: 'topics',
  },
  footer: { links: [{ name: 'About', url: '/about' }], primary: ['© yourtj'] },
  unread: { notifications: false, messages: false, moderationReports: false },
  posting: { maxTitleLength: 80 },
  theme: { enabled: false, current: 'gf-light', themeColor: '#fbfdff' },
  umamiEnabled: false,
} as LayoutPayload

const router = createRouter({
  history: createMemoryHistory(),
  routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
})
await router.push('/')
await router.isReady()

createApp({
  render: () => h(AppShell, { layout, rail: withRail }, {
    default: () => h('div', { class: 'gf-card', 'data-testid': 'feed', style: 'min-height: 1400px' }, '内容列'),
    rail: () => h('div', { class: 'gf-card', 'data-testid': 'rail', style: 'min-height: 600px' }, 'rail'),
  }),
}).use(i18n).use(router).mount('#app')

document.documentElement.dataset.ready = 'true'
