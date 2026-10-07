import { createApp, h, ref } from 'vue'
import { createRouter, createMemoryHistory } from 'vue-router'
import AppShell from '../../../src/site/components/AppShell.vue'
import SettingsPage from '../../../src/site/pages/SettingsPage.vue'
import PostComposer from '../../../src/site/components/PostComposer.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { LayoutPayload, SettingsPageProps } from '@gooseforum/client'

// Render shipped components and CSS; the acceptance server supplies disposable API data.
const params = new URLSearchParams(location.search)
document.documentElement.dataset.theme = params.get('theme') === 'dark' ? 'gf-dark' : 'gf-light'
await setLocale('zh')
const viewer = {
  id: 7,
  username: 'yzxoi',
  email: '',
  avatarUrl: '/assets/static/pic/1.webp',
  isAuthenticated: true,
  canAccessAdmin: false,
  isModerator: false,
  requiresEmailVerification: false,
  adminPermissions: [],
}
const layout: LayoutPayload = {
  site: {
    name: 'yourtj',
    description: '',
    logo: '',
    favicon: '',
    brandType: 'text',
    brandText: 'yourtj',
    brandImage: '',
  },
  viewer,
  sidebar: { categories: [], activeKey: 'settings' },
  footer: { links: [], primary: [] },
  unread: { notifications: false, messages: false },
  posting: { maxTitleLength: 100 },
  theme: { enabled: false, current: 'gf-light', themeColor: '' },
  umamiEnabled: false,
}
const settings: SettingsPageProps = {
  user: {
    id: 7,
    username: 'yzxoi',
    nickname: 'yzxoi',
    email: 'demo@example.test',
    locale: 'zh',
    avatarUrl: viewer.avatarUrl,
    profileCoverUrl: '',
    bio: 'Hello World!',
    signature: '',
    websiteName: '',
    website: '',
    prestige: 0,
    createdAt: '2026-08-06T00:00:00Z',
    externalInformation: {},
    wornBadgeCode: '',
    badges: [],
    wearableBadges: [],
    wornBadge: null,
  },
  googleOAuthReady: false,
  canSetPassword: false,
  stats: {
    topicCount: 14,
    replyCount: 78,
    followerCount: 15,
    followingCount: 3,
    likeReceivedCount: 180,
    likeGivenCount: 32,
    collectionCount: 1,
    createdAt: '2026-08-06T00:00:00Z',
  },
  tabs: ['profile', 'account', 'privacy', 'binding', 'security', 'content', 'deleted', 'general'].map((key) => ({
    key,
    url: `/settings?tab=${key}`,
    active: key === 'profile',
  })),
}
const router = createRouter({
  history: createMemoryHistory(),
  routes: [{ path: '/:pathMatch(.*)*', component: { render: () => null } }],
})
await router.push('/settings')
await router.isReady()
createApp({
  setup() {
    const body = ref('这段回复尚未发布。设置匿名身份时，草稿会保留在这里。')
    const identity = ref<'member' | 'persona'>('member')
    const open = ref(true)
    return () =>
      h(
        AppShell,
        { layout, headerTitle: params.get('screen') === 'composer' ? '回复讨论' : '设置' },
        {
          default: () =>
            params.get('screen') === 'composer'
              ? h(PostComposer, {
                  viewer,
                  authenticated: true,
                  errorMessage: '',
                  successMessage: '',
                  submitting: false,
                  open: open.value,
                  modelValue: body.value,
                  identity: identity.value,
                  'onUpdate:modelValue': (value: string) => {
                    body.value = value
                  },
                  'onUpdate:identity': (value: 'member' | 'persona') => {
                    identity.value = value
                  },
                  'onUpdate:open': (value: boolean) => {
                    open.value = value
                  },
                })
              : h(SettingsPage, { layout, props: settings }),
        },
      )
  },
})
  .use(i18n)
  .use(router)
  .mount('#app')
