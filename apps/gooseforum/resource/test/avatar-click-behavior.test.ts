// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'

// happy-dom 20 把 localStorage 定义为原型 getter，测试环境缺失该全局（与
// quick-publish-draft-guard.test.ts 同一处理）：提供最小内存实现，仅为断言持久化与失败回退。
const memoryStorage = new Map<string, string>()
Object.defineProperty(window, 'localStorage', {
  configurable: true,
  value: {
    getItem: (key: string) => memoryStorage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      memoryStorage.set(key, String(value))
    },
    removeItem: (key: string) => {
      memoryStorage.delete(key)
    },
    clear: () => {
      memoryStorage.clear()
    },
    key: (index: number) => Array.from(memoryStorage.keys())[index] ?? null,
    get length() {
      return memoryStorage.size
    },
  },
})

const storageKey = 'goose:avatar-click-behavior'

interface TargetUser {
  id: number
  username: string
  avatarUrl: string
  publicUid?: string
}

const alice: TargetUser = { id: 42, username: 'alice', avatarUrl: '' }
const anonymousPersona: TargetUser = { ...alice, publicUid: 'persona-uid' }

async function loadModules() {
  const behavior = await import('../src/runtime/avatar-click-behavior')
  const events = await import('../src/runtime/user-card-events')
  return { showUserCard: events.showUserCard, prefs: behavior.useAvatarClickBehavior() }
}

interface ClickInit {
  button?: number
  metaKey?: boolean
  ctrlKey?: boolean
  shiftKey?: boolean
  altKey?: boolean
}

/**
 * 在带 href 的锚点上派发一次点击（与全站 13 处调用点同构），
 * 同时监听卡片事件与 document 冒泡（全站点击拦截器的位置）。
 */
function clickTargetAnchor(
  showUserCard: (user: TargetUser, event: MouseEvent) => void,
  user: TargetUser,
  init: ClickInit = {},
) {
  const anchor = document.createElement('a')
  anchor.href = user.publicUid ? `/a/${user.publicUid}` : `/u/${user.id}`
  document.body.appendChild(anchor)

  const cardListener = vi.fn()
  const documentListener = vi.fn()
  window.addEventListener('goose:user-card-show', cardListener)
  document.addEventListener('click', documentListener)
  anchor.addEventListener('click', (event) => showUserCard(user, event))

  const event = new MouseEvent('click', {
    bubbles: true,
    cancelable: true,
    button: init.button ?? 0,
    metaKey: init.metaKey ?? false,
    ctrlKey: init.ctrlKey ?? false,
    shiftKey: init.shiftKey ?? false,
    altKey: init.altKey ?? false,
  })
  anchor.dispatchEvent(event)
  window.removeEventListener('goose:user-card-show', cardListener)
  document.removeEventListener('click', documentListener)

  return { event, cardListener, documentListener }
}

const modifierCases: Array<[string, ClickInit]> = [
  ['中键', { button: 1 }],
  ['⌘+点击', { metaKey: true }],
  ['Ctrl+点击', { ctrlKey: true }],
  ['Shift+点击', { shiftKey: true }],
  ['Alt+点击', { altKey: true }],
]

