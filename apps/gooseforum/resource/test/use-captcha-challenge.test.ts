import { describe, expect, test, vi } from 'vitest'
import { ApiResponseError, getCaptcha } from '../src/runtime/api'
import { useCaptchaChallenge } from '../src/site/composables/useCaptchaChallenge'

vi.mock('../src/runtime/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../src/runtime/api')>()
  return { ...actual, getCaptcha: vi.fn(async () => ({ captchaId: 'id', captchaImg: 'image' })) }
})

describe('useCaptchaChallenge publish explanation', () => {
  test('shows copy only for publishing guard actions and clears it with the challenge', () => {
    const challenge = useCaptchaChallenge()
    const publishError = new ApiResponseError('captcha', 'common.captchaRequired', undefined, { action: 'topic.write' })
    const replyError = new ApiResponseError('captcha', 'common.captchaRequired', undefined, { action: 'post.create' })
    const loginError = new ApiResponseError('captcha', 'common.captchaRequired', undefined, { action: 'login' })

    expect(challenge.challengeFromError(publishError)).toBe(true)
    expect(challenge.showPublishCaptchaExplanation.value).toBe(true)
    expect(challenge.challengeFromError(replyError)).toBe(true)
    expect(challenge.showPublishCaptchaExplanation.value).toBe(true)
    expect(challenge.challengeFromError(loginError)).toBe(true)
    expect(challenge.showPublishCaptchaExplanation.value).toBe(false)
    expect(challenge.challengeFromError(new Error('other error'))).toBe(false)
    challenge.showPublishCaptchaExplanation.value = true
    challenge.clearCaptcha()
    expect(challenge.showPublishCaptchaExplanation.value).toBe(false)
    expect(getCaptcha).toHaveBeenCalledTimes(3)
  })
})
