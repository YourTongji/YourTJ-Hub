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

import { uploadImage } from '../src/runtime/api'

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function imageFile(): File {
  return new File(['fake-image-bytes'], 'file.png', { type: 'image/png' })
}

const directInit = {
  code: 0,
  result: {
    mode: 'direct',
    name: 'pending/2026/09/02/9f1c2d3e-0000-4000-8000-000000000000.png',
    upload: {
      url: 'https://objects.example.com/forum-bucket',
      method: 'POST',
      fields: { key: 'pending/2026/09/02/9f1c2d3e-0000-4000-8000-000000000000.png', policy: 'xxx' },
      expiresAt: '2026-09-02T12:00:00Z',
    },
  },
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('uploadImage direct/proxy wire shape', () => {
  test('proxy mode falls back to the multipart /file/img-upload upload', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ code: 0, result: { mode: 'proxy' } }))
      .mockResolvedValueOnce(jsonResponse({ code: 0, result: { url: '/file/img/2026/09/02/proxy.png' } }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(uploadImage(imageFile())).resolves.toBe('/file/img/2026/09/02/proxy.png')

    const [initUrl, initInit] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(initUrl).toBe('/file/img-upload/init')
    expect(JSON.parse(String(initInit.body))).toEqual({
      filename: 'file.png',
      contentType: 'image/png',
      size: 'fake-image-bytes'.length,
    })
    const [proxyUrl] = fetchMock.mock.calls[1] as [string]
    expect(proxyUrl).toBe('/file/img-upload')
  })

  test('direct mode uploads to the bucket then completes', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(directInit))
      .mockResolvedValueOnce(new Response(null, { status: 200 }))
      .mockResolvedValueOnce(
        jsonResponse({
          code: 0,
          result: { url: '/file/img/2026/09/02/9f1c2d3e-0000-4000-8000-000000000000.png', filename: 'file.png', size: 68 },
        }),
      )
    vi.stubGlobal('fetch', fetchMock)

    await expect(uploadImage(imageFile())).resolves.toBe('/file/img/2026/09/02/9f1c2d3e-0000-4000-8000-000000000000.png')

    const [bucketUrl, bucketInit] = fetchMock.mock.calls[1] as [string, RequestInit]
    expect(bucketUrl).toBe('https://objects.example.com/forum-bucket')
    const formData = bucketInit.body as FormData
    expect(formData.get('key')).toBe('pending/2026/09/02/9f1c2d3e-0000-4000-8000-000000000000.png')
    expect(formData.get('policy')).toBe('xxx')
    expect(formData.get('file')).toBeInstanceOf(File)

    const [completeUrl, completeInit] = fetchMock.mock.calls[2] as [string, RequestInit]
    expect(completeUrl).toBe('/file/img-upload/complete')
    expect(JSON.parse(String(completeInit.body))).toEqual({
      name: 'pending/2026/09/02/9f1c2d3e-0000-4000-8000-000000000000.png',
    })
  })

  test('bucket upload failure aborts the pending object and rejects', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(directInit))
      .mockResolvedValueOnce(new Response('Forbidden', { status: 403 }))
      .mockResolvedValueOnce(jsonResponse({ code: 0, result: true }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(uploadImage(imageFile())).rejects.toThrow('HTTP 403')

    const [abortUrl, abortInit] = fetchMock.mock.calls[2] as [string, RequestInit]
    expect(abortUrl).toBe('/file/img-upload/abort')
    expect(JSON.parse(String(abortInit.body))).toEqual({
      name: 'pending/2026/09/02/9f1c2d3e-0000-4000-8000-000000000000.png',
    })
  })

  test('network error on the bucket upload still tries complete (object may be in flight)', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(directInit))
      .mockRejectedValueOnce(new TypeError('Failed to fetch'))
      .mockResolvedValueOnce(
        jsonResponse({
          code: 0,
          result: { url: '/file/img/2026/09/02/9f1c2d3e-0000-4000-8000-000000000000.png', filename: 'file.png', size: 68 },
        }),
      )
    vi.stubGlobal('fetch', fetchMock)

    await expect(uploadImage(imageFile())).resolves.toBe('/file/img/2026/09/02/9f1c2d3e-0000-4000-8000-000000000000.png')
    expect(fetchMock.mock.calls[2][0]).toBe('/file/img-upload/complete')
  })

  test('complete business failure aborts the pending object', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(directInit))
      .mockResolvedValueOnce(new Response(null, { status: 200 }))
      .mockResolvedValueOnce(jsonResponse({ code: 1, messageCode: 'upload.invalidImage' }))
      .mockResolvedValueOnce(jsonResponse({ code: 0, result: true }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(uploadImage(imageFile())).rejects.toThrow('api.imageUploadFailed')
    expect(fetchMock.mock.calls[3][0]).toBe('/file/img-upload/abort')
  })

  test('transient complete failure retries once before succeeding', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(directInit))
      .mockResolvedValueOnce(new Response(null, { status: 200 }))
      .mockRejectedValueOnce(new TypeError('Failed to fetch'))
      .mockResolvedValueOnce(
        jsonResponse({
          code: 0,
          result: { url: '/file/img/2026/09/02/9f1c2d3e-0000-4000-8000-000000000000.png', filename: 'file.png', size: 68 },
        }),
      )
    vi.stubGlobal('fetch', fetchMock)

    await expect(uploadImage(imageFile())).resolves.toBe('/file/img/2026/09/02/9f1c2d3e-0000-4000-8000-000000000000.png')
    expect(fetchMock).toHaveBeenCalledTimes(4)
  })
})