describe('头像点击行为偏好（issue #1134）', () => {
  beforeEach(() => {
    window.localStorage.clear()
    document.body.innerHTML = ''
    vi.resetModules()
  })

  test('默认偏好（无本地值）：点击头像弹用户卡片并阻止默认导航', async () => {
    const { showUserCard } = await loadModules()

    const { event, cardListener, documentListener } = clickTargetAnchor(showUserCard, alice)

    expect(cardListener).toHaveBeenCalledTimes(1)
    expect(event.defaultPrevented).toBe(true)
    expect(documentListener).not.toHaveBeenCalled()
  })

  test('直达偏好：不弹卡片也不拦截，锚点 href 交还浏览器与全站点击拦截器', async () => {
    const { showUserCard, prefs } = await loadModules()
    prefs.setAvatarClickBehavior('profile')

    const { event, cardListener, documentListener } = clickTargetAnchor(showUserCard, alice)

    expect(cardListener).not.toHaveBeenCalled()
    expect(event.defaultPrevented).toBe(false)
    expect(documentListener).toHaveBeenCalledTimes(1)
  })

  test('偏好切换即时生效：同一会话内先弹卡片、切换后直达，无需刷新', async () => {
    const { showUserCard, prefs } = await loadModules()

    const before = clickTargetAnchor(showUserCard, alice)
    expect(before.cardListener).toHaveBeenCalledTimes(1)

    prefs.setAvatarClickBehavior('profile')
    expect(prefs.avatarClickBehavior.value).toBe('profile')

    const after = clickTargetAnchor(showUserCard, alice)
    expect(after.cardListener).not.toHaveBeenCalled()
    expect(after.event.defaultPrevented).toBe(false)
  })

  test.each(modifierCases)('修饰键语义在两种偏好下都保留：%s 交还浏览器默认新标签行为', async (_label, init) => {
    for (const behavior of ['card', 'profile'] as const) {
      vi.resetModules()
      const { showUserCard, prefs } = await loadModules()
      prefs.setAvatarClickBehavior(behavior)

      const { event, cardListener } = clickTargetAnchor(showUserCard, alice, init)

      expect(cardListener).not.toHaveBeenCalled()
      expect(event.defaultPrevented).toBe(false)
    }
  })

  test('匿名公开身份（publicUid）在两种偏好下都跟随 /a/{uid} 链接、不弹卡片', async () => {
    for (const behavior of ['card', 'profile'] as const) {
      vi.resetModules()
      const { showUserCard, prefs } = await loadModules()
      prefs.setAvatarClickBehavior(behavior)

      const { event, cardListener } = clickTargetAnchor(showUserCard, anonymousPersona)

      expect(cardListener).not.toHaveBeenCalled()
      expect(event.defaultPrevented).toBe(false)
    }
  })

  test('偏好写入 localStorage，重开浏览器（重新加载模块）后仍为直达', async () => {
    const { prefs } = await loadModules()
    prefs.setAvatarClickBehavior('profile')
    expect(window.localStorage.getItem(storageKey)).toBe('profile')

    vi.resetModules()
    const reloaded = await loadModules()
    expect(reloaded.prefs.avatarClickBehavior.value).toBe('profile')
  })

  test('无值或非法持久化值回退默认卡片模式', async () => {
    window.localStorage.setItem(storageKey, 'bogus')
    const invalid = await loadModules()
    expect(invalid.prefs.avatarClickBehavior.value).toBe('card')

    vi.resetModules()
    window.localStorage.clear()
    const missing = await loadModules()
    expect(missing.prefs.avatarClickBehavior.value).toBe('card')
  })

  test('localStorage 不可用（受限浏览）时读取回退默认、写入不抛错且即时生效', async () => {
    const getItem = vi.spyOn(window.localStorage, 'getItem').mockImplementation(() => {
      throw new Error('storage disabled')
    })
    const { showUserCard, prefs } = await loadModules()
    expect(prefs.avatarClickBehavior.value).toBe('card')
    getItem.mockRestore()

    const setItem = vi.spyOn(window.localStorage, 'setItem').mockImplementation(() => {
      throw new Error('storage disabled')
    })
    expect(() => prefs.setAvatarClickBehavior('profile')).not.toThrow()
    setItem.mockRestore()

    // 写入失败不影响本次会话的分流
    const { cardListener } = clickTargetAnchor(showUserCard, alice)
    expect(cardListener).not.toHaveBeenCalled()
  })

  test('恢复默认：清除持久化并回到卡片模式', async () => {
    const { showUserCard, prefs } = await loadModules()
    prefs.setAvatarClickBehavior('profile')

    prefs.restoreAvatarClickBehavior()

    expect(prefs.avatarClickBehavior.value).toBe('card')
    expect(window.localStorage.getItem(storageKey)).toBeNull()
    const { cardListener } = clickTargetAnchor(showUserCard, alice)
    expect(cardListener).toHaveBeenCalledTimes(1)
  })

  test('跨标签页同步：其他标签页改写偏好后本标签页经 storage 事件即时跟随', async () => {
    const { showUserCard, prefs } = await loadModules()
    expect(prefs.avatarClickBehavior.value).toBe('card')

    // storage 事件只在其他文档触发，这里直接派发以模拟另一标签页写入。
    window.localStorage.setItem(storageKey, 'profile')
    window.dispatchEvent(new StorageEvent('storage', { key: storageKey, newValue: 'profile' }))

    expect(prefs.avatarClickBehavior.value).toBe('profile')
    const { cardListener, event } = clickTargetAnchor(showUserCard, alice)
    expect(cardListener).not.toHaveBeenCalled()
    expect(event.defaultPrevented).toBe(false)
  })

  test('跨标签页恢复默认：removeItem 与 clear（key 为 null）都回到卡片模式', async () => {
    const { prefs } = await loadModules()
    prefs.setAvatarClickBehavior('profile')

    window.localStorage.removeItem(storageKey)
    window.dispatchEvent(new StorageEvent('storage', { key: storageKey, newValue: null }))
    expect(prefs.avatarClickBehavior.value).toBe('card')

    prefs.setAvatarClickBehavior('profile')
    window.localStorage.clear()
    window.dispatchEvent(new StorageEvent('storage', { key: null }))
    expect(prefs.avatarClickBehavior.value).toBe('card')
  })

  test('无关 key 的 storage 事件不改动本标签页偏好', async () => {
    const { prefs } = await loadModules()
    expect(prefs.avatarClickBehavior.value).toBe('card')

    window.localStorage.setItem(storageKey, 'profile')
    window.dispatchEvent(new StorageEvent('storage', { key: 'goose:shell-sidebar-collapsed', newValue: '1' }))

    expect(prefs.avatarClickBehavior.value).toBe('card')
  })
})
