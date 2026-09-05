import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import type * as BrowserNotificationModule from '../src/runtime/browser-notification'

// 模块持有 lastShownAt/channel 内部状态，逐测试 vi.resetModules 保证隔离。
let mod: typeof BrowserNotificationModule
let storage: { getItem: ReturnType<typeof vi.fn>; setItem: ReturnType<typeof vi.fn>; removeItem: ReturnType<typeof vi.fn> }
let pageHidden: boolean

class FakeNotification {
  static permission: NotificationPermission = 'default'
  static requestPermission = vi.fn<() => Promise<NotificationPermission>>()
  static instances: FakeNotification[] = []
  title: string
  options: NotificationOptions
  onclick: (() => void) | null = null
  closed = false

  constructor(title: string, options?: NotificationOptions) {
    this.title = title
    this.options = options || {}
    FakeNotification.instances.push(this)
  }

  close() {
    this.closed = true
  }
}

class FakeBroadcastChannel {
  static instances: FakeBroadcastChannel[] = []
  static posted: Array<{ at: number }> = []
  name: string
  onmessage: ((event: MessageEvent) => void) | null = null

  constructor(name: string) {
    this.name = name
    FakeBroadcastChannel.instances.push(this)
  }

  postMessage(data: unknown) {
    FakeBroadcastChannel.posted.push(data as { at: number })
  }

  close() {}
}

function createStorage() {
  const store = new Map<string, string>()
  return {
    getItem: vi.fn((key: string) => store.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => { store.set(key, value) }),
    removeItem: vi.fn((key: string) => { store.delete(key) }),
  }
}

async function loadModule() {
  vi.resetModules()
  mod = await import('../src/runtime/browser-notification')
}

function stubGlobals(options: { withBroadcastChannel?: boolean } = {}) {
  vi.stubGlobal('navigator', { languages: ['zh-CN'] })
  vi.stubGlobal('window', {
    localStorage: storage,
    Notification: FakeNotification,
    focus: vi.fn(),
    location: { href: 'http://localhost:3010/' },
  })
  if (options.withBroadcastChannel) {
    vi.stubGlobal('BroadcastChannel', FakeBroadcastChannel)
  }
}

// vue 的 runtime-dom 在导入时需要真实 document，因此 document stub 必须在模块加载完成后注入。
function stubDocument() {
  vi.stubGlobal('document', {
    cookie: '',
    get hidden() {
      return pageHidden
    },
  })
}

