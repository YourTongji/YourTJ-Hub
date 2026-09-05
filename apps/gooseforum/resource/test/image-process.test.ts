import { afterEach, describe, expect, test, vi } from 'vitest'

// 复现 Safari 插图上传被拒（「文件内容不是有效的图片格式」）的压缩错标链路：
// compressorjs 的 convertTypes（默认含 PNG）+ convertSize=0 会把 PNG 输入内部
// 改道为 JPEG 输出；Safari canvas 不支持 WebP/GIF 编码时会静默回退导出 PNG。
// 返回的 File 必须以实际产物声明 type 与扩展名，否则 /file/img-upload/init 的
// filename↔contentType 严格校验（以及 complete 的内容嗅探）会拒绝上传。
const { compressorState } = vi.hoisted(() => ({
  compressorState: { outputType: 'image/jpeg', calls: 0, fail: false },
}))

vi.mock('compressorjs', () => ({
  default: class MockCompressor {
    constructor(
      _file: File,
      options: { success: (blob: Blob) => void; error: (error: Error) => void },
    ) {
      compressorState.calls += 1
      queueMicrotask(() => {
        if (compressorState.fail) {
          options.error(new Error('mock compress failed'))
          return
        }
        options.success(new Blob(['mock-encoded-bytes'], { type: compressorState.outputType }))
      })
    }
  },
}))

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

import { processImageFile } from '../src/runtime/image'

function stubCanvasEncode(supportsWebPEncode: boolean) {
  vi.stubGlobal('document', {
    createElement: () => ({
      width: 1,
      height: 1,
      toDataURL: (type: string) =>
        supportsWebPEncode && type === 'image/webp'
          ? 'data:image/webp;mock'
          : 'data:image/png;base64,mock',
    }),
  })
}

function fileWith(name: string, type: string): File {
  return new File(['original-bytes'], name, { type })
}

afterEach(() => {
  compressorState.outputType = 'image/jpeg'
  compressorState.calls = 0
  compressorState.fail = false
  vi.unstubAllGlobals()
})

describe('processImageFile', () => {
  test('Safari：PNG 输入被压缩器改道为 JPEG 时，File 类型与扩展名跟随实际产物', async () => {
    stubCanvasEncode(false)
    compressorState.outputType = 'image/jpeg'
    const png = fileWith('photo.png', 'image/png')

    const result = await processImageFile(png)

    expect(result.file.type).toBe('image/jpeg')
    expect(result.file.name).toBe('photo.jpg')
  })

  test('Safari：WebP 输入在压缩时回退导出 PNG 时，类型与扩展名跟随实际产物', async () => {
    stubCanvasEncode(false)
    compressorState.outputType = 'image/png'
    const webp = fileWith('pic.webp', 'image/webp')

    const result = await processImageFile(webp)

    expect(result.file.type).toBe('image/png')
    expect(result.file.name).toBe('pic.png')
  })

  test('Safari：GIF 原样上传，不经过压缩器（保住动画，避免回退静态 PNG）', async () => {
    stubCanvasEncode(false)
    const gif = fileWith('anim.gif', 'image/gif')

    const result = await processImageFile(gif)

    expect(result.file).toBe(gif)
    expect(result.converted).toBe(false)
    expect(compressorState.calls).toBe(0)
  })

  test('支持 WebP 的浏览器：PNG 转 WebP 后类型与扩展名一致', async () => {
    stubCanvasEncode(true)
    compressorState.outputType = 'image/webp'
    const png = fileWith('photo.png', 'image/png')

    const result = await processImageFile(png)

    expect(result.file.type).toBe('image/webp')
    expect(result.file.name).toBe('photo.webp')
  })

  test('压缩器类型未变时保持原扩展名（JPEG 常规路径不回归）', async () => {
    stubCanvasEncode(false)
    compressorState.outputType = 'image/jpeg'
    const jpg = fileWith('cat.jpg', 'image/jpeg')

    const result = await processImageFile(jpg)

    expect(result.file.type).toBe('image/jpeg')
    expect(result.file.name).toBe('cat.jpg')
  })

  test('压缩失败时回退原文件', async () => {
    stubCanvasEncode(false)
    compressorState.fail = true
    const png = fileWith('photo.png', 'image/png')

    const result = await processImageFile(png)

    expect(result.file).toBe(png)
    expect(result.converted).toBe(false)
  })
})
