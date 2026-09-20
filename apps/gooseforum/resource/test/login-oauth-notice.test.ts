// @vitest-environment happy-dom
// PR #552 review P2：oauthNotice 注册引导页不再展示 GitHub/Google 社交入口。
// 被 OAuth 回调按 issue #531 送来密码注册的身份，再点社交按钮会原样回到本页，
// 形成「按提示注册却回到同一页」的可见循环；普通注册页不受影响。
import { describe, expect, test, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import type { LayoutPayload, LoginPageProps } from '@gooseforum/client'

vi.mock('../src/runtime/api', () => ({
  getCaptcha: vi.fn(async () => ({ captchaId: 'captcha-id', captchaImg: 'data:image/png;base64,x' })),
  login: vi.fn(async () => ({})),
  register: vi.fn(async () => ''),
  forgotPassword: vi.fn(async () => ''),
  verifyTotp: vi.fn(async () => undefined),
}))
vi.mock('../src/runtime/flash-message', () => ({
  queueFlashMessage: vi.fn(),
}))
vi.mock('../src/runtime/site-theme', async () => {
  const { ref } = await import('vue')
  return {
    useSiteTheme: () => ({ isDark: ref(false) }),
    setThemePreference: vi.fn(),
  }
})

import LoginPage from '../src/site/pages/LoginPage.vue'

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
  sidebar: { categories: [], activeKey: '' },
  footer: { links: [], primary: [] },
  unread: { notifications: false, messages: false },
  posting: { maxTitleLength: 100 },
  theme: { enabled: false, current: 'gf-light', themeColor: '' },
  umamiEnabled: false,
}

function buildProps(oauthNotice: boolean): LoginPageProps {
  return {
    initialMode: 'register',
    redirectUrl: '/',
    githubUrl: '/api/auth/github',
    googleUrl: '/api/auth/google',
    googleReady: true,
    termsOfServiceEnabled: false,
    privacyPolicyEnabled: false,
    allowedDomains: ['tongji.edu.cn'],
    oauthNotice,
  }
}

function mountPage(oauthNotice: boolean) {
  return mount(LoginPage, {
    props: { layout, props: buildProps(oauthNotice) },
    global: { plugins: [i18n] },
  })
}

describe('LoginPage oauthNotice 注册引导（PR #552 review P2）', () => {
  test('oauthNotice=true：隐藏 GitHub/Google 社交入口，保留 oauthNoAccount 提示', () => {
    const wrapper = mountPage(true)
    expect(wrapper.text()).toContain(i18n.global.t('auth.oauthNoAccount'))
    expect(wrapper.find('a[href="/api/auth/github"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/api/auth/google"]').exists()).toBe(false)
    wrapper.unmount()
  })

  test('oauthNotice=false：普通注册页仍展示社交入口（不回归）', () => {
    const wrapper = mountPage(false)
    expect(wrapper.find('a[href="/api/auth/github"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/api/auth/google"]').exists()).toBe(true)
    wrapper.unmount()
  })
})


describe('Tongji sign-in and signup', () => {
  for (const initialMode of ['login', 'register'] as const) {
    test(`school entry is present on ${initialMode} and preserves the return destination`, async () => {
      const wrapper = mount(LoginPage, {
        props: { layout, props: { ...buildProps(false), initialMode, tongjiReady: true, tongjiUrl: '/api/auth/tongji?redirect=%2Fcampus' } },
        global: { plugins: [i18n] },
      })
      for (const locale of ['zh', 'en', 'ja', 'de'] as const) {
        i18n.global.locale.value = locale
        await wrapper.vm.$nextTick()
        const links = wrapper.findAll('[data-tongji-login]')
        expect(links).toHaveLength(2)
        expect(links[0].attributes('href')).toBe(`/api/auth/tongji?redirect=%2Fcampus&locale=${locale}`)
        expect(links[0].text()).toContain(i18n.global.t('auth.tongjiLogin'))
        expect(wrapper.text()).toContain(i18n.global.t('auth.tongjiSignupHint'))
      }
      wrapper.unmount()
      i18n.global.locale.value = 'zh'
    })
  }
  test('unconfigured integration stays hidden', () => {
    const wrapper = mountPage(false)
    expect(wrapper.find('[data-tongji-login]').exists()).toBe(false)
    wrapper.unmount()
  })
  test('email conflict has a recovery instruction and no raw upstream content', () => {
    const wrapper = mount(LoginPage, {
      props: { layout, props: { ...buildProps(false), tongjiNotice: 'accountExists' } },
      global: { plugins: [i18n] },
    })
    expect(wrapper.text()).toContain(i18n.global.t('auth.tongjiAccountExists'))
    wrapper.unmount()
  })
})
