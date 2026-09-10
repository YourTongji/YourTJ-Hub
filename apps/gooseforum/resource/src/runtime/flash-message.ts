import { readonly, ref } from 'vue'

export type FlashMessageType = 'success' | 'info' | 'warning' | 'error'

export interface FlashMessage {
  id: number
  type: FlashMessageType
  message: string
  /** 自动关闭总时长（毫秒） */
  durationMs: number
  /** 创建时间戳，用于进度条与剩余时间 */
  createdAt: number
}

const STORAGE_KEY = 'goose:flash-messages'
const MAX_VISIBLE_MESSAGES = 5
/** 默认展示时长：足够读完短句，又不拖沓 */
export const FLASH_DEFAULT_DURATION_MS = 5200

const messages = ref<FlashMessage[]>([])
let nextId = 1
let hydrated = false

interface DismissTimerState {
  timeoutId: number
  /** 剩余毫秒（暂停时冻结） */
  remainingMs: number
  /** 本段计时开始的 performance.now() */
  segmentStartedAt: number
  paused: boolean
}

const dismissTimers = new Map<number, DismissTimerState>()

function readStoredMessages(): Array<{ type: FlashMessageType; message: string }> {
  try {
    const raw = window.sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    window.sessionStorage.removeItem(STORAGE_KEY)
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed
      .map((item) => ({
        type: normalizeType(item?.type),
        message: String(item?.message || '').trim(),
      }))
      .filter((item) => item.message)
  } catch {
    return []
  }
}

function normalizeType(type: unknown): FlashMessageType {
  if (type === 'success' || type === 'info' || type === 'warning' || type === 'error') {
    return type
  }
  return 'info'
}

function scheduleDismiss(id: number, remainingMs: number) {
  const existing = dismissTimers.get(id)
  if (existing?.timeoutId) window.clearTimeout(existing.timeoutId)

  const segmentStartedAt = performance.now()
  const timeoutId = window.setTimeout(() => dismiss(id), Math.max(0, remainingMs))
  dismissTimers.set(id, {
    timeoutId,
    remainingMs,
    segmentStartedAt,
    paused: false,
  })
}

function push(message: string, type: FlashMessageType = 'info', durationMs = FLASH_DEFAULT_DURATION_MS) {
  const text = message.trim()
  if (!text) return

  const item: FlashMessage = {
    id: nextId++,
    type,
    message: text,
    durationMs: Math.max(1200, durationMs),
    createdAt: Date.now(),
  }

  const overflow = messages.value.length - MAX_VISIBLE_MESSAGES + 1
  if (overflow > 0) {
    messages.value.slice(0, overflow).forEach((messageItem) => dismiss(messageItem.id))
  }

  messages.value = [...messages.value, item]
  scheduleDismiss(item.id, item.durationMs)
}

/** 悬停暂停自动关闭，避免用户读一半消失 */
export function pauseFlashDismiss(id: number) {
  const timer = dismissTimers.get(id)
  if (!timer || timer.paused) return
  const elapsed = performance.now() - timer.segmentStartedAt
  const remainingMs = Math.max(0, timer.remainingMs - elapsed)
  window.clearTimeout(timer.timeoutId)
  dismissTimers.set(id, {
    timeoutId: 0,
    remainingMs,
    segmentStartedAt: performance.now(),
    paused: true,
  })
}

/** 移出后按剩余时间继续倒计时 */
export function resumeFlashDismiss(id: number) {
  const timer = dismissTimers.get(id)
  if (!timer || !timer.paused) return
  scheduleDismiss(id, timer.remainingMs)
}

export function queueFlashMessage(message: string, type: FlashMessageType = 'info') {
  const text = message.trim()
  if (!text) return
  try {
    const raw = window.sessionStorage.getItem(STORAGE_KEY)
    const existing = raw ? JSON.parse(raw) : []
    const list = Array.isArray(existing) ? existing : []
    list.push({ type, message: text })
    window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(list.slice(-4)))
  } catch {
    push(text, type)
  }
}

export function hydrateFlashMessages() {
  if (hydrated) return
  hydrated = true
  readStoredMessages().forEach((item) => push(item.message, item.type))
}

export function dismiss(id: number) {
  const timer = dismissTimers.get(id)
  if (timer?.timeoutId) window.clearTimeout(timer.timeoutId)
  dismissTimers.delete(id)
  messages.value = messages.value.filter((item) => item.id !== id)
}

export function useFlashMessages() {
  return {
    messages: readonly(messages),
    push,
    dismiss,
    pause: pauseFlashDismiss,
    resume: resumeFlashDismiss,
  }
}
