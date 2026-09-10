import { onBeforeUnmount, ref, watch } from 'vue'
import { uploadAvatar } from '@/runtime/api'
import { i18n } from '@/runtime/i18n'
import { canvasToImageFile, validateImageFile } from '@/runtime/image'

interface AvatarCropUploadOptions {
  initialAvatarUrl: string
  onStatus: (message: string) => void
  onError: (message: string) => void
}

// 知乎式头像编辑：选择图片 → AvatarImageEditor 所见即所得调整（拖拽/缩放）→
// 输出 300×300 主图 + 96×96 缩略图上传。
export function useAvatarCropUpload(options: AvatarCropUploadOptions) {
  const avatarInput = ref<HTMLInputElement | null>(null)
  const uploadingAvatar = ref(false)
  const avatarCropOpen = ref(false)
  const avatarUrl = ref(options.initialAvatarUrl)
  const avatarImageUrl = ref('')
  const cropError = ref('')
  let cropOpener: HTMLElement | null = null

  watch(avatarCropOpen, (open) => {
    if (open) {
      document.addEventListener('keydown', handleCropKeydown)
    } else {
      document.removeEventListener('keydown', handleCropKeydown)
    }
  })

  onBeforeUnmount(() => {
    document.removeEventListener('keydown', handleCropKeydown)
    revokeImageUrl()
  })

  function handleCropKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape' && avatarCropOpen.value) {
      event.preventDefault()
      closeAvatarCrop()
    }
  }

  // 打开文件选择器
  function chooseAvatar() {
    cropOpener = document.activeElement instanceof HTMLElement ? document.activeElement : null
    avatarInput.value?.click()
  }

  // 选中文件：类型/大小校验后打开编辑器
  async function handleAvatarChange(event: Event) {
    const input = event.target as HTMLInputElement
    const file = input.files?.[0]
    if (!file) return
    // 先清空 input，允许同一文件再次选择
    if (input) input.value = ''

    const validationError = validateImageFile(file, 5 * 1024 * 1024)
    if (validationError) {
      options.onError(validationError)
      return
    }

    revokeImageUrl()
    avatarImageUrl.value = URL.createObjectURL(file)
    avatarCropOpen.value = true
  }

  function closeAvatarCrop() {
    avatarCropOpen.value = false
    cropError.value = ''
    revokeImageUrl()
    cropOpener?.focus()
    cropOpener = null
  }

  // 编辑器保存：canvas → 300/96 WebP 上传，返回新头像 URL
  async function saveAvatarFromCanvas(canvas: HTMLCanvasElement): Promise<string | null> {
    uploadingAvatar.value = true
    cropError.value = ''
    try {
      const avatar300 = await canvasToImageFile(canvas, 'avatar.webp', undefined, 0.86)
      // 缩略图：从同一画布再缩到 96
      const thumbCanvas = document.createElement('canvas')
      thumbCanvas.width = 96
      thumbCanvas.height = 96
      const thumbContext = thumbCanvas.getContext('2d')
      if (!thumbContext) throw new Error(i18n.global.t('avatarCrop.processFailed'))
      thumbContext.imageSmoothingEnabled = true
      thumbContext.imageSmoothingQuality = 'high'
      thumbContext.drawImage(canvas, 0, 0, 96, 96)
      const avatar96 = await canvasToImageFile(thumbCanvas, 'avatar_medium.webp', undefined, 0.9)

      const url = await uploadAvatar([avatar300, avatar96])
      avatarUrl.value = url
      return url
    } catch (err) {
      const message = err instanceof Error ? err.message : i18n.global.t('api.avatarUploadFailed')
      cropError.value = message
      options.onError(message)
      return null
    } finally {
      uploadingAvatar.value = false
    }
  }

  function revokeImageUrl() {
    if (avatarImageUrl.value) URL.revokeObjectURL(avatarImageUrl.value)
    avatarImageUrl.value = ''
  }

  return {
    avatarInput,
    uploadingAvatar,
    avatarCropOpen,
    avatarUrl,
    avatarImageUrl,
    cropError,
    chooseAvatar,
    handleAvatarChange,
    closeAvatarCrop,
    saveAvatarFromCanvas,
  }
}
