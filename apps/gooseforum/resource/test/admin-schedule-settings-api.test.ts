import { afterEach, describe, expect, test, vi } from 'vitest'

// Node 测试环境无 document：mock i18n 为透传键（api.ts → adminText → i18n）。
vi.mock('../src/runtime/i18n', () => ({
  i18n: {
    global: {
      t: (key: string) => key,
    },
  },
}))

import { saveScheduleSettings } from '../src/admin/runtime/api'
import type { ScheduleSettings } from '../src/admin/types'

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('admin schedule settings wire shape', () => {
  test('saveScheduleSettings wraps the payload under { settings } per the contract', async () => {
    // 契约 AdminSaveScheduleSettingsRequest：body = { settings: { sectionTimes: [...] } }。
    // 曾发送裸 { sectionTimes }，后端绑定零值 Settings 后静默存空表（review P1）。
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ code: 0, msg: 'ok', result: null }))
    vi.stubGlobal('fetch', fetchMock)

    const settings: ScheduleSettings = {
      sectionTimes: [{ section: 3, start: '10:00', end: '10:45' }],
    }
    await saveScheduleSettings(settings)

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/admin/save-schedule-settings')
    expect(JSON.parse(String(init.body))).toEqual({
      settings: { sectionTimes: [{ section: 3, start: '10:00', end: '10:45' }] },
    })
  })
})
