import { computed, readonly, ref } from 'vue'
import { i18n } from './i18n'
import { resolveApiMessage } from './api-message'
import { maybeNotifyUnread } from './browser-notification'
import { setUnreadMessagesDocumentTitle } from './document-title'
import type { UnreadBadgeStyle, UnreadStatusPayload } from '@gooseforum/client'

const CACHE_TTL = 10_000
const POLL_INTERVAL = 30_000
const notifications = ref(false)
const messages = ref(false)
const moderationReports = ref(false)
const latestNotificationType = ref('')
const latestUnreadId = ref(0)
const notificationCount = ref<number>()
const messageCount = ref<number>()
const badgeStyle = ref<UnreadBadgeStyle>('number')
const badgeStyleSaving = ref(false)
const badgeStyleError = ref('')
const checked = ref(false)
let ownerId = 0
let ownerGeneration = 0
let readGeneration = 0
let preferenceRevision = 0
let styleInitialized = false
let inFlight: Promise<UnreadStatusPayload> | null = null
let requestAbort: AbortController | undefined
let pollTimer: number | undefined

const notificationMessage = computed(() => {
  if (latestNotificationType.value === 'comment' || latestNotificationType.value === 'topic_post') return i18n.global.t('notifications.newComment')
  if (notifications.value) return i18n.global.t('notifications.newNotification')
  return i18n.global.t('notifications.noUnread')
})
const countValue = (value: unknown): number | undefined =>
  typeof value === 'number' && Number.isSafeInteger(value) && value >= 0 ? value : undefined

