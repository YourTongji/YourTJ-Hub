// @vitest-environment happy-dom
// 回归测试：认证页（登录/重置密码）默认品牌字标必须与 AppShell 同一契约——
// 仅 brandType === 'image' 时采用管理端自定义品牌图（safeUrl 消毒），
// brandType === 'default' 时按站点主题切换 Light/Dark 字标变体。
// 线上实例（dev/prod）存在历史脏配置：brandType=default 且 brandImage 残留
// 旧浅色 PNG，旧实现「只要有 brandImage 就短路」导致深色模式登录页显示浅色 logo。
import { afterEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import { register } from '../src/runtime/api'
import { resolveApiMessage } from '../src/runtime/api-message'
import SiteSelect from '../src/site/components/SiteSelect.vue'
import { setThemePreference } from '../src/runtime/site-theme'
import type { LayoutPayload } from '@gooseforum/client'

vi.mock('../src/runtime/api', () => ({
  forgotPassword: vi.fn(),
  getCaptcha: vi.fn(async () => ({ captchaId: 'test', captchaImg: '' })),
  login: vi.fn(),
  register: vi.fn(),
  resetPassword: vi.fn(async () => 'ok'),
  verifyTotp: vi.fn(),
}))

import LoginPage from '../src/site/pages/LoginPage.vue'
import ResetPasswordPage from '../src/site/pages/ResetPasswordPage.vue'

const LEGACY_PNG = '/static/pic/brand-default.png'

function layout(brandType: 'default' | 'image' | 'text', brandImage: string): LayoutPayload {
  return {
    site: {
      name: 'yourtj',
      description: '',
      logo: '',
      favicon: '',
      brandType,
      brandText: 'yourtj',
      brandImage,
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
    sidebar: { main: [], resources: [], groups: [], categories: [], activeKey: 'topics' },
    footer: { links: [], primary: [] },
    unread: { notifications: false, messages: false, moderationReports: false },
    theme: { enabled: false, current: 'gf-light', themeColor: '#fbfdff' },
    insightFlareEnabled: false,
  }
}

const loginProps = {
  initialMode: 'login' as const,
  redirectUrl: '/',
  githubUrl: '/api/oauth/github',
  googleUrl: '',
  googleReady: false,
  termsOfServiceEnabled: true,
  privacyPolicyEnabled: true,
  allowedDomains: [],
}

function brandSrcs(wrapper: VueWrapper): string[] {
  return wrapper
    .findAll('img')
    .map((img) => img.attributes('src'))
    .filter((src) => src.startsWith('/static/pic/brand-default'))
}

describe('LoginPage default brand wordmark theme switch', () => {
  test('daily signup quota message code resolves to a friendly registration error', () => {
    const fallback = 'Registration failed'
    expect(resolveApiMessage({ messageCode: 'auth.register.dailyQuota' }, fallback)).not.toBe(fallback)
  })

  afterEach(() => {
    setThemePreference('light')
  })

  test('legacy residual brandImage with brandType=default uses dark variant in dark theme', async () => {
    setThemePreference('dark')
    const wrapper = mount(LoginPage, {
      props: { layout: layout('default', LEGACY_PNG), props: loginProps },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()
    const srcs = brandSrcs(wrapper)
    expect(srcs.length).toBeGreaterThan(0)
    expect(new Set(srcs)).toEqual(new Set(['/static/pic/brand-default-dark.webp']))
    wrapper.unmount()
  })

  test('brandType=default with no brandImage uses light variant in light theme', async () => {
    setThemePreference('light')
    const wrapper = mount(LoginPage, {
      props: { layout: layout('default', ''), props: loginProps },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()
    const srcs = brandSrcs(wrapper)
    expect(srcs.length).toBeGreaterThan(0)
    expect(new Set(srcs)).toEqual(new Set(['/static/pic/brand-default.webp']))
    wrapper.unmount()
  })

  test('brandType=image keeps admin custom image regardless of theme', async () => {
    setThemePreference('dark')
    const wrapper = mount(LoginPage, {
      props: { layout: layout('image', '/file/img/custom-brand.png'), props: loginProps },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()
    const srcs = wrapper.findAll('img').map((img) => img.attributes('src'))
    expect(srcs).toContain('/file/img/custom-brand.png')
    expect(brandSrcs(wrapper)).toEqual([])
    wrapper.unmount()
  })

  test('brandType=image with unsafe brandImage falls back to themed default', async () => {
    setThemePreference('dark')
    const wrapper = mount(LoginPage, {
      props: { layout: layout('image', 'javascript:alert(1)'), props: loginProps },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()
    const srcs = brandSrcs(wrapper)
    expect(srcs.length).toBeGreaterThan(0)
    expect(new Set(srcs)).toEqual(new Set(['/static/pic/brand-default-dark.webp']))
    wrapper.unmount()
  })

  test('disabled privacy policy is not linked or required by the registration agreement', async () => {
    const wrapper = mount(LoginPage, {
      props: {
        layout: layout('default', ''),
        props: { ...loginProps, initialMode: 'register', privacyPolicyEnabled: false },
      },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    expect(wrapper.find('a[href="/terms"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/privacy"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('I have read and agree to the terms')
    expect(wrapper.text()).not.toContain('I have read and agree to the terms and privacy policy')
    wrapper.unmount()
  })

  test('disabled terms of service is not linked or required by the registration agreement', async () => {
    const wrapper = mount(LoginPage, {
      props: {
        layout: layout('default', ''),
        props: { ...loginProps, initialMode: 'register', termsOfServiceEnabled: false },
      },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    expect(wrapper.find('a[href="/terms"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/privacy"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('I have read and agree to the privacy policy')
    expect(wrapper.text()).not.toContain('I have read and agree to the terms and privacy policy')
    wrapper.unmount()
  })

  test('single allowed email domain is rendered as a fixed suffix', async () => {
    const wrapper = mount(LoginPage, {
      props: {
        layout: layout('default', ''),
        props: { ...loginProps, initialMode: 'register', allowedDomains: ['tongji.edu.cn'] },
      },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    expect(wrapper.find('#register-email').attributes('type')).toBe('text')
    expect(wrapper.text()).toContain('@tongji.edu.cn')
    expect(wrapper.findComponent(SiteSelect).exists()).toBe(false)
    wrapper.unmount()
  })

  test('multiple allowed domains compose the selected suffix before registering', async () => {
    const registerMock = vi.mocked(register)
    registerMock.mockResolvedValue('ok')
    const wrapper = mount(LoginPage, {
      props: {
        layout: layout('default', ''),
        props: {
          ...loginProps,
          initialMode: 'register',
          termsOfServiceEnabled: false,
          privacyPolicyEnabled: false,
          allowedDomains: ['tongji.edu.cn', 'alumni.tongji.edu.cn'],
        },
      },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    await wrapper.find('input[autocomplete="username"]').setValue('walker')
    await wrapper.find('#register-email').setValue('student')
    wrapper.findComponent(SiteSelect).vm.$emit('update:modelValue', 'alumni.tongji.edu.cn')
    await wrapper.vm.$nextTick()
    const passwords = wrapper.findAll('input[autocomplete="new-password"]')
    await passwords[0].setValue('secret123')
    await passwords[1].setValue('secret123')
    await wrapper.find('input[placeholder="Captcha"]').setValue('abcd')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(registerMock).toHaveBeenCalledWith('walker', 'student@alumni.tongji.edu.cn', 'secret123', 'test', 'abcd', expect.any(String), '')
    wrapper.unmount()
  })

  test('empty allowed domain list keeps the normal email input', async () => {
    const wrapper = mount(LoginPage, {
      props: { layout: layout('default', ''), props: { ...loginProps, initialMode: 'register' } },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    expect(wrapper.find('#register-email').attributes('type')).toBe('email')
    expect(wrapper.findComponent(SiteSelect).exists()).toBe(false)
    wrapper.unmount()
  })
})

describe('ResetPasswordPage default brand wordmark theme switch', () => {
  afterEach(() => {
    setThemePreference('light')
  })

  test('legacy residual brandImage with brandType=default uses dark variant in dark theme', async () => {
    setThemePreference('dark')
    const wrapper = mount(ResetPasswordPage, {
      props: { layout: layout('default', LEGACY_PNG), props: { token: 't' } },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()
    const srcs = brandSrcs(wrapper)
    expect(srcs.length).toBeGreaterThan(0)
    expect(new Set(srcs)).toEqual(new Set(['/static/pic/brand-default-dark.webp']))
    wrapper.unmount()
  })

  test('brandType=default with no brandImage uses light variant in light theme', async () => {
    setThemePreference('light')
    const wrapper = mount(ResetPasswordPage, {
      props: { layout: layout('default', ''), props: { token: 't' } },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()
    const srcs = brandSrcs(wrapper)
    expect(srcs.length).toBeGreaterThan(0)
    expect(new Set(srcs)).toEqual(new Set(['/static/pic/brand-default.webp']))
    wrapper.unmount()
  })

  test('brandType=image keeps admin custom image regardless of theme', async () => {
    setThemePreference('dark')
    const wrapper = mount(ResetPasswordPage, {
      props: { layout: layout('image', '/file/img/custom-brand.png'), props: { token: 't' } },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()
    expect(wrapper.findAll('img').map((img) => img.attributes('src'))).toContain('/file/img/custom-brand.png')
    expect(brandSrcs(wrapper)).toEqual([])
    wrapper.unmount()
  })
})
