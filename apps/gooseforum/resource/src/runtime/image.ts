import { i18n } from './i18n'
import type CompressorType from 'compressorjs'

export interface ImageProcessResult {
  file: File
  converted: boolean
  originalSize: number
  newSize: number
}

export function supportsWebP(): boolean {
  const canvas = document.createElement('canvas')
  canvas.width = 1
  canvas.height = 1
  return canvas.toDataURL('image/webp').startsWith('data:image/webp')
}

export function validateImageFile(file: File, maxSize = 10 * 1024 * 1024): string | null {
  if (!file.type.startsWith('image/')) return i18n.global.t('image.selectFile')
  if (file.size > maxSize) return i18n.global.t('image.maxSize', { size: (maxSize / 1024 / 1024).toFixed(0) })
  return null
}

export async function processImageFile(file: File, quality = 0.85): Promise<ImageProcessResult> {
  const originalSize = file.size
  // GIF 不进压缩管线：canvas 无法编码 GIF（Safari 静默回退导出静态 PNG，
  // Chrome 成功时也会丢掉动画帧），原样上传保留动图。
  if (file.type === 'image/gif') {
    return { file, converted: false, originalSize, newSize: originalSize }
  }
  const targetType = supportsWebP() ? 'image/webp' : file.type || 'image/jpeg'
  const shouldConvert = targetType === 'image/webp' && !file.type.includes('webp')

  try {
    const converted = await compressImage(file, targetType, quality)
    return {
      file: converted,
      converted: shouldConvert || converted.size !== originalSize,
      originalSize,
      newSize: converted.size,
    }
  } catch (error) {
    console.warn(i18n.global.t('image.optimizeFailed'), error)
    return {
      file,
      converted: false,
      originalSize,
      newSize: originalSize,
    }
  }
}

export async function createSquareAvatarFile(file: File, size = 400, quality = 0.86): Promise<File> {
  const bitmap = await createBitmap(file)
  const sourceSize = Math.min(bitmap.width, bitmap.height)
  const sx = Math.floor((bitmap.width - sourceSize) / 2)
  const sy = Math.floor((bitmap.height - sourceSize) / 2)

  const canvas = document.createElement('canvas')
  canvas.width = size
  canvas.height = size
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error(i18n.global.t('image.processFailed'))

  ctx.imageSmoothingEnabled = true
  ctx.imageSmoothingQuality = 'high'
  ctx.drawImage(bitmap, sx, sy, sourceSize, sourceSize, 0, 0, size, size)

  const blob = await canvasToBlob(canvas, file.type || 'image/png', quality)
  return new File([blob], file.name, {
    type: blob.type || file.type,
    lastModified: Date.now(),
  })
}

export async function canvasToImageFile(
  canvas: HTMLCanvasElement,
  filename: string,
  mimeType = supportsWebP() ? 'image/webp' : 'image/jpeg',
  quality = 0.86,
): Promise<File> {
  const blob = await canvasToBlob(canvas, mimeType, quality)
  return new File([blob], filename.replace(/\.[^/.]+$/, mimeType === 'image/webp' ? '.webp' : '.jpg'), {
    type: mimeType,
    lastModified: Date.now(),
  })
}

// imageTypeExtension 把图片 MIME 映射为后端上传白名单内的扩展名；
// canvas 实际只会产出 png/jpeg/webp，其余条目仅为完备性。
function imageTypeExtension(mimeType: string): string {
  switch (mimeType) {
    case 'image/png':
      return '.png'
    case 'image/gif':
      return '.gif'
    case 'image/webp':
      return '.webp'
    case 'image/bmp':
      return '.bmp'
    default:
      return '.jpg'
  }
}

async function compressImage(file: File, mimeType: string, quality: number): Promise<File> {
  const { default: Compressor } = await import('compressorjs') as { default: typeof CompressorType }
  return new Promise((resolve, reject) => {
    new Compressor(file, {
      quality,
      mimeType,
      checkOrientation: true,
      convertSize: 0,
      success: (result) => {
        // 输出以压缩器实际产物为准：convertTypes 改道（PNG→JPEG）或目标
        // 编码不被 canvas 支持时（如 Safari 的 WebP/GIF），实际字节类型可能
        // 与请求的 mimeType 不同。文件名扩展名与 File.type 必须跟随实际
        // 产物，否则上传 init 的 filename↔contentType 严格校验会拒绝。
        const actualType = result.type || mimeType
        if (result instanceof File && result.type === actualType) {
          resolve(result)
          return
        }
        const baseName = file.name.replace(/\.[^/.]+$/, '')
        resolve(new File([result], `${baseName}${imageTypeExtension(actualType)}`, {
          type: actualType,
          lastModified: Date.now(),
        }))
      },
      error: (error) => {
        reject(new Error(i18n.global.t('image.compressFailed', { message: error.message })))
      },
    })
  })
}

async function createBitmap(file: File): Promise<ImageBitmap> {
  if ('createImageBitmap' in window) {
    return createImageBitmap(file, { imageOrientation: 'from-image' })
  }

  const url = URL.createObjectURL(file)
  try {
    const image = await new Promise<HTMLImageElement>((resolve, reject) => {
      const img = new Image()
      img.onload = () => resolve(img)
      img.onerror = () => reject(new Error(i18n.global.t('image.readFailed')))
      img.src = url
    })
    const canvas = document.createElement('canvas')
    canvas.width = image.naturalWidth
    canvas.height = image.naturalHeight
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error(i18n.global.t('image.processFailed'))
    ctx.drawImage(image, 0, 0)
    return createImageBitmap(canvas)
  } finally {
    URL.revokeObjectURL(url)
  }
}

function canvasToBlob(canvas: HTMLCanvasElement, mimeType: string, quality: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (blob) resolve(blob)
        else reject(new Error(i18n.global.t('image.encodeFailed')))
      },
      mimeType,
      quality,
    )
  })
}
