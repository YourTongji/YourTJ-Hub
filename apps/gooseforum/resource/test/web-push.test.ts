// @vitest-environment happy-dom
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import vm from 'node:vm'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import {
  currentPushSubscription,
  disableWebPush,
  enableWebPush,
  fetchPushConfig,
  isWebPushSupported,
} from '../src/runtime/web-push'

// Web Push 运行时（runtime/web-push.ts）与静态 Service Worker（static/sw.js）
// 的行为测试。sw.js 是 classic 纯 push worker（无 fetch handler），用 vm 沙箱
// 直接执行其源码并驱动 push / notificationclick 事件，避免引入 SW 运行时依赖。

const SW_SOURCE = readFileSync(resolve(__dirname, '../static/sw.js'), 'utf8')
const ENDPOINT = 'https://push.example.test/sub/abc'
// 65B（含 0x04 前缀）base64url P-256 未压缩点 —— 真实长度，验证透传解码
const APP_KEY = 'BHhk8ysQtKmNMOliOxJf3p6perMqxI-EXQB-vgekUMHQ22pn1RlCmh0OmJFVpxEhBGw3nmaRdeaas9Y2UofSndg'

/** 装配满足 isWebPushSupported 的浏览器环境（可按需覆盖）。 */
function installPushCapabilities(overrides: {
  serviceWorker?: {
    register?: (...args: unknown[]) => Promise<unknown>
    getRegistration?: (...args: unknown[]) => Promise<unknown>
  }
  pushManager?: unknown
} = {}) {
  vi.stubGlobal('isSecureContext', true)
  vi.stubGlobal('PushManager', overrides.pushManager ?? class PushManager {})
  vi.stubGlobal('Notification', class Notification {})
  const serviceWorker = {
    register: overrides.serviceWorker?.register ?? vi.fn(async () => ({ pushManager: {} })),
    getRegistration: overrides.serviceWorker?.getRegistration ?? vi.fn(async () => null),
  }
  Object.defineProperty(navigator, 'serviceWorker', { value: serviceWorker, configurable: true })
  return serviceWorker as {
    register: ReturnType<typeof vi.fn>
    getRegistration: ReturnType<typeof vi.fn>
  }
}

function jsonResponse(body: unknown, ok = true) {
  return { ok, json: async () => body } as Response
}

/** 按请求 URL 分发的 fetch mock（fetchPushConfig 与 subscribe/unsubscribe 共用）。 */
function stubFetchByUrl(routes: Record<string, Response>) {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input)
    for (const [prefix, response] of Object.entries(routes)) {
      if (url.startsWith(prefix)) return response
    }
    return jsonResponse({ code: 1 }, false)
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

