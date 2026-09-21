// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import TongjiOnboarding from '../src/site/components/TongjiOnboarding.vue'
import { forgotPassword, getCaptcha, saveUserName } from '../src/runtime/api'
vi.mock('../src/runtime/api', () => ({
  saveUserName: vi.fn(),
  getCaptcha: vi.fn(),
  forgotPassword: vi.fn(),
}))
const render = (returnTo = '/api/oauth/authorize?request=abc') => mount(TongjiOnboarding, {
  props: { username: 'student', email: '1234567@tongji.edu.cn', returnTo },
  global: { plugins: [i18n] },
})
beforeEach(() => {
  vi.resetAllMocks()
  vi.mocked(getCaptcha).mockResolvedValue({ captchaId: 'challenge', captchaImg: 'data:image/png;base64,AA==' })
})
describe('Tongji first login guide', () => {
  test('skip and continue preserve local OIDC continuation without setting credentials', () => {
    const wrapper = render()
    expect(wrapper.findAll('a').map(a => a.attributes('href'))).toEqual(['/api/oauth/authorize?request=abc', '/api/oauth/authorize?request=abc'])
    expect(saveUserName).not.toHaveBeenCalled()
    expect(forgotPassword).not.toHaveBeenCalled()
  })
  test.each(['https://evil.test', '//evil.test', '/\\evil.test', 'javascript:alert(1)', '/\nevil.test'])('rejects unsafe return target %s', (target) => {
    expect(render(target).find('a').attributes('href')).toBe('/')
  })
  test('saves username, keeps validation failures visible and allows retry', async () => {
    const wrapper = render()
    vi.mocked(saveUserName).mockRejectedValueOnce(new Error('Already taken'))
    await wrapper.find('input').setValue(' chosen ')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(saveUserName).toHaveBeenCalledWith('chosen')
    expect(wrapper.find('[role=alert]').text()).toBe('Already taken')
    vi.mocked(saveUserName).mockResolvedValueOnce(true)
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(wrapper.find('[role=alert]').exists()).toBe(false)
    expect(wrapper.find('[role=status]').exists()).toBe(true)
  })
  test('password setup uses the bound email and captcha, retaining errors on refresh', async () => {
    const wrapper = render()
    await wrapper.find('section button').trigger('click')
    await flushPromises()
    vi.mocked(forgotPassword).mockRejectedValueOnce(new Error('Incorrect captcha'))
    await wrapper.find('section input').setValue('abcd')
    await wrapper.find('section form').trigger('submit')
    await flushPromises()
    expect(forgotPassword).toHaveBeenCalledWith('1234567@tongji.edu.cn', 'challenge', 'abcd')
    expect(wrapper.find('[role=alert]').text()).toBe('Incorrect captcha')
    expect(getCaptcha).toHaveBeenCalledTimes(2)
    expect((wrapper.find('section input').element as HTMLInputElement).value).toBe('')
    vi.mocked(forgotPassword).mockResolvedValueOnce('Check your email')
    await wrapper.find('section input').setValue('efgh')
    await wrapper.find('section form').trigger('submit')
    await flushPromises()
    expect(wrapper.find('[role=status]').text()).toBe('Check your email')
    expect(wrapper.find('section form').exists()).toBe(false)
  })
  test('captcha failure is visible and retryable', async () => {
    vi.mocked(getCaptcha).mockRejectedValueOnce(new Error('Offline'))
    const wrapper = render()
    await wrapper.find('section button').trigger('click')
    await flushPromises()
    expect(wrapper.find('[role=alert]').text()).toBe('Offline')
    expect(wrapper.find('section button[type=submit]').attributes('disabled')).toBeDefined()
    await wrapper.find('section button[type=button]').trigger('click')
    await flushPromises()
    expect(wrapper.find('img').exists()).toBe(true)
  })
})
