import { ref } from 'vue'
import { ApiResponseError, getCaptcha } from '@/runtime/api'

/**
 * 验证码挑战状态：当写操作返回 common.captchaRequired 时，
 * 加载验证码图片并让用户输入答案，随后携带 captchaId/captchaCode 重试。
 */
export function useCaptchaChallenge() {
  const captchaRequired = ref(false)
  const captchaId = ref('')
  const captchaImg = ref('')
  const captchaCode = ref('')
  const captchaLoading = ref(false)

  async function loadCaptcha() {
    captchaLoading.value = true
    captchaCode.value = ''
    try {
      const captcha = await getCaptcha()
      captchaId.value = captcha.captchaId
      captchaImg.value = captcha.captchaImg
      captchaRequired.value = true
    } finally {
      captchaLoading.value = false
    }
  }

  function clearCaptcha() {
    captchaRequired.value = false
    captchaId.value = ''
    captchaImg.value = ''
    captchaCode.value = ''
  }

  /** 若错误为「需要验证码」，加载验证码并返回 true（调用方应展示挑战 UI）。 */
  function challengeFromError(err: unknown): boolean {
    if (err instanceof ApiResponseError && err.messageCode === 'common.captchaRequired') {
      void loadCaptcha()
      return true
    }
    return false
  }

  return {
    captchaRequired,
    captchaId,
    captchaImg,
    captchaCode,
    captchaLoading,
    loadCaptcha,
    clearCaptcha,
    challengeFromError,
  }
}
