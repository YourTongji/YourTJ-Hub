import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { uploadImageFile } from '@/runtime/api'
import { i18n } from '@/runtime/i18n'
import { canvasToImageFile, validateImageFile } from '@/runtime/image'

interface CoverCropUploadOptions {
  onStatus: (message: string) => void
  onError: (message: string) => void
  /** 非致命提示（如尺寸不足），短暂横幅后自动隐去；未提供时回退 onError */
  onWarning?: (message: string) => void
}

export const COVER_ASPECT_RATIO = 5
export const COVER_MOBILE_RATIO = 3
/** 封面源图最小像素：对齐输出 1600×320 的 0.75×，避免明显模糊与非横图 */
export const COVER_MIN_WIDTH = 1200
export const COVER_MIN_HEIGHT = 240

// 知乎式封面编辑：选择图片 → CoverImageEditor 所见即所得调整（拖拽/缩放）→
// 按当前视图输出 1600x320 → WebP 上传到 /file/img-upload。
export function useCoverCropUpload(options: CoverCropUploadOptions) {
  const coverInput = ref<HTMLInputElement | null>(null)
  const coverCropOpen = ref(false)
  const coverImageUrl = ref('')
  const uploadingCover = ref(false)
  const coverCropError = ref('')
  let cropOpener: HTMLElement | null = null

  watch(coverCropOpen, (open) => {
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

  // 封面编辑浮层打开期间按 Esc 关闭
  function handleCropKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape' && coverCropOpen.value) {
      event.preventDefault()
      closeCoverCrop()
    }
  }

  // 打开文件选择器
  function chooseCover() {
    cropOpener = document.activeElement instanceof HTMLElement ? document.activeElement : null
    coverInput.value?.click()
  }

  // 读取图片自然尺寸（不依赖 DOM）
  function readImageNaturalSize(file: File): Promise<{ width: number; height: number }> {
    return new Promise((resolve, reject) => {
      const objectUrl = URL.createObjectURL(file)
      const image = new Image()
      image.onload = () => {
        const width = image.naturalWidth
        const height = image.naturalHeight
        URL.revokeObjectURL(objectUrl)
        resolve({ width, height })
      }
      image.onerror = () => {
        URL.revokeObjectURL(objectUrl)
        reject(new Error(i18n.global.t('image.selectFile')))
      }
      image.src = objectUrl
    })
  }

  // 选中文件：类型/大小 + 最小宽高校验；不合格用短暂 flash 提示（知乎式横幅）
  async function handleCoverChange(event: Event) {
    const input = event.target as HTMLInputElement
    const file = input.files?.[0]
    if (!file) return
    // 先清空 input，允许同一文件再次选择
    if (input) input.value = ''

    const validationError = validateImageFile(file, 10 * 1024 * 1024)
    if (validationError) {
      options.onError(validationError)
      return
    }

    try {
      const { width, height } = await readImageNaturalSize(file)
      if (width < COVER_MIN_WIDTH || height < COVER_MIN_HEIGHT) {
        // 知乎式短暂横幅：warning 自动隐去（GlobalFlash ~5.2s）
        const message = i18n.global.t('settings.cover.minSizeRequired', {
          width: COVER_MIN_WIDTH,
          height: COVER_MIN_HEIGHT,
        })
        if (options.onWarning) options.onWarning(message)
        else options.onError(message)
        return
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : i18n.global.t('image.selectFile')
      options.onError(message)
      return
    }

    revokeImageUrl()
    coverImageUrl.value = URL.createObjectURL(file)
    coverCropOpen.value = true
  }

  function closeCoverCrop() {
    coverCropOpen.value = false
    revokeImageUrl()
    cropOpener?.focus()
    cropOpener = null
  }

  // 编辑器保存：canvas → WebP 上传，返回封面 URL
  async function saveCoverFromCanvas(canvas: HTMLCanvasElement): Promise<string | null> {
    uploadingCover.value = true
    coverCropError.value = ''
    try {
      const coverFile = await canvasToImageFile(canvas, 'cover.webp', undefined, 0.9)
      const coverUrl = await uploadImageFile(coverFile, coverFile.name)
      // 成功文案由页面层 pushMediaFlash 统一发出，避免双提示
      return coverUrl
    } catch (err) {
      const message = err instanceof Error ? err.message : i18n.global.t('api.imageUploadFailed')
      coverCropError.value = message
      options.onError(message)
      return null
    } finally {
      uploadingCover.value = false
    }
  }

  function revokeImageUrl() {
    if (coverImageUrl.value) URL.revokeObjectURL(coverImageUrl.value)
    coverImageUrl.value = ''
  }

  return {
    coverInput,
    coverCropOpen,
    coverImageUrl,
    uploadingCover,
    coverCropError,
    chooseCover,
    handleCoverChange,
    closeCoverCrop,
    saveCoverFromCanvas,
  }
}
