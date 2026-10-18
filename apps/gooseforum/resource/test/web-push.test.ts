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
  prepareWebPush,
  rebindPushSubscription,
  resetPushPreparation,
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
  resetPushPreparation() // 模块级预取缓存不得跨用例复用
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
  test('reuses the cached preparation across calls', async () => {
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

    await enableWebPush('zh')
    await enableWebPush('ja')

    expect(serviceWorker.register).toHaveBeenCalledTimes(1)
    const configCalls = vi.mocked(fetch).mock.calls.filter(([url]) => String(url) === '/api/forum/push/config')
    expect(configCalls).toHaveLength(1)
    expect(pushManager.subscribe).toHaveBeenCalledTimes(2)
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
describe('prepareWebPush', () => {
  test('returns null when unsupported', () => {
    vi.stubGlobal('isSecureContext', false)
    expect(prepareWebPush()).toBeNull()
  })

  test('caches the config fetch and SW registration across calls', async () => {
    const serviceWorker = installPushCapabilities({
      serviceWorker: { register: vi.fn(async () => ({ pushManager: {} })), getRegistration: vi.fn() },
    })
    stubFetchByUrl({ '/api/forum/push/config': jsonResponse({ code: 0, result: { configured: true, applicationServerKey: APP_KEY } }) })

    await expect(prepareWebPush()).resolves.toBeDefined()
    await expect(prepareWebPush()).resolves.toBeDefined()

    expect(serviceWorker.register).toHaveBeenCalledTimes(1)
    const configCalls = vi.mocked(fetch).mock.calls.filter(([url]) => String(url) === '/api/forum/push/config')
    expect(configCalls).toHaveLength(1)
  })

  test('resets the cache on failure so a later call retries', async () => {
    const serviceWorker = installPushCapabilities({
      serviceWorker: { register: vi.fn(async () => ({ pushManager: {} })), getRegistration: vi.fn() },
    })
    stubFetchByUrl({ '/api/forum/push/config': jsonResponse({ code: 1 }, false) })
    await expect(prepareWebPush()).rejects.toMatchObject({ code: 'network' })

    stubFetchByUrl({ '/api/forum/push/config': jsonResponse({ code: 0, result: { configured: true, applicationServerKey: APP_KEY } }) })
    await expect(prepareWebPush()).resolves.toBeDefined()
    expect(serviceWorker.register).toHaveBeenCalledTimes(1) // 失败后重试会重新走完整准备
  })
})

describe('rebindPushSubscription', () => {
  test('returns false when unsupported', async () => {
    vi.stubGlobal('isSecureContext', false)
    await expect(rebindPushSubscription('zh')).resolves.toBe(false)
  })

  test('returns false when there is no browser subscription', async () => {
    installPushCapabilities({ serviceWorker: { register: vi.fn(), getRegistration: vi.fn(async () => null) } })
    const fetchMock = stubFetchByUrl({ '/api/forum/push/subscribe': jsonResponse({ code: 0 }) })
    await expect(rebindPushSubscription('zh')).resolves.toBe(false)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  test('persists the existing browser subscription to the current account', async () => {
    installPushCapabilities({
      serviceWorker: {
        register: vi.fn(),
        getRegistration: vi.fn(async () => ({
          pushManager: {
            getSubscription: vi.fn(async () => ({
              toJSON: () => ({ endpoint: ENDPOINT, keys: { p256dh: 'p256dh-value', auth: 'auth-value' } }),
            })),
          },
        })),
      },
    })
    stubFetchByUrl({ '/api/forum/push/subscribe': jsonResponse({ code: 0 }) })

    await expect(rebindPushSubscription('fr')).resolves.toBe(true)

    const call = vi.mocked(fetch).mock.calls.find(([url]) => String(url) === '/api/forum/push/subscribe')
    expect(call).toBeDefined()
    const init = call![1] as RequestInit
    expect(JSON.parse(String(init.body))).toEqual({
      subscription: { endpoint: ENDPOINT, keys: { p256dh: 'p256dh-value', auth: 'auth-value' } },
      lang: 'fr',
    })
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
  // BroadcastChannel 假件：sw.js 弹出通知后向共享去重频道广播 {at} 时间戳，
  // 轮询型浏览器通知（browser-notification.ts）收到后在去重窗口内让位。
  // 仅在 loadWorker({ withBroadcastChannel: true }) 时注入沙箱。
  class FakeBroadcastChannel {
    static instances: FakeBroadcastChannel[] = []
    static posted: Array<{ at: number }> = []
    name: string
    onmessage: ((event: MessageEvent<{ at?: number }>) => void) | null = null

    constructor(name: string) {
      this.name = name
      FakeBroadcastChannel.instances.push(this)
    }

    postMessage(data: unknown) {
      FakeBroadcastChannel.posted.push(data as { at: number })
    }

    close() {}
  }

  function resetBroadcastChannelFake() {
    FakeBroadcastChannel.instances = []
    FakeBroadcastChannel.posted = []
  }
  interface WorkerHarness {
    handlers: Record<string, (event: any) => void>
    clients: {
      matchAll: ReturnType<typeof vi.fn>
      openWindow: ReturnType<typeof vi.fn>
    }
    showNotification: ReturnType<typeof vi.fn>
    fetch: ReturnType<typeof vi.fn>
  }

  function loadWorker(options: {
    withBroadcastChannel?: boolean
    fetch?: unknown
    abortSignal?: unknown
    abortController?: unknown
    timers?: { setTimeout: (handler: () => void, timeout: number) => unknown; clearTimeout: (id: ReturnType<typeof setTimeout>) => void }
  } = {}): WorkerHarness {
    const handlers: Record<string, (event: any) => void> = {}
    const showNotification = vi.fn(async () => undefined)
    const openWindow = vi.fn(async () => undefined)
    const clients = {
      matchAll: vi.fn(async () => []),
      openWindow,
    }
    const sources = new Map<string, unknown>()
    const sourceCache = {match: async (key:string) => sources.get(key), put: async (key:string, value:unknown)=>{sources.set(key,value)}, keys: async ()=>[...sources.keys()], delete: async (key:string)=>sources.delete(key)}
    const sandbox: Record<string, unknown> = {
      console,
      location: {origin:'https://yourtj.example'},
      Response,
      AbortSignal,
      fetch: vi.fn(async () => ({ok:true,json:async()=>({code:0,result:{viewerId:7}})})),
      addEventListener(type: string, handler: (event: any) => void) {
        handlers[type] = handler
      },
      clients,
      registration: { showNotification },
      caches: {
        open: async () => {
          return sourceCache
        },
      },
      URL, // vm 沙箱不提供全局 URL，注入 Node 实现供 sw.js 使用
    }
    if (options.fetch) {
      sandbox.fetch = options.fetch
    }
    if ('abortSignal' in options) {
      sandbox.AbortSignal = options.abortSignal
    }
    if ('abortController' in options) {
      sandbox.AbortController = options.abortController
    }
    if (options.timers) {
      sandbox.setTimeout = options.timers.setTimeout
      sandbox.clearTimeout = options.timers.clearTimeout
    }
    if (options.withBroadcastChannel) {
      resetBroadcastChannelFake()
      sandbox.BroadcastChannel = FakeBroadcastChannel
    }
    sandbox.self = sandbox
    vm.runInNewContext(SW_SOURCE, sandbox)
    return { handlers, clients, showNotification, fetch: sandbox.fetch as ReturnType<typeof vi.fn> }
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
    expect(Object.keys(handlers).sort()).toEqual(['message', 'notificationclick', 'push'])
  })

  test('shows a notification without a tag when payload has no id', async () => {
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
      data: { url: '/p/post/42#post-7', recipientId:undefined, sourceKind:undefined },
    })
  })

  test('uses the payload id as a unique notification tag', async () => {
    const harness = loadWorker()
    harness.clients.matchAll.mockResolvedValueOnce([{ focused: false }])
    await pushEvent(harness, {
      id: 1024,
      title: '新回复',
      body: '有人回复了你的主题',
      url: '/p/post/42',
      icon: '/static/pic/icon_300.webp',
    })
    expect(harness.showNotification).toHaveBeenCalledWith('新回复', {
      body: '有人回复了你的主题',
      icon: '/static/pic/icon_300.webp',
      data: { url: '/p/post/42', recipientId:undefined, sourceKind:undefined },
      tag: 'goose-push-notification:1024',
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

  test('focuses a window already at the target path without navigating', async () => {
    const focus = vi.fn(async () => undefined)
    const navigate = vi.fn(async () => undefined)
    const harness = loadWorker()
    // 完整目标比较：origin + pathname + search + hash（本用例两侧都无 query/hash）
    harness.clients.matchAll.mockResolvedValueOnce([{ url: 'https://yourtj.example/p/post/9', focus, navigate }])
    await notificationClick(harness, { url: '/p/post/9' })
    expect(focus).toHaveBeenCalledOnce()
    expect(navigate).not.toHaveBeenCalled()
    expect(harness.clients.openWindow).not.toHaveBeenCalled()
  })

  test('navigates a window on another path to the deep link before focusing', async () => {
    const focus = vi.fn(async () => undefined)
    const navigate = vi.fn(async () => undefined)
    const harness = loadWorker()
    harness.clients.matchAll.mockResolvedValueOnce([{ url: 'https://yourtj.example/home', focus, navigate }])
    await notificationClick(harness, { url: '/p/post/9' })
    expect(navigate).toHaveBeenCalledWith('/p/post/9')
    expect(focus).toHaveBeenCalledOnce()
    expect(focus.mock.invocationCallOrder[0]).toBeGreaterThan(navigate.mock.invocationCallOrder[0])
    expect(harness.clients.openWindow).not.toHaveBeenCalled()
  })

  test('falls back to openWindow when navigate throws', async () => {
    const focus = vi.fn(async () => undefined)
    const navigate = vi.fn(async () => { throw new Error('cross-origin') })
    const harness = loadWorker()
    harness.clients.matchAll.mockResolvedValueOnce([{ url: 'https://yourtj.example/home', focus, navigate }])
    await notificationClick(harness, { url: '/p/post/9' })
    expect(harness.clients.openWindow).toHaveBeenCalledWith('/p/post/9')
    expect(focus).not.toHaveBeenCalled() // 绝不聚焦未成功导航的窗口
  })

  test('falls back to openWindow when no window can navigate', async () => {
    const focus = vi.fn(async () => undefined)
    const harness = loadWorker()
    harness.clients.matchAll.mockResolvedValueOnce([
      { url: 'https://yourtj.example/about:blank', focus },
      { url: 'https://yourtj.example/home', focus },
    ])
    await notificationClick(harness, { url: '/p/post/9' })
    expect(harness.clients.openWindow).toHaveBeenCalledWith('/p/post/9')
    expect(focus).not.toHaveBeenCalled()
  })

  test('opens a new window when none is open, falling back to /notifications', async () => {
    const harness = loadWorker()
    await notificationClick(harness, { url: '/p/post/9' })
    expect(harness.clients.openWindow).toHaveBeenCalledWith('/p/post/9')

    const noUrl = loadWorker()
    await notificationClick(noUrl, null)
    expect(noUrl.clients.openWindow).toHaveBeenCalledWith('/notifications')
  })

  test('broadcasts a dedup timestamp after showing a push notification', async () => {
    const harness = loadWorker({ withBroadcastChannel: true })
    harness.clients.matchAll.mockResolvedValueOnce([{ focused: false }])
    await pushEvent(harness, { title: '新回复', body: '…', url: '/p/post/42', icon: '/icon.png' })
    expect(harness.showNotification).toHaveBeenCalledOnce()
    // 轮询通道（browser-notification.ts）在同一频道收到 {at} 后 60s 内让位。
    expect(FakeBroadcastChannel.posted).toHaveLength(1)
    expect(typeof FakeBroadcastChannel.posted[0].at).toBe('number')
    expect(FakeBroadcastChannel.instances).toHaveLength(1)
  })

  test('does not broadcast when the reader is focused (nothing shown)', async () => {
    const harness = loadWorker({ withBroadcastChannel: true })
    harness.clients.matchAll.mockResolvedValueOnce([{ focused: true }])
    await pushEvent(harness, { title: '新回复', body: '…', url: '/p/post/42', icon: '/icon.png' })
    expect(harness.showNotification).not.toHaveBeenCalled()
    expect(FakeBroadcastChannel.posted).toHaveLength(0)
  })

  test('still shows the notification when BroadcastChannel is unavailable', async () => {
    const harness = loadWorker() // 沙箱不注入 BroadcastChannel：广播优雅跳过
    harness.clients.matchAll.mockResolvedValueOnce([{ focused: false }])
    await pushEvent(harness, { title: '新回复', body: '…', url: '/p/post/42', icon: '/icon.png' })
    expect(harness.showNotification).toHaveBeenCalledOnce()
  })
  test('durable source IDs deduplicate repeats without suppressing distinct notifications', async () => {
    const harness = loadWorker()
    await pushEvent(harness,{id:1,title:'One',url:'/notifications'})
    await pushEvent(harness,{id:1,title:'One again',url:'/notifications'})
    await pushEvent(harness,{id:2,title:'Two',url:'/notifications'})
    expect(harness.showNotification).toHaveBeenCalledTimes(2)
  })
  test('chat push is suppressed only in the focused matching conversation', async () => {
    const harness = loadWorker()
    harness.handlers.message({source:{id:'tab'},data:{type:'goose:active-chat',peerId:9}})
    harness.clients.matchAll.mockResolvedValue([{id:'tab',focused:true,url:'https://yourtj.example/messages'}])
    await pushEvent(harness,{id:1,sourceKind:'chat_message',recipientId:7,title:'Message',url:'/messages?userId=9'})
    expect(harness.showNotification).not.toHaveBeenCalled()
    await pushEvent(harness,{id:2,sourceKind:'chat_message',recipientId:7,title:'Other message',url:'/messages?userId=10'})
    expect(harness.showNotification).toHaveBeenCalledOnce()
  })
  test('stale recipient chat clicks return to messages without the old peer', async () => {
    const harness = loadWorker()
    await notificationClick(harness,{sourceKind:'chat_message',recipientId:8,url:'/messages?userId=9'})
    expect(harness.clients.openWindow).toHaveBeenCalledWith('/messages')
  })
  test('chat clicks change conversation even when pathname matches', async () => {
    const harness = loadWorker()
    const focus=vi.fn(); const navigate=vi.fn()
    harness.clients.matchAll.mockResolvedValue([{url:'https://yourtj.example/messages?userId=10',focus,navigate}])
    await notificationClick(harness,{sourceKind:'chat_message',recipientId:7,url:'/messages?userId=9'})
    expect(navigate).toHaveBeenCalledWith('/messages?userId=9')
  })
  test('external click URLs cannot navigate away from this origin', async () => {
    const harness = loadWorker()
    await notificationClick(harness,{url:'https://evil.example/path'})
    expect(harness.clients.openWindow).toHaveBeenCalledWith('/notifications')
  })

  test('chat URL alone cannot suppress a push before the conversation has loaded', async () => {
    const harness = loadWorker()
    harness.clients.matchAll.mockResolvedValue([{id:'unloaded',focused:true,url:'https://yourtj.example/messages?userId=9'}])
    await pushEvent(harness,{id:17,sourceKind:'chat_message',recipientId:7,title:'Message',url:'/messages?userId=9'})
    expect(harness.showNotification).toHaveBeenCalledOnce()
  })
  test('a queued activity for a previous account is not shown or opened', async () => {
    const harness = loadWorker()
    await pushEvent(harness,{id:18,sourceKind:'notification',recipientId:8,title:'Old account activity',url:'/p/post/42'})
    expect(harness.showNotification).not.toHaveBeenCalled()
    await notificationClick(harness,{sourceKind:'notification',recipientId:8,url:'/p/post/42'})
    expect(harness.clients.openWindow).toHaveBeenCalledWith('/notifications')
  })

  // 收件人复核只在服务端「明确确认当前是另一个账号」时抑制推送；无法判定
  // （请求失败/超时 >5s/非 2xx/响应不可解析/会话过期）时页面关闭仍会收到通知，
  // 但不展示推送原文：标题换成服务端下发的通用文案、正文丢弃，否则身份不明的
  // 浏览器会看到推送内容（issue #1183 AC-6）。
  describe('an undecidable recipient check shows a body-less notification', () => {
    const pushedTitle = 'Topic title only the recipient should see'
    const pushedBody = 'Message body only the recipient should see'
    const pushedPayload = (id: number) => ({
      id,
      sourceKind: 'chat_message',
      recipientId: 7,
      title: pushedTitle,
      body: pushedBody,
      genericTitle: '你收到一条系统通知',
      url: '/messages?userId=9',
    })
    function expectBodylessNotification(harness: WorkerHarness) {
      expect(harness.showNotification).toHaveBeenCalledOnce()
      const [title, options] = harness.showNotification.mock.calls[0] as [string, { body?: string }]
      expect(title).toBe('你收到一条系统通知')
      expect(options.body).toBeUndefined()
      expect(JSON.stringify([title, options])).not.toContain(pushedTitle)
      expect(JSON.stringify([title, options])).not.toContain(pushedBody)
    }

    const failureCases: Array<[string, unknown]> = [
      ['the check request fails', vi.fn(async () => { throw new Error('offline') })],
      ['the response is not 2xx', vi.fn(async () => ({ok:false,status:500,json:async()=>({})}))],
      ['the response is not parseable', vi.fn(async () => ({ok:true,json:async()=>{throw new Error('not json')}}))],
      ['the session expired and the API answers an unauthenticated envelope', vi.fn(async () => ({ok:true,json:async()=>({code:1,message:'unauthorized',result:null})}))],
    ]
    for (const [name, fetchImpl] of failureCases) {
      test(`${name}: the notification is still shown without the push copy`, async () => {
        const harness = loadWorker({ fetch: fetchImpl })
        await pushEvent(harness, pushedPayload(19))
        expectBodylessNotification(harness)
      })
    }

    test('the check times out after 5s: the notification is still shown without the push copy', async () => {
      const timeouts: number[] = []
      const hangingFetch = vi.fn((_url: string, init: any) => new Promise((_resolve, reject) => {
        init.signal.addEventListener('abort', () => reject(new Error('timeout')))
      }))
      const harness = loadWorker({
        fetch: hangingFetch,
        abortSignal: { timeout: (ms: number) => { timeouts.push(ms); return AbortSignal.timeout(10) } },
      })
      await pushEvent(harness, pushedPayload(20))
      expect(timeouts).toEqual([5000])
      expect(hangingFetch).toHaveBeenCalledOnce()
      expectBodylessNotification(harness)
    })

    test('a browser without AbortSignal.timeout bounds the check with AbortController and drops the copy', async () => {
      const delays: number[] = []
      const hangingFetch = vi.fn((_url: string, init: any) => new Promise((_resolve, reject) => {
        init.signal.addEventListener('abort', () => reject(new Error('timeout')))
      }))
      const harness = loadWorker({
        fetch: hangingFetch,
        abortSignal: undefined,
        abortController: AbortController,
        timers: {
          setTimeout: (handler: () => void, ms: number) => { delays.push(ms); return setTimeout(handler, Math.min(ms, 10)) },
          clearTimeout,
        },
      })
      await pushEvent(harness, pushedPayload(21))
      expect(delays).toEqual([5000])
      expect(hangingFetch).toHaveBeenCalledOnce()
      expectBodylessNotification(harness)
    })

    test('a server payload without genericTitle is still shown body-less, without the push copy', async () => {
      const harness = loadWorker({ fetch: vi.fn(async () => { throw new Error('offline') }) })
      await pushEvent(harness, {
        id: 22, sourceKind: 'chat_message', recipientId: 7, title: pushedTitle, body: pushedBody, url: '/messages?userId=9',
      })
      expect(harness.showNotification).toHaveBeenCalledOnce()
      const [title, options] = harness.showNotification.mock.calls[0] as [string, { body?: string }]
      expect(title).toBe('')
      expect(options.body).toBeUndefined()
      expect(JSON.stringify([title, options])).not.toContain(pushedBody)
    })

    test('an undecided click lands on the generic destination, not the pushed URL', async () => {
      const harness = loadWorker({ fetch: vi.fn(async () => { throw new Error('offline') }) })
      await notificationClick(harness,{sourceKind:'chat_message',recipientId:7,url:'/messages?userId=9'})
      expect(harness.clients.openWindow).toHaveBeenCalledWith('/messages')
    })

    test('an undecided non-chat click falls back to /notifications', async () => {
      const harness = loadWorker({ fetch: vi.fn(async () => { throw new Error('offline') }) })
      await notificationClick(harness,{sourceKind:'notification',recipientId:7,url:'/p/post/42'})
      expect(harness.clients.openWindow).toHaveBeenCalledWith('/notifications')
    })

    test('the recipient check reads the lightweight config endpoint, not the messages page', async () => {
      const harness = loadWorker()
      await pushEvent(harness, pushedPayload(24))
      expect(harness.fetch).toHaveBeenCalled()
      expect(harness.fetch.mock.calls.map(call => call[0])).not.toContain('/messages')
      expect(harness.fetch.mock.calls[0][0]).toBe('/api/forum/push/config')
    })

    test('a confirmed different account still suppresses and redirects', async () => {
      const harness = loadWorker()
      await pushEvent(harness,{id:23,sourceKind:'notification',recipientId:8,title:'Old account activity',url:'/p/post/42'})
      expect(harness.showNotification).not.toHaveBeenCalled()
      await notificationClick(harness,{sourceKind:'notification',recipientId:8,url:'/p/post/42'})
      expect(harness.clients.openWindow).toHaveBeenCalledWith('/notifications')
    })
  })

})
