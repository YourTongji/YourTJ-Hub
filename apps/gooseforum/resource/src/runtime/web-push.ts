// Web Push 运行时封装：能力检测、实例配置读取、Service Worker 注册、
// 浏览器订阅生命周期与后端订阅持久化的桥接。
//
// 与 #449 的页面内浏览器通知不同，Web Push 在页面完全关闭后仍由浏览器
// 投递（Service Worker 接收 push 事件）。通道依赖实例 [webpush] VAPID 配置
// （configured=false 时前端不显示开关）；浏览器能力（SW + PushManager +
// secure context）缺失时同样隐藏。
//
// 本模块只抛 PushError（携带稳定 code），用户可见文案由调用方按 i18n 转译。

export type PushErrorCode =
  | 'unsupported' // 浏览器不支持 Web Push（无 SW/PushManager/非 secure context）
  | 'unconfigured' // 实例未配置 VAPID（通道关闭）
  | 'permission-denied' // 用户拒绝推送授权
  | 'subscribe-failed' // PushManager.subscribe 失败（含 iOS 非主屏等）
  | 'unsubscribe-failed' // 浏览器侧退订失败
  | 'network' // 后端接口失败

export class PushError extends Error {
  code: PushErrorCode
  constructor(code: PushErrorCode) {
    super(code)
    this.code = code
    this.name = 'PushError'
  }
}

export interface PushConfig {
  configured: boolean
  applicationServerKey?: string
}

// SW_URL 必须稳定（不随构建 hash 变化）：注册同 URL 不会重装，随机 query
// 会让每次页面加载都视为「新 SW」并破坏订阅关联。
const SW_URL = '/sw.js'

/** 浏览器是否具备 Web Push 前置能力。 */
export function isWebPushSupported(): boolean {
  return typeof window !== 'undefined'
    && window.isSecureContext
    && 'serviceWorker' in navigator
    && 'PushManager' in window
    && 'Notification' in window
}

/** 读取实例推送配置（GET /api/forum/push/config）。 */
export async function fetchPushConfig(): Promise<PushConfig> {
  const response = await fetch('/api/forum/push/config', {
    headers: { Accept: 'application/json' },
  })
  if (!response.ok) throw new PushError('network')
  const data = await response.json() as {
    code?: number
    result?: PushConfig
  }
  if (data.code !== undefined && data.code !== 0) throw new PushError('network')
  const result = data.result
  return {
    configured: Boolean(result?.configured),
    applicationServerKey: result?.applicationServerKey,
  }
}

/** 注册根 scope Service Worker（幂等）。 */
export async function ensureServiceWorker(): Promise<ServiceWorkerRegistration> {
  return navigator.serviceWorker.register(SW_URL, { updateViaCache: 'none' })
}

/** 返回当前浏览器已存在的推送订阅（未开启时为 null）。 */
export async function currentPushSubscription(): Promise<PushSubscription | null> {
  if (!isWebPushSupported()) return null
  const registration = await navigator.serviceWorker.getRegistration(SW_URL).catch(() => null)
  if (!registration?.pushManager) return null
  return registration.pushManager.getSubscription()
}

// base64url 字符串 → Uint8Array（PushManager.subscribe 的
// applicationServerKey 需要 ArrayBuffer；浏览器只接受 P-256 未压缩点）。
function urlBase64ToUint8Array(base64url: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64url.length % 4)) % 4)
  const base64 = (base64url + padding).replace(/-/g, '+').replace(/_/g, '/')
  const raw = atob(base64)
  const output = new Uint8Array(raw.length)
  for (let i = 0; i < raw.length; i += 1) output[i] = raw.charCodeAt(i)
  return output
}

/** 后端持久化订阅（POST /api/forum/push/subscribe）。 */
async function persistSubscription(subscription: PushSubscription, lang: string): Promise<void> {
  const json = subscription.toJSON() as {
    endpoint: string
    keys: { p256dh: string; auth: string }
  }
  const response = await fetch('/api/forum/push/subscribe', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      subscription: {
        endpoint: json.endpoint,
        keys: { p256dh: json.keys.p256dh, auth: json.keys.auth },
      },
      lang,
    }),
  })
  const data = await response.json().catch(() => undefined) as { code?: number } | undefined
  if (!response.ok || (data && data.code !== undefined && data.code !== 0)) {
    throw new PushError('network')
  }
}

/**
 * 开启 Web Push：注册 SW → subscribe（浏览器授权）→ 后端持久化。
 * 必须由用户手势触发（浏览器要求 subscribe 在用户激活上下文中调用）。
 */
export async function enableWebPush(lang: string): Promise<void> {
  if (!isWebPushSupported()) throw new PushError('unsupported')
  const config = await fetchPushConfig()
  if (!config.configured || !config.applicationServerKey) throw new PushError('unconfigured')

  const registration = await ensureServiceWorker()
  if (!registration.pushManager) throw new PushError('unsupported')

  let subscription: PushSubscription
  try {
    subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(config.applicationServerKey),
    })
  } catch (error) {
    // 浏览器拒绝授权（NotAllowedError）与 iOS 非主屏（AbortError/NotAllowedError）
    // 都归入可解释错误，由调用方展示 permission 文案。
    if (error instanceof DOMException && error.name === 'NotAllowedError') {
      throw new PushError('permission-denied')
    }
    throw new PushError('subscribe-failed')
  }

  await persistSubscription(subscription, lang)
}

/**
 * 关闭 Web Push：后端解绑 endpoint → 浏览器侧退订。
 * 幂等：没有订阅时静默成功。
 */
export async function disableWebPush(): Promise<void> {
  const subscription = await currentPushSubscription()
  if (!subscription) return
  const json = subscription.toJSON() as { endpoint: string }
  // 后端先删（含 endpoint 不属于当前用户时静默成功），再浏览器退订；
  // 任一侧失败都不应让本地状态卡在「仍开启」。
  await fetch('/api/forum/push/unsubscribe', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ endpoint: json.endpoint }),
  }).catch(() => undefined)
  try {
    await subscription.unsubscribe()
  } catch {
    throw new PushError('unsubscribe-failed')
  }
}
