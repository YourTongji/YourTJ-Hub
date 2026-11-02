// @vitest-environment happy-dom
// issue #643 回归：注册邮箱域名白名单为空（shipped default）时，后端存量
// 配置行缺键会把 allowedDomains 序列化为 null；LoginPage setup 曾直接解引用
// 导致整页白屏。修复后页面必须容错挂载，注册表单退化为自由邮箱输入。
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { register } from '../src/runtime/api'
import { i18n } from '../src/runtime/i18n'
import type { LayoutPayload, LoginPageProps } from '@gooseforum/client'

vi.mock('../src/runtime/api', () => ({
  getCaptcha: vi.fn(async () => ({ captchaId: 'captcha-id', captchaImg: 'data:image/png;base64,x' })),
  getPasskeyStatus: vi.fn(async () => ({ enabled: false })),
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
import { getCaptcha, forgotPassword } from '../src/runtime/api'

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

function buildProps(allowedDomains: unknown): LoginPageProps {
  return {
    initialMode: 'register',
    redirectUrl: '/',
    githubUrl: '/api/auth/github',
    githubReady: true,
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
  test('registration rules are visible and associated before typing', () => {
    const wrapper=mount(LoginPage,{props:{layout,props:buildProps([])},global:{plugins:[i18n]}})
    expect(wrapper.get('input[autocomplete="username"]').attributes('aria-describedby')).toBe('register-username-rules')
    expect(wrapper.get('#register-username-rules').text()).toContain('6–32')
    expect(wrapper.get('input[autocomplete="new-password"]').attributes('aria-describedby')).toBe('register-password-rules')
    expect(wrapper.get('#register-password-rules').text()).toContain('6–64')
    expect(wrapper.findAll('input').every(input=>!input.element.value)).toBe(true)
    wrapper.unmount()
  })

})


describe('LoginPage captcha refresh ownership (#1220 F14)', () => {
  const challenge = (id: string) => ({ captchaId: id, captchaImg: `data:image/png;base64,${id}` })
  beforeEach(() => { vi.mocked(getCaptcha).mockReset(); vi.mocked(forgotPassword).mockClear() })
  const render = (initialMode: 'register' | 'forgot' = 'register') => mount(LoginPage, { props: { layout, props: { ...buildProps([]), initialMode } }, global: { plugins: [i18n] } })
  const code = (w: ReturnType<typeof render>) => w.find(`input[placeholder="${i18n.global.t('auth.captcha')}"]`)
  const refresh = (w: ReturnType<typeof render>) => w.findAll('button').find(b => b.find('img.gf-captcha-image').exists())!

  test('only a successful new challenge clears the old answer', async () => {
    vi.mocked(getCaptcha).mockResolvedValueOnce(challenge('first')).mockResolvedValueOnce(challenge('second'))
    const w = render(); await flushPromises()
    await code(w).setValue('OLD-CODE')
    await refresh(w).trigger('click'); await flushPromises()
    expect(w.find('.gf-captcha-image').attributes('src')).toContain('second')
    expect((code(w).element as HTMLInputElement).value).toBe('')
    w.unmount()
  })
  test('failed refresh retains both the usable image and answer', async () => {
    vi.mocked(getCaptcha).mockResolvedValueOnce(challenge('first')).mockRejectedValueOnce(new Error('offline'))
    const w = render('forgot'); await flushPromises()
    await w.find('input[type="email"]').setValue('owner@example.test')
    await code(w).setValue('OLD-CODE')
    await refresh(w).trigger('click'); await flushPromises()
    expect(w.find('.gf-captcha-image').attributes('src')).toContain('first')
    expect((code(w).element as HTMLInputElement).value).toBe('OLD-CODE')
    await w.find('form').trigger('submit'); await flushPromises()
    expect(forgotPassword).toHaveBeenCalledWith('owner@example.test', 'first', 'OLD-CODE', '')
    w.unmount()
  })
  test('refresh is single flight and neither button nor submit sends mismatched input', async () => {
    let finish!: (value: ReturnType<typeof challenge>) => void
    vi.mocked(getCaptcha).mockResolvedValueOnce(challenge('first')).mockReturnValueOnce(new Promise(resolve => { finish = resolve }))
    const w = render('forgot'); await flushPromises()
    await w.find('input[type="email"]').setValue('owner@example.test')
    await code(w).setValue('OLD-CODE')
    const button = refresh(w)
    await button.trigger('click')
    await button.trigger('click')
    await w.find('form').trigger('submit'); await flushPromises()
    expect(getCaptcha).toHaveBeenCalledTimes(2)
    expect(forgotPassword).not.toHaveBeenCalled()
    finish(challenge('second')); await flushPromises(); w.unmount()
  })
  test('an initial failure offers a named retry without accepting a missing challenge', async () => {
    vi.mocked(getCaptcha).mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(challenge('retry'))
    const w = render('forgot'); await flushPromises()
    await w.find('input[type="email"]').setValue('owner@example.test')
    await code(w).setValue('not-a-challenge')
    await w.find('form').trigger('submit'); await flushPromises()
    expect(forgotPassword).not.toHaveBeenCalled()
    const retry = w.find('button[aria-busy]')
    expect(retry.attributes('aria-label')).toBe(i18n.global.t('auth.refreshCaptcha'))
    await retry.trigger('click'); await flushPromises()
    expect(w.find('.gf-captcha-image').attributes('src')).toContain('retry')
    expect((code(w).element as HTMLInputElement).value).toBe('')
    w.unmount()
  })
  test('a late response from the previous mode cannot replace the new challenge', async () => {
    let finish!: (value: ReturnType<typeof challenge>) => void
    vi.mocked(getCaptcha).mockReturnValueOnce(new Promise(resolve => { finish = resolve })).mockResolvedValueOnce(challenge('new-mode'))
    const w = render()
    await w.findAll('.gf-segmented-item')[0]!.trigger('click'); await flushPromises()
    expect(getCaptcha).toHaveBeenCalledTimes(2)
    finish(challenge('old-mode')); await flushPromises()
    expect(w.find('.gf-captcha-image').attributes('src')).toContain('new-mode')
    w.unmount()
  })
})
describe('password registration availability (#1220 F12)', () => {
  for (const state of ['disabled', 'daily_limit', 'mail_unavailable', 'unavailable'] as const) {
    test(`shows ${state} before collecting a registration`, async () => {
      const w = mount(LoginPage, { props: { layout, props: { ...buildProps([]), passwordSignupState: state } }, global: { plugins: [i18n] } })
      await flushPromises()
      expect(w.find('input[type="email"]').exists()).toBe(false)
      expect(w.find('form button[type="submit"]').exists()).toBe(false)
      expect(w.find('[data-testid="signup-availability"]').exists()).toBe(true)
      expect(w.findAll('.gf-segmented-item')).toHaveLength(2)
      w.unmount()
    })
  }
})

// #1220 F04: exercise the actual form, including a domain that differs from
// the selector's initial value. Abort after the request so no navigation occurs.
describe('registration email paste', () => {
  for (const { input, domains, expected } of [
    { input: 'student', domains: ['tongji.edu.cn'], expected: 'student@tongji.edu.cn' },
    { input: 'student@TONGJI.EDU.CN', domains: ['tongji.edu.cn'], expected: 'student@tongji.edu.cn' },
    { input: '  student@s.TONGJI.edu.cn  ', domains: ['tongji.edu.cn', 's.tongji.edu.cn'], expected: 'student@s.tongji.edu.cn' },
    { input: 'student@example.test', domains: [], expected: 'student@example.test' },
    ...['student@example.test', 'student@@tongji.edu.cn', '@tongji.edu.cn', 'student@', 'stu dent@tongji.edu.cn'].map(input => ({ input, domains: ['tongji.edu.cn'], expected: null })),
  ]) test(`${input} with ${domains.length} allowed domains`, async () => {
    vi.mocked(register).mockClear().mockRejectedValue(new Error('request observed'))
    const wrapper = mount(LoginPage, {
      props: { layout, props: { ...buildProps(domains), captchaRequired: false } },
      global: { plugins: [i18n] },
    })
    try {
      await flushPromises()
      await wrapper.get('input[autocomplete="username"]').setValue('new-user')
      await wrapper.get('#register-email').setValue(input)
      for (const password of wrapper.findAll('input[autocomplete="new-password"]')) await password.setValue('password123')
      await wrapper.get('form').trigger('submit')
      await flushPromises()
      if (expected == null) {
        expect(register).not.toHaveBeenCalled()
        expect((wrapper.get('#register-email').element as HTMLInputElement).value).toBe(input.trim())
      } else {
        expect(register).toHaveBeenCalledOnce()
        expect(vi.mocked(register).mock.calls[0][1]).toBe(expected)
        if (domains.length) expect((wrapper.get('#register-email').element as HTMLInputElement).value).toBe(expected.split('@')[0])
      }
    } finally { wrapper.unmount() }
  })
})

for (const redirect of ['/publish?id=42', '//evil.test']) test(`availability recheck preserves a locally selected register mode and safe destination ${redirect}`, async () => {
  const assign = vi.spyOn(window.location, 'assign').mockImplementation(() => {})
  const w = mount(LoginPage, { props: { layout, props: { ...buildProps([]), initialMode: 'login', passwordSignupState: 'disabled', redirectUrl: redirect } }, global: { plugins: [i18n] } })
  try {
    await flushPromises()
    await w.findAll('.gf-segmented-item')[1].trigger('click')
    await w.get('[data-testid="signup-availability"] button').trigger('click')
    expect(assign).toHaveBeenCalledOnce()
    const url = new URL(assign.mock.calls[0][0], 'https://forum.example.test')
    expect(url.pathname).toBe('/login')
    expect(url.searchParams.get('register')).toBe('true')
    expect(url.searchParams.get('redirect')).toBe(redirect.startsWith('//') ? '/' : redirect)
  } finally { w.unmount(); assign.mockRestore() }
})
