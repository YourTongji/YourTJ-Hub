import type { CampusDataset, CampusDatasetKey, CampusStatus, CampusMessageDetail } from '@gooseforum/client'
const messages: Record<string, string> = {
  'campus.disabled': '校园连接尚未启用，请联系站点管理员。',
  'campus.upstreamUnavailable': '学校服务暂时不可用，请稍后重试。',
  'campus.authorizationRequired': '需要重新授权以继续同步，已绑定的身份仍然保留。',
  'campus.authorizationExpired': '这次认证已过期或失效，请重新发起。',
  'campus.identityUnavailable': '该身份无法绑定，可能已被其他账号绑定，或连接状态已改变。原绑定未改变。',
  'campus.connectionChanged': '绑定状态已改变，请刷新页面后重试。',
  'campus.messageAuthorizationRequired': '查看消息正文需要更新学校授权，当前身份绑定会保留。',
  'campus.messageUnavailable': '这条消息已失效或不在你的消息列表中。',
}
export class CampusError extends Error { constructor(public code: string, message: string) { super(message) } }
async function request<T>(path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`/api/campus/${path}`, {
    method: body === undefined ? 'GET' : 'POST', credentials: 'same-origin', cache: 'no-store', signal,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const data = await response.json()
  if (!response.ok || data.code !== 0) throw new CampusError(data.messageCode || '', messages[data.messageCode] || (response.status === 401 ? '请先登录 YourTJ。' : response.status === 429 ? '请求较频繁，请稍后重试。' : '操作失败，请稍后重试。'))
  return data.result as T
}
export const campusAPI = {
  status: (signal?: AbortSignal) => request<CampusStatus>('status', undefined, signal),
  start: (mode: 'bind' | 'replace' | 'reauthorize') => request<{ url: string }>('tongji/start', { mode }),
  confirm: () => request<null>('tongji/confirm', {}),
  unbind: (revision: string) => request<null>('tongji/unbind', { revision }),
  dataset: (key: CampusDatasetKey, signal?: AbortSignal) => request<CampusDataset>(`data/${key}`, undefined, signal),
  message: (id: string, signal?: AbortSignal) => request<CampusMessageDetail>(`messages/${encodeURIComponent(id)}`, undefined, signal),
}
