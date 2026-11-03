// @vitest-environment happy-dom
import { expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import { resolveApiMessage } from '../src/runtime/api-message'
import type { LayoutPayload, LoginPageProps } from '@gooseforum/client'

vi.mock('../src/runtime/api', () => ({
  getCaptcha: vi.fn(async () => ({ captchaId: 'captcha-id', captchaImg: 'data:image/png;base64,x' })),
  getPasskeyStatus: vi.fn(async () => ({ enabled: false })),
  login: vi.fn(async () => ({})),
  register: vi.fn(async () => ''),
  forgotPassword: vi.fn(async () => ''),
  verifyTotp: vi.fn(async () => undefined),
}))
vi.mock('../src/runtime/flash-message', () => ({ queueFlashMessage: vi.fn() }))
vi.mock('../src/runtime/site-theme', async () => {
  const { ref } = await import('vue')
  return { useSiteTheme: () => ({ isDark: ref(false) }), setThemePreference: vi.fn() }
})

import { register } from '../src/runtime/api'
import LoginPage from '../src/site/pages/LoginPage.vue'

const layout: LayoutPayload = {
  site: { name: 'yourtj', description: '', logo: '', favicon: '', brandType: 'text', brandText: 'yourtj', brandImage: '' },
  viewer: { id: 0, username: '', email: '', avatarUrl: '', isAuthenticated: false, canAccessAdmin: false, isModerator: false, requiresEmailVerification: false, adminPermissions: [] },
  sidebar: { categories: [], activeKey: '' },
  footer: { links: [], primary: [] },
  unread: { notifications: false, messages: false },
  posting: { maxTitleLength: 100 },
  theme: { enabled: false, current: 'gf-light', themeColor: '' },
  umamiEnabled: false,
}

const props: LoginPageProps = {
  initialMode: 'register',
  redirectUrl: '/',
  githubUrl: '',
  githubReady: false,
  googleUrl: '',
  googleReady: false,
  termsOfServiceEnabled: false,
  privacyPolicyEnabled: false,
  allowedDomains: [],
  oauthNotice: false,
  captchaRequired: false,
}

function occupancyError(message: string, messageCode: string) {
  return Object.assign(new Error(message), { messageCode })
}

test.each(['zh', 'en', 'ja', 'de'] as const)('registration hangs occupancy copy on the matching field in %s', async (locale) => {
  await setLocale(locale)
  const usernameMessage = resolveApiMessage({ messageCode: 'auth.username.exists' }, 'fallback')
  const emailMessage = resolveApiMessage({ messageCode: 'auth.email.exists' }, 'fallback')
  const generic = resolveApiMessage({ messageCode: 'auth.register.failed' }, 'fallback')
  expect(usernameMessage).not.toBe(emailMessage)
  expect(usernameMessage).not.toBe(generic)
  expect(emailMessage).not.toBe(generic)

  const wrapper = mount(LoginPage, { props: { layout, props }, global: { plugins: [i18n] } })
  await flushPromises()
  const fill = async () => {
    await wrapper.find('input[autocomplete="username"]').setValue('taken-user')
    await wrapper.find('#register-email').setValue('student@example.test')
    const passwords = wrapper.findAll('input[autocomplete="new-password"]')
    await passwords[0].setValue('password123')
    await passwords[1].setValue('password123')
  }

  vi.mocked(register).mockRejectedValueOnce(occupancyError(usernameMessage, 'auth.username.exists'))
  await fill()
  await wrapper.find('form').trigger('submit')
  await flushPromises()
  expect(wrapper.get('#register-username-error').text()).toBe(usernameMessage)
  expect(wrapper.find('#register-email-error').exists()).toBe(false)
  expect(wrapper.find('#auth-form-error').exists()).toBe(false)
  expect(wrapper.get('input[autocomplete="username"]').attributes('aria-invalid')).toBe('true')
  expect(wrapper.get('input[autocomplete="username"]').attributes('aria-describedby')).toContain('register-username-error')

  vi.mocked(register).mockRejectedValueOnce(occupancyError(emailMessage, 'auth.email.exists'))
  await fill()
  await wrapper.find('form').trigger('submit')
  await flushPromises()
  expect(wrapper.get('#register-email-error').text()).toBe(emailMessage)
  expect(wrapper.find('#register-username-error').exists()).toBe(false)
  expect(wrapper.find('#auth-form-error').exists()).toBe(false)
  expect(wrapper.get('#register-email').attributes('aria-invalid')).toBe('true')
  expect(wrapper.get('#register-email').attributes('aria-describedby')).toBe('register-email-error')

  vi.mocked(register).mockRejectedValueOnce(occupancyError(generic, 'auth.register.failed'))
  await fill()
  await wrapper.find('form').trigger('submit')
  await flushPromises()
  expect(wrapper.get('#auth-form-error').text()).toBe(generic)
  expect(wrapper.find('#register-username-error').exists()).toBe(false)
  expect(wrapper.find('#register-email-error').exists()).toBe(false)
  wrapper.unmount()
  await setLocale('zh')
})
