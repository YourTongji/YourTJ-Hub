import { afterEach, describe, expect, test, vi } from 'vitest'

// Node 测试环境无 document：mock i18n 为透传键（runtime/api → api-message → i18n）。
vi.mock('../src/runtime/i18n', () => ({
  i18n: {
    global: {
      t: (key: string) => key,
      te: () => false,
      locale: { value: 'zh' },
      getLocaleMessage: () => ({ serverMessages: { 'content.aiModeration.blocked': '内容未通过站点发布规则，请调整后重试。' } }),
    },
  },
}))

import { ApiResponseError, CHECKING_MESSAGE_CODE, createPost, PENDING_REVIEW_MESSAGE_CODE, pendingReviewMessage, sensitiveWordsFromError, submitTopic, submitTopicResult, updatePost } from '../src/runtime/api'
import { replayAiModeration, saveAiModerationSettings } from '../src/admin/runtime/api'
import type { AiModerationSettingsInput } from '../src/admin/types'

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

const topicInput = { topicId: 0, title: 't', content: 'c', categoryId: [1], topicStatus: 1, contentType: 3 }

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('publish pending-review signal (issue #975)', () => {
  test('submitTopicResult reports pendingReview from the success envelope messageCode', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ code: 0, result: 975, messageCode: PENDING_REVIEW_MESSAGE_CODE })))
    await expect(submitTopicResult(topicInput)).resolves.toEqual({ id: 975, pendingReview: true })

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ code: 0, result: 976 })))
    await expect(submitTopicResult(topicInput)).resolves.toEqual({ id: 976, pendingReview: false })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ code: 0, result: 977 })))
    await expect(submitTopic(topicInput)).resolves.toBe(977)
  })

  test('AI block surfaces a stable messageCode without sensitive-word highlights', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ code: 1, result: null, messageCode: 'content.aiModeration.blocked' })))
    const error = await submitTopic(topicInput).catch((err: unknown) => err)
    expect(error).toBeInstanceOf(ApiResponseError)
    expect((error as ApiResponseError).messageCode).toBe('content.aiModeration.blocked')
    expect((error as ApiResponseError).message).toBe('内容未通过站点发布规则，请调整后重试。')
    // 编辑器据此保留草稿：AI 拦截不携带 words，不误标敏感词高亮。
    expect(sensitiveWordsFromError(error)).toEqual([])
  })

  test('reply create and edit results carry pendingReview', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ code: 0, messageCode: PENDING_REVIEW_MESSAGE_CODE, result: { id: 5, postNo: 2, renderedContent: '' } })))
    await expect(createPost(1, 'content')).resolves.toMatchObject({ id: 5, pendingReview: true })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ code: 0, result: { id: 6, postNo: 3, renderedContent: '' } })))
    expect(await createPost(1, 'content')).not.toHaveProperty('pendingReview')

    const updated = { id: 7, content: 'x', renderedContent: '', updatedAt: '', lastEditorId: 1, lastEditedAt: '', revisionCount: 2 }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ code: 0, messageCode: PENDING_REVIEW_MESSAGE_CODE, result: updated })))
    await expect(updatePost(7, 'x')).resolves.toMatchObject({ id: 7, pendingReview: true })
  })

  test('check-after-publishing results are pending and flagged as checking', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ code: 0, result: 978, messageCode: CHECKING_MESSAGE_CODE })))
    const topic = await submitTopicResult(topicInput)
    expect(topic).toEqual({ id: 978, pendingReview: true, checking: true })
    // i18n 被 mock 为透传键：checking 走“正在自动检查”文案，其余走“已提交审核”。
    expect(pendingReviewMessage(topic)).toBe('api.checking')
    expect(pendingReviewMessage({})).toBe('api.pendingReview')

    vi.stubGlobal('fetch', vi.fn().mockImplementation(async () => jsonResponse({ code: 0, messageCode: CHECKING_MESSAGE_CODE, result: { id: 8, postNo: 4, renderedContent: '' } })))
    await expect(createPost(1, 'content')).resolves.toMatchObject({ id: 8, pendingReview: true, checking: true })
    const updated = { id: 9, content: 'x', renderedContent: '', updatedAt: '', lastEditorId: 1, lastEditedAt: '', revisionCount: 2 }
    vi.stubGlobal('fetch', vi.fn().mockImplementation(async () => jsonResponse({ code: 0, messageCode: CHECKING_MESSAGE_CODE, result: updated })))
    await expect(updatePost(9, 'x')).resolves.toMatchObject({ id: 9, pendingReview: true, checking: true })
  })
})

describe('admin AI moderation wire shape', () => {
  test('save wraps settings and replay sends candidate options', async () => {
    const fetchMock = vi.fn().mockImplementation(async () => jsonResponse({ code: 0, result: 'success' }))
    vi.stubGlobal('fetch', fetchMock)
    const settings = { enabled: true, mode: 'shadow', jevApiKey: 'sk-new', clearVisionApiKey: true } as AiModerationSettingsInput
    await saveAiModerationSettings(settings)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/admin/save-ai-moderation-settings')
    expect(JSON.parse(String(init.body))).toEqual({ settings: { enabled: true, mode: 'shadow', jevApiKey: 'sk-new', clearVisionApiKey: true } })

    await replayAiModeration()
    expect(JSON.parse(String((fetchMock.mock.calls[1] as [string, RequestInit])[1].body))).toEqual({})
  })
})
