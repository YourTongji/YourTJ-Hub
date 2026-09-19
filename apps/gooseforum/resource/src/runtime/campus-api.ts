import type { CampusDataset, CampusDatasetKey, CampusStatus, CampusMessageDetail, CampusCalendarExport } from '@gooseforum/client'
import { i18n } from '@/runtime/i18n'
const messages: Record<string, string> = {
  'campus.disabled': 'campus.errorDisabled',
  'campus.upstreamUnavailable': 'campus.errorUpstream',
  'campus.authorizationRequired': 'campus.errorAuthorization',
  'campus.authorizationExpired': 'campus.errorExpired',
  'campus.identityUnavailable': 'campus.errorIdentity',
  'campus.connectionChanged': 'campus.errorChanged',
  'campus.messageAuthorizationRequired': 'campus.errorMessageAuthorization',
  'campus.messageUnavailable': 'campus.errorMessage',
  'campus.calendarIncomplete': 'campus.errorCalendar',
  'campus.rulesUnavailable': 'campus.errorRulesUnavailable',
  'campus.rulesInvalid': 'campus.errorPublishedRules',
  'campus.calendarEmpty': 'campus.errorEmptyCalendar',
}
export class CampusError extends Error { constructor(public code: string, message: string) { super(message) } }
async function request<T>(path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`/api/campus/${path}`, {
    method: body === undefined ? 'GET' : 'POST', credentials: 'same-origin', cache: 'no-store', signal,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const data = await response.json()
  if (!response.ok || data.code !== 0) throw new CampusError(data.messageCode || '', i18n.global.t(Object.hasOwn(messages, data.messageCode) ? messages[data.messageCode]! : response.status === 401 ? 'campus.errorLogin' : response.status === 429 ? 'campus.errorLimited' : 'campus.errorFailed'))
  return data.result as T
}
export const campusAPI = {
  status: (signal?: AbortSignal) => request<CampusStatus>('status', undefined, signal),
  start: (mode: 'bind' | 'replace' | 'reauthorize') => request<{ url: string }>('tongji/start', { mode }),
  confirm: () => request<null>('tongji/confirm', {}),
  unbind: (revision: string) => request<null>('tongji/unbind', { revision }),
  dataset: (key: CampusDatasetKey, signal?: AbortSignal) => request<CampusDataset>(`data/${key}`, undefined, signal),
  message: (id: string, signal?: AbortSignal) => request<CampusMessageDetail>(`messages/${encodeURIComponent(id)}`, undefined, signal),
  exportCalendar: (signal?: AbortSignal, applyAdjustments = true) => request<CampusCalendarExport>(`calendar-export?applyAdjustments=${applyAdjustments}`, undefined, signal),
}