beforeEach(() => {
  installPushCapabilities()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('isWebPushSupported', () => {
  test('false in non-secure context', () => {
    vi.stubGlobal('isSecureContext', false)
    expect(isWebPushSupported()).toBe(false)
  })

  test('false without Service Worker / PushManager / Notification', () => {
    Reflect.deleteProperty(window, 'PushManager')
    Reflect.deleteProperty(navigator, 'serviceWorker')
    expect(isWebPushSupported()).toBe(false)
  })

  test('true when all prerequisites exist', () => {
    expect(isWebPushSupported()).toBe(true)
  })
})

describe('fetchPushConfig', () => {
  test('returns configured=false when instance disabled', async () => {
    stubFetchByUrl({ '/api/forum/push/config': jsonResponse({ code: 0, result: { configured: false } }) })
    await expect(fetchPushConfig()).resolves.toEqual({ configured: false })
  })

  test('returns applicationServerKey when enabled', async () => {
    stubFetchByUrl({ '/api/forum/push/config': jsonResponse({ code: 0, result: { configured: true, applicationServerKey: APP_KEY } }) })
    await expect(fetchPushConfig()).resolves.toEqual({ configured: true, applicationServerKey: APP_KEY })
  })

  test('throws network on HTTP error or error code', async () => {
    stubFetchByUrl({ '/api/forum/push/config': jsonResponse({ code: 1 }, false) })
    await expect(fetchPushConfig()).rejects.toMatchObject({ code: 'network' })
    stubFetchByUrl({ '/api/forum/push/config': jsonResponse({ code: 1 }) })
    await expect(fetchPushConfig()).rejects.toMatchObject({ code: 'network' })
  })
})

describe('currentPushSubscription', () => {
  test('returns null when unsupported', async () => {
    vi.stubGlobal('isSecureContext', false)
    await expect(currentPushSubscription()).resolves.toBeNull()
  })

  test('returns null without a registration', async () => {
    const serviceWorker = installPushCapabilities({
      serviceWorker: { register: vi.fn(), getRegistration: vi.fn(async () => null) },
    })
    await expect(currentPushSubscription()).resolves.toBeNull()
    expect(serviceWorker.getRegistration).toHaveBeenCalledWith('/sw.js')
  })

  test('returns the active subscription', async () => {
    const subscription = { endpoint: ENDPOINT }
    installPushCapabilities({
      serviceWorker: {
        register: vi.fn(),
        getRegistration: vi.fn(async () => ({
          pushManager: { getSubscription: vi.fn(async () => subscription) },
        })),
      },
    })
    await expect(currentPushSubscription()).resolves.toBe(subscription)
  })
})

describe('enableWebPush', () => {
  test('throws unsupported before any network call', async () => {
    vi.stubGlobal('isSecureContext', false)
    await expect(enableWebPush('zh')).rejects.toMatchObject({ code: 'unsupported' })
  })

  test('throws unconfigured when instance disabled', async () => {
    stubFetchByUrl({ '/api/forum/push/config': jsonResponse({ code: 0, result: { configured: false } }) })
    await expect(enableWebPush('zh')).rejects.toMatchObject({ code: 'unconfigured' })
  })

  test('registers worker and persists the browser subscription', async () => {
    const pushManager = {
      subscribe: vi.fn(async () => ({
        toJSON: () => ({
          endpoint: ENDPOINT,
          keys: { p256dh: 'p256dh-value', auth: 'auth-value' },
        }),
      })),
    }
    const serviceWorker = installPushCapabilities({
      serviceWorker: {
        register: vi.fn(async () => ({ pushManager })),
        getRegistration: vi.fn(),
      },
    })
    stubFetchByUrl({
      '/api/forum/push/config': jsonResponse({ code: 0, result: { configured: true, applicationServerKey: APP_KEY } }),
      '/api/forum/push/subscribe': jsonResponse({ code: 0 }),
    })

    await enableWebPush('ja')

    expect(serviceWorker.register).toHaveBeenCalledWith('/sw.js', { updateViaCache: 'none' })
    expect(pushManager.subscribe).toHaveBeenCalledWith({
      userVisibleOnly: true,
      applicationServerKey: expect.any(Uint8Array),
    })
    const key = pushManager.subscribe.mock.calls[0][0].applicationServerKey as Uint8Array
    expect(key).toHaveLength(65) // P-256 未压缩点
    const subscribeCall = vi.mocked(fetch).mock.calls.find(([url]) => String(url) === '/api/forum/push/subscribe')
    expect(subscribeCall).toBeDefined()
    const init = subscribeCall![1] as RequestInit
    expect(JSON.parse(String(init.body))).toEqual({
      subscription: { endpoint: ENDPOINT, keys: { p256dh: 'p256dh-value', auth: 'auth-value' } },
      lang: 'ja',
    })
  })

  test('maps NotAllowedError to permission-denied', async () => {
    installPushCapabilities({
      serviceWorker: {
        register: vi.fn(async () => ({
          pushManager: { subscribe: vi.fn(async () => { throw new DOMException('denied', 'NotAllowedError') }) },
        })),
        getRegistration: vi.fn(),
      },
    })
    stubFetchByUrl({ '/api/forum/push/config': jsonResponse({ code: 0, result: { configured: true, applicationServerKey: APP_KEY } }) })
    await expect(enableWebPush('zh')).rejects.toMatchObject({ code: 'permission-denied' })
  })

  test('maps other subscribe failures to subscribe-failed', async () => {
    installPushCapabilities({
      serviceWorker: {
        register: vi.fn(async () => ({
          pushManager: { subscribe: vi.fn(async () => { throw new Error('boom') }) },
        })),
        getRegistration: vi.fn(),
      },
    })
    stubFetchByUrl({ '/api/forum/push/config': jsonResponse({ code: 0, result: { configured: true, applicationServerKey: APP_KEY } }) })
    await expect(enableWebPush('zh')).rejects.toMatchObject({ code: 'subscribe-failed' })
  })

  test('throws network when persisting fails', async () => {
    installPushCapabilities({
      serviceWorker: {
        register: vi.fn(async () => ({
          pushManager: {
            subscribe: vi.fn(async () => ({ toJSON: () => ({ endpoint: ENDPOINT, keys: { p256dh: 'p', auth: 'a' } }) })),
          },
        })),
        getRegistration: vi.fn(),
      },
    })
    stubFetchByUrl({
      '/api/forum/push/config': jsonResponse({ code: 0, result: { configured: true, applicationServerKey: APP_KEY } }),
      '/api/forum/push/subscribe': jsonResponse({ code: 1 }),
    })
    await expect(enableWebPush('zh')).rejects.toMatchObject({ code: 'network' })
  })
})

describe('disableWebPush', () => {
  test('no-op when there is no subscription', async () => {
    installPushCapabilities({ serviceWorker: { register: vi.fn(), getRegistration: vi.fn(async () => null) } })
    const fetchMock = stubFetchByUrl({ '/api/forum/push/unsubscribe': jsonResponse({ code: 0 }) })
    await expect(disableWebPush()).resolves.toBeUndefined()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  test('unbinds backend endpoint then unsubscribes locally', async () => {
    const unsubscribe = vi.fn(async () => true)
    installPushCapabilities({
      serviceWorker: {
        register: vi.fn(),
        getRegistration: vi.fn(async () => ({
          pushManager: { getSubscription: vi.fn(async () => ({ toJSON: () => ({ endpoint: ENDPOINT }), unsubscribe })) },
        })),
      },
    })
    stubFetchByUrl({ '/api/forum/push/unsubscribe': jsonResponse({ code: 0 }) })

    await disableWebPush()

    const call = vi.mocked(fetch).mock.calls.find(([url]) => String(url) === '/api/forum/push/unsubscribe')
    expect(call).toBeDefined()
    const init = call![1] as RequestInit
    expect(JSON.parse(String(init.body))).toEqual({ endpoint: ENDPOINT })
    expect(unsubscribe).toHaveBeenCalledOnce()
  })

  test('throws unsubscribe-failed when browser unsubscribe rejects', async () => {
    installPushCapabilities({
      serviceWorker: {
        register: vi.fn(),
        getRegistration: vi.fn(async () => ({
          pushManager: {
            getSubscription: vi.fn(async () => ({
              toJSON: () => ({ endpoint: ENDPOINT }),
              unsubscribe: vi.fn(async () => { throw new Error('boom') }),
            })),
          },
        })),
      },
    })
    stubFetchByUrl({ '/api/forum/push/unsubscribe': jsonResponse({ code: 0 }) })
    await expect(disableWebPush()).rejects.toMatchObject({ code: 'unsubscribe-failed' })
  })
})

describe('static sw.js (pure push worker)', () => {
  interface WorkerHarness {
    handlers: Record<string, (event: any) => void>
    clients: {
      matchAll: ReturnType<typeof vi.fn>
      openWindow: ReturnType<typeof vi.fn>
    }
    showNotification: ReturnType<typeof vi.fn>
  }

  function loadWorker(): WorkerHarness {
    const handlers: Record<string, (event: any) => void> = {}
    const showNotification = vi.fn(async () => undefined)
    const openWindow = vi.fn(async () => undefined)
    const clients = {
      matchAll: vi.fn(async () => []),
      openWindow,
    }
    const sandbox: Record<string, unknown> = {
      console,
      addEventListener(type: string, handler: (event: any) => void) {
        handlers[type] = handler
      },
      clients,
      registration: { showNotification },
    }
    sandbox.self = sandbox
    vm.runInNewContext(SW_SOURCE, sandbox)
    return { handlers, clients, showNotification }
  }

  function pushEvent(harness: WorkerHarness, data: unknown): Promise<void> {
    const waiters: Promise<void>[] = []
    const event = {
      data: data === null ? null : { json: () => data },
      waitUntil(promise: Promise<void>) { waiters.push(promise) },
    }
    harness.handlers.push(event)
    return Promise.all(waiters).then(() => undefined)
  }

  function notificationClick(harness: WorkerHarness, data: unknown): Promise<void> {
    const waiters: Promise<void>[] = []
    harness.handlers.notificationclick({
      notification: { close: vi.fn(), data },
      waitUntil(promise: Promise<void>) { waiters.push(promise) },
    })
    return Promise.all(waiters).then(() => undefined)
  }

  test('registers only push and notificationclick (no fetch interception)', () => {
    const { handlers } = loadWorker()
    expect(Object.keys(handlers).sort()).toEqual(['notificationclick', 'push'])
  })

  test('shows a notification when the payload is valid and no window is focused', async () => {
    const harness = loadWorker()
    harness.clients.matchAll.mockResolvedValueOnce([{ focused: false }])
    await pushEvent(harness, {
      title: '新回复',
      body: '有人回复了你的主题',
      url: '/p/post/42#post-7',
      icon: '/static/pic/icon_300.webp',
    })
    expect(harness.showNotification).toHaveBeenCalledWith('新回复', {
      body: '有人回复了你的主题',
      icon: '/static/pic/icon_300.webp',
      data: { url: '/p/post/42#post-7' },
      tag: 'goose-push',
    })
  })

  test('does nothing for null / malformed / untitled payloads', async () => {
    const noPayload = loadWorker()
    await pushEvent(noPayload, null)
    expect(noPayload.showNotification).not.toHaveBeenCalled()

    const malformed = loadWorker()
    const waiters: Promise<void>[] = []
    malformed.handlers.push({
      data: { json: () => { throw new Error('not json') } },
      waitUntil(promise: Promise<void>) { waiters.push(promise) },
    })
    await Promise.all(waiters)
    expect(malformed.showNotification).not.toHaveBeenCalled()

    const untitled = loadWorker()
    await pushEvent(untitled, { body: 'no title' })
    expect(untitled.showNotification).not.toHaveBeenCalled()
  })

  test('skips the notification while the reader is focused on the site', async () => {
    const harness = loadWorker()
    harness.clients.matchAll.mockResolvedValueOnce([{ focused: true }])
    await pushEvent(harness, { title: '新回复', body: '…', url: '/p/post/42', icon: '/icon.png' })
    expect(harness.showNotification).not.toHaveBeenCalled()
  })

  test('notificationclick closes and focuses an open window, else opens the payload URL', async () => {
    const focus = vi.fn(async () => undefined)
    const harness = loadWorker()
    harness.clients.matchAll.mockResolvedValueOnce([{ focus }])
    await notificationClick(harness, { url: '/p/post/9' })
    expect(focus).toHaveBeenCalled()
    expect(harness.clients.openWindow).not.toHaveBeenCalled()

    const fallback = loadWorker()
    await notificationClick(fallback, { url: '/p/post/9' })
    expect(fallback.clients.openWindow).toHaveBeenCalledWith('/p/post/9')

    const noUrl = loadWorker()
    await notificationClick(noUrl, null)
    expect(noUrl.clients.openWindow).toHaveBeenCalledWith('/notifications')
  })
})
