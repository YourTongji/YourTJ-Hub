import { nextTick, onBeforeUnmount, ref } from 'vue'
import type Cropper from 'cropperjs'
import { uploadImageFile } from '@/runtime/api'
import { i18n } from '@/runtime/i18n'
import { canvasToImageFile, validateImageFile } from '@/runtime/image'

interface CoverCropUploadOptions {
  onStatus: (message: string) => void
  onError: (message: string) => void
}

export const COVER_ASPECT_RATIO = 4
export const COVER_OUTPUT_WIDTH = 1600
export const COVER_OUTPUT_HEIGHT = 400

// 个人资料封面：选择图片 → 4:1 选区裁剪 → 上传 /file/img-upload → 返回封面 URL。
// 与头像裁剪（useAvatarCropUpload）共用 cropperjs 模板，仅选区比例与输出尺寸不同。
export function useCoverCropUpload(options: CoverCropUploadOptions) {
  const uploadingCover = ref(false)
  const coverInput = ref<HTMLInputElement | null>(null)
  const coverCropperImage = ref<HTMLImageElement | null>(null)
  const coverCropModalOpen = ref(false)
  const coverCropImageUrl = ref('')
  const coverCropPreviewUrl = ref('')
  const coverCropError = ref('')
  const coverSourceFile = ref<File | null>(null)
  let cropper: Cropper | undefined
  let cropperContainer: HTMLElement | undefined
  let cropPreviewFrame = 0

  onBeforeUnmount(() => {
    destroyCropper()
    revokeCropImageUrl()
  })

  function chooseCover() {
    coverInput.value?.click()
  }

  async function handleCoverChange(event: Event) {
    const file = (event.target as HTMLInputElement).files?.[0]
    if (!file) return
    const validationError = validateImageFile(file, 10 * 1024 * 1024)
    if (validationError) return options.onError(validationError)

    openCropModal(file)
    if (coverInput.value) coverInput.value.value = ''
  }

  function openCropModal(file: File) {
    destroyCropper()
    revokeCropImageUrl()
    coverCropError.value = ''
    coverSourceFile.value = file
    coverCropImageUrl.value = URL.createObjectURL(file)
    coverCropModalOpen.value = true
    void nextTick(() => {
      void initCropper()
    })
  }

  async function initCropper() {
    const image = coverCropperImage.value
    if (!image) return
    const { default: Cropper } = await import('cropperjs')
    if (!coverCropModalOpen.value || coverCropperImage.value !== image) return

    cropper = new Cropper(image, {
      template: `
        <cropper-canvas background>
          <cropper-image translatable scalable rotatable></cropper-image>
          <cropper-shade hidden></cropper-shade>
          <cropper-handle action="select" plain></cropper-handle>
          <cropper-selection aspect-ratio="${COVER_ASPECT_RATIO}" movable resizable zoomable outlined>
            <cropper-grid role="grid" bordered covered></cropper-grid>
            <cropper-crosshair centered></cropper-crosshair>
            <cropper-handle action="move" theme-color="rgba(37, 99, 235, 0.35)"></cropper-handle>
            <cropper-handle action="n-resize"></cropper-handle>
            <cropper-handle action="e-resize"></cropper-handle>
            <cropper-handle action="s-resize"></cropper-handle>
            <cropper-handle action="w-resize"></cropper-handle>
            <cropper-handle action="ne-resize"></cropper-handle>
            <cropper-handle action="nw-resize"></cropper-handle>
            <cropper-handle action="se-resize"></cropper-handle>
            <cropper-handle action="sw-resize"></cropper-handle>
          </cropper-selection>
        </cropper-canvas>
      `,
    })
    void resetCropSelectionToMaxWidth()
    const container = cropper.container as HTMLElement
    cropperContainer = container
    container.addEventListener('pointerup', scheduleCropPreview)
    container.addEventListener('wheel', scheduleCropPreview, { passive: true })
    container.addEventListener('keyup', scheduleCropPreview)
  }

  // 初始选区覆盖图片最大可用宽度（按 4:1 高度向下兼容），
  // 让用户默认拿到最宽的封面区域，再手动微调。
  async function resetCropSelectionToMaxWidth() {
    const cropperImageElement = cropper?.getCropperImage()
    const cropperCanvas = cropper?.getCropperCanvas()
    const selection = cropper?.getCropperSelection()
    if (!cropperImageElement || !cropperCanvas || !selection) return

    try {
      await cropperImageElement.$ready()
    } catch {
      return
    }

    window.requestAnimationFrame(() => {
      const canvasRect = cropperCanvas.getBoundingClientRect()
      const imageRect = cropperImageElement.getBoundingClientRect()
      if (imageRect.width <= 0 || imageRect.height <= 0) return

      const width = imageRect.width
      const height = width / COVER_ASPECT_RATIO
      if (height > imageRect.height) {
        const adjustedHeight = imageRect.height
        const adjustedWidth = adjustedHeight * COVER_ASPECT_RATIO
        const x = imageRect.left - canvasRect.left + (imageRect.width - adjustedWidth) / 2
        const y = imageRect.top - canvasRect.top
        selection.$change(x, y, adjustedWidth, adjustedHeight, COVER_ASPECT_RATIO, true)
      } else {
        const x = imageRect.left - canvasRect.left
        const y = imageRect.top - canvasRect.top + (imageRect.height - height) / 2
        selection.$change(x, y, width, height, COVER_ASPECT_RATIO, true)
      }
      void updateCropPreview()
    })
  }

  function closeCropModal() {
    coverCropModalOpen.value = false
    coverSourceFile.value = null
    coverCropError.value = ''
    destroyCropper()
    revokeCropImageUrl()
  }

  function destroyCropper() {
    window.cancelAnimationFrame(cropPreviewFrame)
    cropPreviewFrame = 0
    if (cropperContainer) {
      cropperContainer.removeEventListener('pointerup', scheduleCropPreview)
      cropperContainer.removeEventListener('wheel', scheduleCropPreview)
      cropperContainer.removeEventListener('keyup', scheduleCropPreview)
      cropperContainer = undefined
    }
    cropper?.destroy()
    cropper = undefined
    coverCropPreviewUrl.value = ''
  }

  function revokeCropImageUrl() {
    if (coverCropImageUrl.value) URL.revokeObjectURL(coverCropImageUrl.value)
    coverCropImageUrl.value = ''
  }

  async function uploadCroppedCover(): Promise<string> {
    if (!cropper || !coverSourceFile.value) {
      throw new Error(i18n.global.t('avatarCrop.selectArea'))
    }

    uploadingCover.value = true
    coverCropError.value = ''
    try {
      const selection = cropper.getCropperSelection()
      if (!selection) throw new Error(i18n.global.t('avatarCrop.selectArea'))
      const canvas = await selection.$toCanvas({
        width: COVER_OUTPUT_WIDTH,
        height: COVER_OUTPUT_HEIGHT,
        beforeDraw(context) {
          context.imageSmoothingEnabled = true
          context.imageSmoothingQuality = 'high'
        },
      })
      const coverFile = await canvasToImageFile(canvas, 'cover.webp', undefined, 0.86)
      const coverUrl = await uploadImageFile(coverFile, coverFile.name)
      closeCropModal()
      options.onStatus(i18n.global.t('avatarCrop.updated'))
      return coverUrl
    } catch (err) {
      const message = err instanceof Error ? err.message : i18n.global.t('api.imageUploadFailed')
      coverCropError.value = message
      options.onError(message)
      throw err
    } finally {
      uploadingCover.value = false
    }
  }

  function scheduleCropPreview() {
    window.cancelAnimationFrame(cropPreviewFrame)
    cropPreviewFrame = window.requestAnimationFrame(() => {
      void updateCropPreview()
    })
  }

  async function updateCropPreview() {
    const selection = cropper?.getCropperSelection()
    if (!selection) return
    try {
      const canvas = await selection.$toCanvas({
        width: COVER_OUTPUT_WIDTH / 4,
        height: COVER_OUTPUT_HEIGHT / 4,
        beforeDraw(context) {
          context.imageSmoothingEnabled = true
          context.imageSmoothingQuality = 'high'
        },
      })
      coverCropPreviewUrl.value = canvas.toDataURL('image/webp', 0.82)
    } catch {
      coverCropPreviewUrl.value = ''
    }
  }

  return {
    uploadingCover,
    coverInput,
    coverCropperImage,
    coverCropModalOpen,
    coverCropImageUrl,
    coverCropPreviewUrl,
    coverCropError,
    chooseCover,
    handleCoverChange,
    closeCropModal,
    uploadCroppedCover,
  }
}
