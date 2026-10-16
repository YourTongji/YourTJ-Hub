import { getPushPreferences } from './web-push'
import { notificationText } from './notification-text'
import { i18n } from './i18n'
import { navigateAppTo } from './app-navigation'
import { fetchNotifications } from './api'
import type { NotificationPayload } from '@gooseforum/client'

/**
 * 浏览器级通知（Web Notification API，issue #444）。
 *
 * 纯前端通道：复用 unread-status 的 30s 轮询结果，按最新未读通知 id（latestUnreadId）推进触发：
 * 页面处于后台时出现比已通知 id 更新的未读才弹系统通知，未读持续期间新通知不漏报；
 * 响应不含 id（旧版本/兜底）时保留「无未读 → 有未读」翻转语义。需要用户显式开启
 * （设置页 privacy 开关）并授予权限；权限被拒时优雅降级为铃铛红点 + 文档标题，不影响现有行为。
 */

const STORAGE_KEY = 'goose:browser-notifications'
const DEDUP_CHANNEL_NAME = 'goose:browser-notifications-dedup'
const DEDUP_STORAGE_KEY = 'goose:notification-shown-at'
// Bounded source ledger (localStorage + Cache API): the newest 256 source IDs
// cover the polling replay window. A longer window needs an expiry store.
const SOURCE_STORAGE_KEY = 'goose:notification-sources'
// 去重窗口取轮询间隔（30s）的两倍：各标签页轮询相位不同，最坏情况两个 tab 的翻转检测相差近一个周期。
const DEDUP_WINDOW_MS = 60_000
const NOTIFICATION_ICON = '/static/pic/icon_300.webp'
const NOTIFICATION_TAG = 'goose:unread'

const shownSources = new Set<string>()
let lastShownAt = 0
let channel: BroadcastChannel | null | undefined
// 已通知过的最新未读通知 id：仅本标签页内存，不落 localStorage
// （跨标签页/SW 由来源 ID 账本负责；无 ID 的旧响应保留时间窗）。
let lastNotifiedUnreadId = 0
let unreadGeneration = 0

// 模块加载即打开去重频道（不等首次弹出）：Web Push（sw.js）在弹出通知后会
// 向同一频道广播 {source, at}。若只在 markShown 时才懒建频道，首个轮询翻转发生在
// SW 广播之后（本页尚未监听）就会漏掉抑制信号，造成同一事件双弹。
ensureDedupChannel()

export function isBrowserNotificationSupported(): boolean {
  return typeof window !== 'undefined' && 'Notification' in window
}

export function isBrowserNotificationEnabled(): boolean {
  if (!isBrowserNotificationSupported()) return false
  try {
    return window.localStorage.getItem(STORAGE_KEY) === 'on'
  } catch {
    return false
  }
}

function writePreference(enabled: boolean) {
  try {
    if (enabled) window.localStorage.setItem(STORAGE_KEY, 'on')
    else window.localStorage.removeItem(STORAGE_KEY)
  } catch {
    // 偏好读写失败只影响系统通知，不应破坏设置页其他功能。
  }
}

export function requestBrowserNotificationPermission(): Promise<NotificationPermission> {
  if (!isBrowserNotificationSupported()) return Promise.resolve('denied')
  if (window.Notification.permission !== 'default') return Promise.resolve(window.Notification.permission)
  return window.Notification.requestPermission()
}

/** 开启浏览器通知：请求权限并在授予后写入偏好；未授予时不落偏好，返回 false。 */
export async function enableBrowserNotifications(): Promise<boolean> {
  const permission = await requestBrowserNotificationPermission()
  if (permission !== 'granted') return false
  writePreference(true)
  return true
}

export function disableBrowserNotifications() {
  writePreference(false)
}

function readShownAt(): number {
  try {
    return Number(window.localStorage.getItem(DEDUP_STORAGE_KEY)) || 0
  } catch {
    return 0
  }
}

function ensureDedupChannel() {
  if (channel !== undefined) return
  // BroadcastChannel 尚不可用（老浏览器/模块加载早于 channel 注入的测试环境）
  // 时保持 undefined 直接返回：后续环境可用时 markShown 会重试重建，避免
  // 永久锁死为 null 导致跨标签页/SW 的抑制信号全部丢失。
  if (typeof BroadcastChannel === 'undefined') return
  try {
    const bc = new BroadcastChannel(DEDUP_CHANNEL_NAME)
    bc.onmessage = (event: MessageEvent<{ at?: number; source?: string }>) => {
      if (event.data?.source) {
        shownSources.add(event.data.source)
        if (shownSources.size > 256) shownSources.delete(shownSources.values().next().value!)
      }
      const at = event.data?.at
      if (typeof at === 'number' && at > lastShownAt) lastShownAt = at
    }
    channel = bc
  } catch {
    // 构造失败（如隐私模式限制）：本会话内不再重试。
    channel = null
  }
}

function markShown(source = '') {
  if (source) {
    shownSources.add(source)
    if (shownSources.size > 256) shownSources.delete(shownSources.values().next().value!)
    try {
      const sources = JSON.parse(window.localStorage.getItem(SOURCE_STORAGE_KEY) || '[]') as string[]
      window.localStorage.setItem(SOURCE_STORAGE_KEY, JSON.stringify([...sources.filter(key => key !== source), source].slice(-256)))
    } catch {}
  }
  lastShownAt = Date.now()
  try {
    window.localStorage.setItem(DEDUP_STORAGE_KEY, String(lastShownAt))
  } catch {
    // 存储不可用时至少保留本标签页内（内存）去重。
  }
  ensureDedupChannel()
  channel?.postMessage({ at: lastShownAt, source })
}