function normalizeStatus(data: Partial<UnreadStatusPayload> | null | undefined): UnreadStatusPayload {
  return {
    notifications: Boolean(data?.notifications),
    messages: Boolean(data?.messages),
    moderationReports: Boolean(data?.moderationReports),
    latestNotificationType: data?.latestNotificationType || '',
    latestUnreadId: typeof data?.latestUnreadId === 'number' ? data.latestUnreadId : 0,
    notificationCount: countValue(data?.notificationCount),
    messageCount: countValue(data?.messageCount),
    badgeStyle: data?.badgeStyle === 'dot' || data?.badgeStyle === 'number' ? data.badgeStyle : undefined,
  }
}
function snapshot(): UnreadStatusPayload {
  return {
    notifications: notifications.value, messages: messages.value,
    moderationReports: moderationReports.value, latestNotificationType: latestNotificationType.value,
    latestUnreadId: latestUnreadId.value, notificationCount: notificationCount.value,
    messageCount: messageCount.value, badgeStyle: badgeStyle.value,
  }
}
function readCache(): UnreadStatusPayload | null {
  if (!ownerId) return null
  try {
    const raw = localStorage.getItem(`goose:unread-status:${ownerId}`)
    if (!raw) return null
    const cached = JSON.parse(raw) as { data?: UnreadStatusPayload; timestamp?: number }
    if (!cached.timestamp || Date.now() - cached.timestamp > CACHE_TTL) return null
    return normalizeStatus(cached.data)
  } catch { return null }
}
function writeCache() {
  if (!ownerId) return
  try {
    localStorage.setItem(`goose:unread-status:${ownerId}`, JSON.stringify({ data: snapshot(), timestamp: Date.now() }))
  } catch { /* Polling and display must remain usable without local storage. */ }
}
function applyUnread(data: Partial<UnreadStatusPayload> | null | undefined, options: { includeStyle?: boolean; styleRevision?: number } = {}) {
  const status = normalizeStatus(data)
  const hadNotifications = notifications.value
  notifications.value = status.notifications
  messages.value = status.messages
  moderationReports.value = status.moderationReports || false
  latestNotificationType.value = status.latestNotificationType || ''
  latestUnreadId.value = status.latestUnreadId || 0
  notificationCount.value = status.notificationCount
  messageCount.value = status.messageCount
  if (options.includeStyle && !badgeStyleSaving.value &&
      (options.styleRevision === undefined || options.styleRevision === preferenceRevision)) {
    badgeStyle.value = status.badgeStyle ?? 'number'
    styleInitialized = true
    preferenceRevision++
  }
  checked.value = true
  setUnreadMessagesDocumentTitle(status.messages)
  maybeNotifyUnread(hadNotifications, status.notifications, status.latestNotificationType || '', status.latestUnreadId || 0)
  writeCache()
}
async function fetchUnreadStatus(signal: AbortSignal) {
  const response = await fetch('/api/forum/unread-status', { headers: { Accept: 'application/json' }, signal })
  if (!response.ok) throw new Error(`HTTP ${response.status}`)
  const data = await response.json() as {
    code?: number; messageCode?: string; params?: Record<string, unknown>
    result?: UnreadStatusPayload; data?: UnreadStatusPayload
  }
  if (data.code !== undefined && data.code !== 0) throw new Error(resolveApiMessage(data, i18n.global.t('notifications.checkFailed')))
  return normalizeStatus(data.result ?? data.data)
}
async function refresh(force = false): Promise<UnreadStatusPayload> {
  if (!ownerId) return snapshot()
  if (!force) {
    const cached = readCache()
    if (cached) {
      applyUnread(cached)
      void refresh(true).catch(() => undefined)
      return cached
    }
  }
  if (!inFlight) {
    const captured = readGeneration
    const styleRevision = preferenceRevision
    const abort = requestAbort = new AbortController()
    const operation = fetchUnreadStatus(abort.signal).then(data => {
      if (captured === readGeneration) applyUnread(data, { includeStyle: true, styleRevision })
      return data
    })
    inFlight = operation
    void operation.finally(() => {
      if (inFlight === operation) inFlight = null
      if (requestAbort === abort) requestAbort = undefined
    }).catch(() => undefined)
  }
  return inFlight
}
function stopPolling(clearOwner = false) {
  if (pollTimer !== undefined) window.clearInterval(pollTimer)
  pollTimer = undefined
  readGeneration++
  requestAbort?.abort()
  requestAbort = undefined
  inFlight = null
  if (clearOwner) {
    ownerGeneration++
    ownerId = 0
    preferenceRevision++
    styleInitialized = false
    badgeStyle.value = 'number'
    badgeStyleSaving.value = false
    badgeStyleError.value = ''
    notifications.value = messages.value = moderationReports.value = checked.value = false
    latestNotificationType.value = ''
    latestUnreadId.value = 0
    notificationCount.value = messageCount.value = undefined
    setUnreadMessagesDocumentTitle(false)
  }
}
function startPolling(initial?: Partial<UnreadStatusPayload>, accountId = ownerId) {
  if (accountId !== ownerId) {
    stopPolling(true)
    ownerId = accountId
  }
  if (!ownerId) return
  if (initial != null) applyUnread(initial, { includeStyle: !styleInitialized })
  if (pollTimer !== undefined) return
  if (initial == null) {
    const cached = readCache()
    if (cached) applyUnread(cached, { includeStyle: !styleInitialized })
    void refresh(true).catch(() => undefined)
  }
  pollTimer = window.setInterval(() => { void refresh(true).catch(() => undefined) }, POLL_INTERVAL)
}
async function updateBadgeStyle(style: UnreadBadgeStyle): Promise<boolean> {
  if (!ownerId || !checked.value || badgeStyleSaving.value) return false
  if (style === badgeStyle.value) return true
  const capturedOwner = ownerGeneration
  const previous = badgeStyle.value
  const revision = ++preferenceRevision
  const current = () => capturedOwner === ownerGeneration && revision === preferenceRevision
  badgeStyleSaving.value = true
  badgeStyleError.value = ''
  badgeStyle.value = style
  let failure = ''
  try {
    const response = await fetch('/api/forum/notification-preferences', {
      method: 'PUT', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ badgeStyle: style }),
    })
    const body = await response.json()
    if (!response.ok || body.code !== 0) {
      failure = resolveApiMessage(body, i18n.global.t('api.operationFailed'))
      throw new Error('preference request failed')
    }
    if (body.result?.badgeStyle !== 'dot' && body.result?.badgeStyle !== 'number') throw new Error('invalid preference response')
    if (!current()) return false
    badgeStyle.value = body.result.badgeStyle
    writeCache()
    return true
  } catch {
    if (current()) {
      badgeStyle.value = previous
      badgeStyleError.value = failure || i18n.global.t('api.operationFailed')
      writeCache()
    }
    return false
  } finally {
    if (current()) {
      preferenceRevision++
      badgeStyleSaving.value = false
    }
  }
}
function clearNotifications() {
  applyUnread({ ...snapshot(), notifications: false, notificationCount: 0, latestNotificationType: '', latestUnreadId: 0 })
}
function setNotifications(hasUnread: boolean, count?: number) {
  applyUnread({ ...snapshot(), notifications: hasUnread, notificationCount: hasUnread ? countValue(count) : 0,
    latestNotificationType: hasUnread ? latestNotificationType.value : '', latestUnreadId: hasUnread ? latestUnreadId.value : 0 })
}
function clearMessages() {
  applyUnread({ ...snapshot(), messages: false, messageCount: 0 })
}
export function useUnreadStatus() {
  return {
    notifications: readonly(notifications), messages: readonly(messages),
    notificationCount: readonly(notificationCount), messageCount: readonly(messageCount),
    badgeStyle: readonly(badgeStyle), badgeStyleSaving: readonly(badgeStyleSaving), badgeStyleError: readonly(badgeStyleError),
    moderationReports: readonly(moderationReports), latestNotificationType: readonly(latestNotificationType),
    checked: readonly(checked), notificationMessage,
    startPolling, stopPolling, refresh, applyUnread, updateBadgeStyle, clearNotifications, setNotifications, clearMessages,
  }
}
