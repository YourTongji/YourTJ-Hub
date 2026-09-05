import { afterEach, describe, expect, test, vi } from 'vitest'

// Node 测试环境无 document：mock i18n 为透传键（api.ts → resolveApiMessage → i18n）。
// te 返回 false 使失败路径回退到 fallback 文案，避免触碰真实 locale 消息表。
vi.mock('../src/runtime/i18n', () => ({
  i18n: {
    global: {
      t: (key: string) => key,
      te: () => false,
      locale: { value: 'zh' },
      getLocaleMessage: () => ({}),
    },
  },
}))

import { ApiResponseError, followUser, submitTopic } from '../src/runtime/api'

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

const emailRequired403 = {
  result: null,
  code: 1,
  messageCode: 'permission.emailRequired',
  params: { action: '写入', actionCode: 'write' },
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('CheckWritableAccount 403 envelope parsing (issue #404/#415)', () => {
  test('submitTopic surfaces permission.emailRequired instead of a generic HTTP 403', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(emailRequired403, 403))
    vi.stubGlobal('fetch', fetchMock)

    await expect(
      submitTopic({ topicId: 0, title: 't', content: 'c', categoryId: [1], topicStatus: 0 }),
    ).rejects.toMatchObject<ApiResponseError>({
      name: 'ApiResponseError',
      messageCode: 'permission.emailRequired',
    })
  })

  test('followUser surfaces permission.emailRequired instead of a generic HTTP 403', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(emailRequired403, 403))
    vi.stubGlobal('fetch', fetchMock)

    await expect(followUser(42, true)).rejects.toMatchObject<ApiResponseError>({
      name: 'ApiResponseError',
      messageCode: 'permission.emailRequired',
    })
  })

  test('a non-envelope 403 keeps the generic HTTP error', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('denied', { status: 403 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(followUser(42, true)).rejects.toThrow('HTTP 403')
  })
})
