// @vitest-environment happy-dom
import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import ActivationEmailAction from '../src/site/components/ActivationEmailAction.vue'
import { getActivationStatus, resendActivationEmail } from '../src/runtime/api'
vi.mock('../src/runtime/api', () => ({ getActivationStatus: vi.fn(), resendActivationEmail: vi.fn() }))
const ready = { currentEmail: 'old@example.test', pendingEmail: 'new@example.test', targetEmail: 'new@example.test', targetKind: 'pending', state: 'ready', remainingToday: 3, dailyLimit: 3, retryAfterSeconds: 0 } as const
const mounted: ReturnType<typeof mount>[] = []
function render() { const wrapper = mount(ActivationEmailAction, { props: { accountId: 7 }, global: { plugins: [i18n] } }); mounted.push(wrapper); return wrapper }
afterEach(() => { mounted.splice(0).forEach(w => w.unmount()); vi.clearAllMocks(); vi.restoreAllMocks(); vi.useRealTimers() })
describe('authoritative activation status', () => {
  it('shows actual staged target, refreshes after sending and blocks cooldown', async () => {
    vi.mocked(getActivationStatus).mockResolvedValueOnce(ready).mockResolvedValueOnce({ ...ready, state: 'cooldown', remainingToday: 2, retryAfterSeconds: 60 })
    vi.mocked(resendActivationEmail).mockResolvedValue('任务已受理')
    const w = render(); await flushPromises()
    expect(w.text()).toContain('new@example.test'); expect(w.text()).toContain('3 / 3')
    await w.findAll('button')[0]!.trigger('click'); await flushPromises()
    expect(resendActivationEmail).toHaveBeenCalledTimes(1); expect(getActivationStatus).toHaveBeenCalledTimes(2)
    expect(w.text()).toContain('2 / 3'); expect(w.findAll('button')[0]!.attributes('disabled')).toBeDefined()
  })
  for (const state of ['verified', 'disabled', 'frozen', 'daily_limit', 'mail_unavailable', 'missing_email'] as const) {
    it(`blocks sending for ${state}`, async () => {
      vi.mocked(getActivationStatus).mockResolvedValue({ ...ready, state })
      const w = render(); await flushPromises()
      expect(w.findAll('button')[0]!.attributes('disabled')).toBeDefined()
      expect(resendActivationEmail).not.toHaveBeenCalled()
    })
  }
  it('uses elapsed time, rechecks expiry and never automatically sends', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] })
    let elapsed = 100
    vi.spyOn(performance, 'now').mockImplementation(() => elapsed)
    vi.mocked(getActivationStatus).mockResolvedValueOnce({ ...ready, state: 'cooldown', retryAfterSeconds: 2 }).mockResolvedValueOnce(ready)
    const w = render(); await flushPromises()
    vi.spyOn(Date, 'now').mockReturnValue(9999999999999)
    await vi.advanceTimersByTimeAsync(1000); await flushPromises()
    expect(getActivationStatus).toHaveBeenCalledTimes(1)
    expect(w.findAll('button')[0]!.attributes('disabled')).toBeDefined()
    elapsed += 2001
    await vi.advanceTimersByTimeAsync(1000); await flushPromises()
    expect(getActivationStatus).toHaveBeenCalledTimes(2)
    expect(w.findAll('button')[0]!.attributes('disabled')).toBeUndefined()
    expect(resendActivationEmail).not.toHaveBeenCalled()
  })
  it('does not accept an old account response', async () => {
    let finish!: (value: typeof ready) => void
    vi.mocked(getActivationStatus).mockReturnValueOnce(new Promise(resolve => { finish = resolve })).mockResolvedValueOnce({ ...ready, targetEmail: 'next@example.test' })
    const w = render(); await w.setProps({ accountId: 8 }); await flushPromises()
    finish(ready); await flushPromises()
    expect(w.text()).toContain('next@example.test'); expect(w.text()).not.toContain('new@example.test')
  })
  it('fails closed and allows a read-only retry', async () => {
    vi.mocked(getActivationStatus).mockRejectedValueOnce(new Error('暂不可用')).mockResolvedValueOnce(ready)
    const w = render(); await flushPromises()
    expect(w.findAll('button')[0]!.attributes('disabled')).toBeDefined()
    await w.findAll('button')[1]!.trigger('click'); await flushPromises()
    expect(w.findAll('button')[0]!.attributes('disabled')).toBeUndefined()
    expect(resendActivationEmail).not.toHaveBeenCalled()
  })
})
