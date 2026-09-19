import { afterEach, describe, expect, it, vi } from 'vitest'
import { i18n } from '../src/runtime/i18n'
import zh from '../src/locales/zh'
import en from '../src/locales/en'
import ja from '../src/locales/ja'
import de from '../src/locales/de'
import { calendarRulesAPI, teachingDateLabel } from '../src/runtime/calendar-rules-api'
import { campusAPI, CampusError } from '../src/runtime/campus-api'
afterEach(() => { vi.unstubAllGlobals(); i18n.global.locale.value = 'zh' })
describe('private campus client', () => {
  it('exports with the forum session, no caching, and cancellation', async () => {
    const fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ code: 0, result: { content: 'calendar' } }) })
    vi.stubGlobal('fetch', fetch)
    const controller = new AbortController()
    await campusAPI.exportCalendar(controller.signal)
    expect(fetch).toHaveBeenCalledWith('/api/campus/calendar-export?applyAdjustments=true', expect.objectContaining({ credentials: 'same-origin', cache: 'no-store', signal: controller.signal }))
    await campusAPI.exportCalendar(controller.signal, false)
    expect(fetch).toHaveBeenLastCalledWith('/api/campus/calendar-export?applyAdjustments=false', expect.any(Object))
  })
  it('never caches school records or exposes tokens', async () => {
    const payload = { key: 'grades', status: 'ready', rows: [] }
    const fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ code: 0, result: payload }) })
    vi.stubGlobal('fetch', fetch)
    expect(await campusAPI.dataset('grades')).toEqual(payload)
    expect(fetch).toHaveBeenCalledWith('/api/campus/data/grades', expect.objectContaining({ credentials: 'same-origin', cache: 'no-store' }))
  })
  it('keeps school authorization expiry separate from forum logout', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 409, json: async () => ({ code: 1, messageCode: 'campus.authorizationRequired' }) }))
    await expect(campusAPI.dataset('grades')).rejects.toBeInstanceOf(CampusError)
    await expect(campusAPI.dataset('grades')).rejects.toMatchObject({ code: 'campus.authorizationRequired' })
  })
  it('encodes message identifiers and never caches the body', async () => {
    const fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ code: 0, result: { content: '正文' } }) })
    vi.stubGlobal('fetch', fetch)
    await campusAPI.message('123?other=1')
    expect(fetch).toHaveBeenCalledWith('/api/campus/messages/123%3Fother%3D1', expect.objectContaining({ cache: 'no-store', credentials: 'same-origin' }))
  })
  it('sends the current generation when unlinking', async () => {
    const fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ code: 0, result: null }) })
    vi.stubGlobal('fetch', fetch)
    await campusAPI.unbind('current-binding')
    expect(fetch).toHaveBeenCalledWith('/api/campus/tongji/unbind', expect.objectContaining({ method: 'POST', body: JSON.stringify({ revision: 'current-binding' }) }))
  })
})

for (const [locale, messages] of Object.entries({ zh, en, ja, de })) {
  it(`localizes campus and rule errors in ${locale} without trusting server text`, async () => {
    i18n.global.setLocaleMessage(locale, messages)
    ;(i18n.global.locale as unknown as { value: string }).value = locale
    const fetch = vi.fn().mockResolvedValue({ ok: false, status: 409, json: async () => ({ code: 1, messageCode: 'campus.authorizationRequired' }) })
    vi.stubGlobal('fetch', fetch)
    await expect(campusAPI.dataset('grades')).rejects.toMatchObject({ code: 'campus.authorizationRequired', message: messages.campus.errorAuthorization })
    fetch.mockResolvedValue({ ok: false, status: 400, json: async () => ({ code: 1, messageCode: 'campus.rulesInvalid', params: { reason: '日期冲突' } }) })
    await expect(calendarRulesAPI.save({ revision: 'r', rules: { holidays: [], moves: [] } })).rejects.toThrow(locale === 'zh' ? '日期冲突' : messages.campus.errorRulesInvalid)
    expect(teachingDateLabel('2026-09-20', locale)).toBe(new Intl.DateTimeFormat(locale, { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', weekday: 'short' }).format(new Date('2026-09-20T12:00:00+08:00')))
    fetch.mockResolvedValue({ ok: false, status: 429, json: async () => ({ code: 1, messageCode: 'unknown', message: '不要显示' }) })
    await expect(campusAPI.status()).rejects.toThrow(messages.campus.errorLimited)
  })
}