/**
 * 展示浏览器通知。全部条件满足才弹：
 * 浏览器支持、偏好开启、权限已授予、页面处于后台、去重窗口内未被其他标签页弹过。
 * mention 带有已查询的楼层目标时直达楼层；缺字段/失败回退通知中心。
 */
function canShowBrowserNotification(source = ''): boolean {
  if (!isBrowserNotificationSupported()) return false
  if (!isBrowserNotificationEnabled()) return false
  if (window.Notification.permission !== 'granted') return false
  if (!document.hidden) return false
  if (source) {
    if (shownSources.has(source)) return false
    try { if ((JSON.parse(window.localStorage.getItem(SOURCE_STORAGE_KEY) || '[]') as string[]).includes(source)) return false } catch {}
    return true
  }
  const shownAt = Math.max(lastShownAt, readShownAt())
  if (Date.now() - shownAt < DEDUP_WINDOW_MS) return false

  return true
}

function mentionTarget(item?: NotificationPayload): string {
  const positive = (value: unknown): value is number => typeof value === 'number' && Number.isSafeInteger(value) && value > 0
  const topicId = item?.payload?.topicId ?? item?.topic?.id
  if (!positive(topicId)) return '/notifications'
  const base = `/p/post/${topicId}`
  const postNo = item?.payload?.postNo
  if (positive(postNo)) return `${base}/${postNo}`
  const postId = item?.payload?.postId
  return positive(postId) ? `${base}#post-${postId}` : base
}

export function showBrowserNotification(type: string, item?: NotificationPayload, sourceId = item?.id ?? 0): boolean {
  const source = sourceId > 0 ? `notification:${sourceId}` : ''
  if (!canShowBrowserNotification(source)) return false
  const target = item?.topic?.url || (type === 'mention' ? mentionTarget(item) : '/notifications')
  const notification = new window.Notification(i18n.global.t('notifications.title'), {
    body: notificationText(item ?? { eventType: type, payload: { actorId: 0 } }, i18n.global.t),
    icon: NOTIFICATION_ICON,
    tag: source ? `goose-push-${source}` : NOTIFICATION_TAG,
  })
  notification.onclick = () => {
    notification.close()
    window.focus()
    if (!navigateAppTo(target)) {
      // 导航桥未注册（极端时序）：回退整页跳转，保证点击必然可达通知中心
      window.location.href = target
    }
  }
  markShown(source)
  return true
}

async function showSourceNotification(id: number, type: string) {
  const generation = unreadGeneration
  const source = `notification:${id}`
  if (!canShowBrowserNotification(source)) return
  let item: NotificationPayload | undefined
  try {
    // Cursor is exclusive: this bounded query selects this exact notification,
    // even if newer arrivals have changed the first page in the meantime.
    const result = await fetchNotifications('all', id + 1, 1)
    item = result.items.find(candidate => candidate.id === id)
  } catch {
    // Preserve the generic mention notice when the detail request is unavailable.
  }
  if (generation !== unreadGeneration || item?.isRead) return
  try {
    const preferences = await getPushPreferences()
    const semantic = item?.payload.hasDirectMention ? 'mention' : item?.eventType || type
    const group = ['mention','post_reply','comment'].includes(semantic) ? 'mentionsAndReplies'
      : semantic === 'like' ? 'activity'
        : ['follow','topic_post','wiki_updated'].includes(semantic) ? 'following' : 'system'
    if (!preferences[group]) return
    if (group === 'system' && preferences.systemDefaultOnly && !['review_approved','review_rejected'].includes(semantic)) return
  } catch { return } // No account policy proof: keep the unread indicator without external presentation.
  const show = async () => {
    if (generation !== unreadGeneration) return
    let cache: Cache | undefined
    const key = new URL(`/__notification-source/${source}`, window.location.href).href
    try {
      if (typeof caches !== 'undefined') {
        cache = await caches.open('goose-notification-sources-v1')
        if (await cache.match(key)) return
      }
    } catch { cache = undefined }
    if (!showBrowserNotification(type, item, id) || !cache) return
    try {
      await cache.put(key, new Response('1'))
      const keys = await cache.keys()
      for (const old of keys.slice(0, Math.max(0, keys.length - 256))) await cache.delete(old)
    } catch {} // Storage restrictions leave the source broadcast and tab ledger as fallback.
  }
  if (typeof navigator !== 'undefined' && navigator.locks) {
    try { await navigator.locks.request(`goose-notification:${source}`, show) }
    catch { await show() }
  } else await show()
}

/**
 * 轮询触发入口：按最新未读通知 id 推进，未读持续期间新通知（更大 id）不漏报。
 * 语义：页面可见时用户在看红点/页面本身，只推进游标不弹（避免「已见还补弹」）；
 * 切后台后新 id 才会触发；每轮只弹最新 id，按来源 ID 去重。
 */
export function maybeNotifyUnread(previous: boolean, current: boolean, type: string, latestUnreadId = 0) {
  if (!current) {
    // 已读清零：重置游标
    lastNotifiedUnreadId = 0
    unreadGeneration++
    return
  }
  const id = latestUnreadId > 0 ? latestUnreadId : 0
  if (id <= 0) {
    // 无 id（旧响应/兜底）：保留原翻转语义
    if (!previous) showBrowserNotification(type)
    return
  }
  if (!document.hidden) {
    lastNotifiedUnreadId = Math.max(lastNotifiedUnreadId, id)
    return
  }
  // 页面在后台
  if (id > lastNotifiedUnreadId) {
    // 先推进再尝试弹：去重窗口/权限缺失抑制时避免下轮对同一 id 重复尝试
    lastNotifiedUnreadId = id
    if (Number.isSafeInteger(id + 1)) return showSourceNotification(id, type)
  }
}