beforeEach(async () => {
  vi.useFakeTimers()
  pageHidden = true
  storage = createStorage()
  FakeNotification.instances = []
  FakeNotification.permission = 'default'
  FakeNotification.requestPermission = vi.fn<() => Promise<NotificationPermission>>(async () => 'granted')
  FakeBroadcastChannel.instances = []
  FakeBroadcastChannel.posted = []
  stubGlobals()
  await loadModule()
  stubDocument()
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

function enableForTest() {
  storage.setItem('goose:browser-notifications', 'on')
  FakeNotification.permission = 'granted'
}

describe('偏好读写', () => {
  test('isBrowserNotificationSupported 反映浏览器能力', () => {
    expect(mod.isBrowserNotificationSupported()).toBe(true)
    vi.stubGlobal('window', { localStorage: storage })
    expect(mod.isBrowserNotificationSupported()).toBe(false)
  })

  test('isBrowserNotificationEnabled 读取 localStorage 开关', () => {
    expect(mod.isBrowserNotificationEnabled()).toBe(false)
    storage.setItem('goose:browser-notifications', 'on')
    expect(mod.isBrowserNotificationEnabled()).toBe(true)
    storage.removeItem('goose:browser-notifications')
    expect(mod.isBrowserNotificationEnabled()).toBe(false)
  })

  test('enableBrowserNotifications 授权后写入偏好并返回 true', async () => {
    expect(await mod.enableBrowserNotifications()).toBe(true)
    expect(FakeNotification.requestPermission).toHaveBeenCalledOnce()
    expect(storage.setItem).toHaveBeenCalledWith('goose:browser-notifications', 'on')
    expect(mod.isBrowserNotificationEnabled()).toBe(true)
  })

  test('enableBrowserNotifications 拒绝授权时不落偏好并返回 false', async () => {
    FakeNotification.requestPermission = vi.fn<() => Promise<NotificationPermission>>(async () => 'denied')
    expect(await mod.enableBrowserNotifications()).toBe(false)
    expect(storage.setItem).not.toHaveBeenCalledWith('goose:browser-notifications', 'on')
    expect(mod.isBrowserNotificationEnabled()).toBe(false)
  })

  test('enableBrowserNotifications 已授权时不再弹权限请求', async () => {
    FakeNotification.permission = 'granted'
    expect(await mod.enableBrowserNotifications()).toBe(true)
    expect(FakeNotification.requestPermission).not.toHaveBeenCalled()
  })

  test('disableBrowserNotifications 移除偏好', () => {
    storage.setItem('goose:browser-notifications', 'on')
    mod.disableBrowserNotifications()
    expect(storage.removeItem).toHaveBeenCalledWith('goose:browser-notifications')
    expect(mod.isBrowserNotificationEnabled()).toBe(false)
  })
})

describe('showBrowserNotification', () => {
  test('不支持浏览器不展示', () => {
    enableForTest()
    vi.stubGlobal('window', { localStorage: storage })
    expect(mod.showBrowserNotification('comment')).toBe(false)
    expect(FakeNotification.instances).toHaveLength(0)
  })

  test('偏好关闭不展示', () => {
    FakeNotification.permission = 'granted'
    expect(mod.showBrowserNotification('comment')).toBe(false)
    expect(FakeNotification.instances).toHaveLength(0)
  })

  test('权限未授予不展示', () => {
    storage.setItem('goose:browser-notifications', 'on')
    expect(mod.showBrowserNotification('comment')).toBe(false)
    expect(FakeNotification.instances).toHaveLength(0)
  })

  test('页面前台不展示', () => {
    enableForTest()
    pageHidden = false
    expect(mod.showBrowserNotification('comment')).toBe(false)
    expect(FakeNotification.instances).toHaveLength(0)
  })

  test('条件满足时展示通知并按类型映射文案', () => {
    enableForTest()
    expect(mod.showBrowserNotification('follow')).toBe(true)
    expect(FakeNotification.instances).toHaveLength(1)
    const notification = FakeNotification.instances[0]
    expect(notification.title).toBe('通知')
    expect(notification.options.body).toBe('关注了你')
    expect(notification.options.icon).toBe('/static/pic/icon_300.webp')
    expect(notification.options.tag).toBe('goose:unread')
  })

  test('不同通知类型映射不同文案', () => {
    enableForTest()
    mod.showBrowserNotification('comment')
    vi.advanceTimersByTime(60_000)
    mod.showBrowserNotification('badge')
    expect(FakeNotification.instances).toHaveLength(2)
    expect(FakeNotification.instances[0].options.body).toBe('有新的评论回复')
    expect(FakeNotification.instances[1].options.body).toBe('你获得了新的徽章')
  })

  test('未知类型回落通用文案', () => {
    enableForTest()
    expect(mod.showBrowserNotification('system')).toBe(true)
    expect(FakeNotification.instances[0].options.body).toBe('有新的通知')
  })

  test('点击通知聚焦页面并跳转通知中心', () => {
    enableForTest()
    expect(mod.showBrowserNotification('comment')).toBe(true)
    const notification = FakeNotification.instances[0]
    notification.onclick?.()
    expect(notification.closed).toBe(true)
    expect((window as unknown as { focus: ReturnType<typeof vi.fn> }).focus).toHaveBeenCalled()
    expect((window as unknown as { location: { href: string } }).location.href).toBe('/notifications')
  })

  test('去重窗口内重复触发只弹一次', () => {
    enableForTest()
    expect(mod.showBrowserNotification('comment')).toBe(true)
    expect(mod.showBrowserNotification('comment')).toBe(false)
    expect(FakeNotification.instances).toHaveLength(1)
    vi.advanceTimersByTime(60_000)
    expect(mod.showBrowserNotification('comment')).toBe(true)
    expect(FakeNotification.instances).toHaveLength(2)
  })

  test('其他标签页广播去重：收到消息后本标签页不再弹', async () => {
    enableForTest()
    // happy-dom 自带真实 BroadcastChannel：模块加载即建频道的实现会先抓到
    // 真实实例，之后 stub 的 Fake 收不到 post。因此先 stub、再重载模块，
    // 让模块级 ensureDedupChannel 持有 Fake。
    vi.stubGlobal('BroadcastChannel', FakeBroadcastChannel)
    await loadModule()
    expect(mod.showBrowserNotification('comment')).toBe(true)
    expect(FakeBroadcastChannel.posted).toHaveLength(1)
    // 模拟另一标签页/SW 刚弹过通知
    FakeBroadcastChannel.instances[0].onmessage?.({ data: { at: Date.now() + 1 } } as MessageEvent)
    expect(mod.showBrowserNotification('comment')).toBe(false)
    expect(FakeNotification.instances).toHaveLength(1)
  })
})

describe('maybeNotifyUnread 翻转检测', () => {
  test('非「无未读→有未读」翻转不弹', () => {
    enableForTest()
    mod.maybeNotifyUnread(true, true, 'comment')
    mod.maybeNotifyUnread(true, false, 'comment')
    mod.maybeNotifyUnread(false, false, 'comment')
    expect(FakeNotification.instances).toHaveLength(0)
  })

  test('无未读→有未读翻转时弹通知', () => {
    enableForTest()
    mod.maybeNotifyUnread(false, true, 'comment')
    expect(FakeNotification.instances).toHaveLength(1)
    expect(FakeNotification.instances[0].options.body).toBe('有新的评论回复')
  })
})