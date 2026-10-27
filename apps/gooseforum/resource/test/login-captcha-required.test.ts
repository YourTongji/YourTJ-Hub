// @vitest-environment happy-dom
// issue #1278 回归：管理端关闭验证码（securitySettings.captchaRequired=false）
// 后，服务端已放行，但登录/注册/忘记密码仍在前端无条件要求验证码，导致
// 关闭验证码的站点无法登录。修复后页面按 payload 的 captchaRequired 条件
// 渲染与校验，验证码未开启时不再请求、不拦截、也不随请求提交字段。
import { describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import type { LayoutPayload, LoginPageProps } from '@gooseforum/client'

vi.mock('../src/runtime/api', () => ({
  getCaptcha: vi.fn(async () => ({ captchaId: 'captcha-id', captchaImg: 'data:image/png;base64,x' })),
  getPasskeyStatus: vi.fn(async () => ({ enabled: false })),
  // 故意失败：避免 happy-dom 在登录成功后尝试真实导航；断言只关心调用参数。
  login: vi.fn(async () => {
    throw new Error('server down')
  }),
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

import { forgotPassword, getCaptcha, login } from '../src/runtime/api'
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

function buildProps(captchaRequired: boolean | undefined, initialMode: 'login' | 'register' | 'forgot'): LoginPageProps {
  return {
    initialMode,
    redirectUrl: '/',
    githubUrl: '',
    githubReady: false,
    googleUrl: '',
    googleReady: false,
    termsOfServiceEnabled: false,
    privacyPolicyEnabled: false,
    allowedDomains: [],
    oauthNotice: false,
    captchaRequired,
  }
}

function mountPage(captchaRequired: boolean | undefined, initialMode: 'login' | 'register' | 'forgot' = 'login') {
  return mount(LoginPage, {
    props: { layout, props: buildProps(captchaRequired, initialMode) },
    global: { plugins: [i18n] },
  })
}

describe('LoginPage 验证码开关（issue #1278）', () => {
  test('captchaRequired=false：登录不再要求验证码，也不拉取验证码', async () => {
    vi.mocked(getCaptcha).mockClear()
    vi.mocked(login).mockClear()
    const wrapper = mountPage(false)
    await flushPromises()
    expect(getCaptcha).not.toHaveBeenCalled()

    await wrapper.find('input[autocomplete="username"]').setValue('alice')
    await wrapper.find('input[autocomplete="current-password"]').setValue('secret')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(login).toHaveBeenCalledWith('alice', 'secret', '', '')
    wrapper.unmount()
  })

  test('captchaRequired=true：仍要求验证码（不回归默认配置）', async () => {
    vi.mocked(getCaptcha).mockClear()
    vi.mocked(login).mockClear()
    const wrapper = mountPage(true)
    await flushPromises()
    expect(getCaptcha).toHaveBeenCalled()

    await wrapper.find('input[autocomplete="username"]').setValue('alice')
    await wrapper.find('input[autocomplete="current-password"]').setValue('secret')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(login).not.toHaveBeenCalled()

    // 填上验证码后照常提交，captchaId/captchaCode 随请求下发。
    await wrapper.findAll('form')[0].findAll('input')[2].setValue('abcd')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(login).toHaveBeenCalledWith('alice', 'secret', 'captcha-id', 'abcd')
    wrapper.unmount()
  })

  test('captchaRequired 缺省（旧 payload）：按需要验证码处理', async () => {
    vi.mocked(getCaptcha).mockClear()
    const wrapper = mountPage(undefined)
    await flushPromises()
    expect(getCaptcha).toHaveBeenCalled()
    wrapper.unmount()
  })

  test('captchaRequired=false：忘记密码不要求验证码', async () => {
    vi.mocked(forgotPassword).mockClear()
    const wrapper = mountPage(false, 'forgot')
    await flushPromises()

    await wrapper.find('input[type="email"]').setValue('alice@example.com')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(forgotPassword).toHaveBeenCalledWith('alice@example.com', '', '', '')
    wrapper.unmount()
  })
})
