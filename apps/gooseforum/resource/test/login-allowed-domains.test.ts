// @vitest-environment happy-dom
// issue #643 回归：注册邮箱域名白名单为空（shipped default）时，后端存量
// 配置行缺键会把 allowedDomains 序列化为 null；LoginPage setup 曾直接解引用
// 导致整页白屏。修复后页面必须容错挂载，注册表单退化为自由邮箱输入。
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
  insightFlareEnabled: false,
}

function buildProps(allowedDomains: unknown): LoginPageProps {
  return {
    initialMode: 'register',
    redirectUrl: '/',
    githubUrl: '/api/auth/github',
    googleUrl: '/api/auth/google',
    googleReady: true,
    termsOfServiceEnabled: false,
    privacyPolicyEnabled: false,
    allowedDomains: allowedDomains as LoginPageProps['allowedDomains'],
    oauthNotice: false,
  }
}

describe('LoginPage allowedDomains null 容错（issue #643）', () => {
  test('allowedDomains=null：注册页正常挂载，退化为自由邮箱输入', () => {
    const wrapper = mount(LoginPage, {
      props: { layout, props: buildProps(null) },
      global: { plugins: [i18n] },
    })
    expect(wrapper.find('form').exists()).toBe(true)
    // 自由邮箱分支：type=email 输入框；不再渲染域名后缀/域名下拉
    expect(wrapper.find('input[type="email"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('@')
    wrapper.unmount()
  })

  test('allowedDomains 为单域名数组：仍渲染域名后缀（不回归）', () => {
    const wrapper = mount(LoginPage, {
      props: { layout, props: buildProps(['tongji.edu.cn']) },
      global: { plugins: [i18n] },
    })
    expect(wrapper.find('input[type="email"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('@tongji.edu.cn')
    wrapper.unmount()
  })
})
