import { createApp } from 'vue'
import { createRouter, createMemoryHistory } from 'vue-router'
import AdminApp from '../../../src/admin/AdminApp.vue'
import AnonymousManagementPage from '../../../src/admin/pages/management/AnonymousManagementPage.vue'
import FeedStatisticsPage from '../../../src/admin/pages/FeedStatisticsPage.vue'
import StatsPage from '../../../src/admin/pages/StatsPage.vue'
import { configureAdminAccess } from '../../../src/admin/runtime/access'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import { layout } from '../profile'
import type {
  AdminAnonymousIdentity,
  FeedSummary,
} from '../../../src/admin/runtime/api'
import '../../../src/admin/styles/admin.css'

// Synthetic data renders production components/CSS, with no real private mappings.
const params = new URLSearchParams(location.search)
await setLocale((params.get('lang') || 'zh') as 'zh' | 'en' | 'ja' | 'de')
if (params.get('theme') === 'dark')
  document.documentElement.classList.add('dark')
layout.viewer = {
  ...layout.viewer,
  username: 'demo_admin',
  avatarUrl: '/assets/test/fixtures/browser/anonymous-admin-avatar-0.svg',
  canAccessAdmin: true,
  adminPermissions: [0, 7],
}
layout.site.name = 'YourTJ Forum'
layout.site.brandText = 'YourTJ Forum'
configureAdminAccess([0, 7])
const names = [
  '躲进云里的猫',
  '抱着松果的熊',
  '捧着月光的兔',
  '趴在书桌的狐',
  '睡在秋风的鹿',
  '望着山顶的鸟',
]
const identities: AdminAnonymousIdentity[] = Array.from(
  { length: 24 },
  (_, index) => ({
    kind: 'persona',
    publicUid: (index + 1).toString(16).padStart(32, '0'),
    name: names[index % names.length]!,
    avatarUrl: `/assets/test/fixtures/browser/anonymous-admin-avatar-${index % 3}.svg`,
    profileUrl: '/a/demo',
    selectedAt: '2026-10-07T08:00:00Z',
    disabled: index === 2,
    governanceDisabled: index === 1,
    owner: {
      userId: 101 + index,
      username: `campus_user_${String(index + 1).padStart(2, '0')}`,
      closed: index === 4,
      frozen: index === 3,
    },
  }),
)
const summary: FeedSummary = {
  enabled: true,
  rankingReady: true,
  metricsEnabled: true,
  rolloutPercent: 20,
  rawRetentionDays: 30,
  paramsHash: '7c918b2d3ad9f642',
  truncated: false,
  health: {
    accepted: 12684,
    dropped: 0,
    queueLength: 12,
    queueBytes: 32768,
    oldestQueuedMs: 86,
    sampleQueueLength: 3,
    sampleQueueBytes: 8192,
    backgroundFailures: 0,
    lastRankAt: 1791379200,
    previousEpochIncomplete: false,
    epoch: 'demo-process',
    parameters: {
      queryTimeoutMS: 250,
      backgroundPerSecond: 20,
      personalization: {
        OutOfNetwork: 0.75,
        AuthorDiversity: 0.5,
        RecencyHalfLifeHours: 12,
      },
    },
  },
  rows: ['served', 'visible', 'open', 'read', 'like', 'reply'].map(
    (metric, index) => ({
      day: '2026-10-07',
      feed: index < 4 ? 'for_you' : 'daily',
      hash: '7c918b2d3ad9f642',
      rankHash: '48abc192',
      experiment: 'autumn-2026',
      variant: index % 2 ? 'treatment' : 'control',
      weightVariant: 'base',
      capability: 'web-v1',
      metric,
      count: [6840, 4210, 1628, 1245, 386, 129][index]!,
    }),
  ),
  periods: [
    {
      id: 'autumn-2026',
      hash: '7c918b2d3ad9f642',
      enrollUntil: '2026-10-14T08:00:00Z',
      analyzeAt: '2026-10-22T08:00:00Z',
      aborted: '',
      assignedControl: 820,
      assignedTreatment: 816,
      result: '',
    },
  ],
}
const originalFetch = window.fetch.bind(window)
window.fetch = async (input, init) => {
  const url =
    typeof input === 'string'
      ? input
      : input instanceof URL
        ? input.href
        : input.url
  const body = typeof init?.body === 'string' ? JSON.parse(init.body) : {}
  const json = (result: unknown) =>
    new Response(JSON.stringify({ code: 0, result }), {
      headers: { 'Content-Type': 'application/json' },
    })
  if (url.endsWith('/anonymous-identities/list')) {
    const matches = identities.filter(
      (row) =>
        (!body.search ||
          `${row.name} ${row.publicUid} ${row.owner.username} ${row.owner.userId}`.includes(
            body.search,
          )) &&
        (body.status === 'all' ||
          body.status ===
            (row.governanceDisabled
              ? 'banned'
              : row.disabled
                ? 'disabled'
                : 'active')),
    )
    return json({
      items: matches.slice(
        (body.page - 1) * body.pageSize,
        body.page * body.pageSize,
      ),
      total: matches.length,
      page: body.page,
      pageSize: body.pageSize,
    })
  }
  if (url.endsWith('/anonymous-identities/govern')) {
    identities.find(
      (row) => row.publicUid === body.publicUid,
    )!.governanceDisabled = body.disabled
    return json(true)
  }
  if (url.endsWith('/feed/summary')) return json(summary)
  if (url.endsWith('/get-site-statistics'))
    return json({
      userCount: 1641,
      userMonthCount: 109,
      topicMaxId: 483,
      topicMonthCount: 79,
      postMaxId: 2104,
      linksCount: 6,
    })
  if (url.endsWith('/traffic-overview'))
    return json(
      Array.from({ length: 7 }, (_, i) => ({
        date: `2026-10-0${i + 1}`,
        regCount: 6 + i,
        topicCount: 12 + i,
        replyCount: 38 + i * 3,
        courseReviewCount: i,
      })),
    )
  if (url.endsWith('/server-version'))
    return json({
      version: 'v0.0.50',
      commit: 'demo',
      buildDate: '2026-10-07',
      mode: 'development',
    })
  if (url.includes('api.github.com'))
    return new Response('[]', {
      headers: { 'Content-Type': 'application/json' },
    })
  return originalFetch(input, init)
}
const router = createRouter({
  history: createMemoryHistory(),
  routes: [
    { path: '/admin/anonymous-identities', component: AnonymousManagementPage },
    { path: '/admin/feed-statistics', component: FeedStatisticsPage },
    { path: '/admin', component: StatsPage },
  ],
})
await router.push(
  params.get('page') === 'feed'
    ? '/admin/feed-statistics'
    : params.get('page') === 'home'
      ? '/admin'
      : '/admin/anonymous-identities',
)
createApp(AdminApp, {
  payload: {
    component: 'manage-home',
    props: {},
    layout,
    meta: { title: 'Admin preview' },
    url: router.currentRoute.value.path,
    version: 'test',
  },
})
  .use(i18n)
  .use(router)
  .mount('#app')
document.documentElement.dataset.ready = 'true'
