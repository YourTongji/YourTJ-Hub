import type { CampusCalendarSettings, CampusCalendarDraft, CampusCalendarParseRequest } from '@gooseforum/client'
import { i18n, currentLocale } from '@/runtime/i18n'
const messages: Record<string, string> = {
  'campus.rulesInvalid': 'campus.errorRulesInvalid',
  'campus.rulesChanged': 'campus.errorRulesChanged',
  'campus.rulesAIUnavailable': 'campus.errorRulesAI',
  'campus.rulesAIOutput': 'campus.errorRulesOutput',
  'campus.rulesLimited': 'campus.errorRulesLimited',
}
async function request<T>(path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  const response = await fetch(path, { method: body === undefined ? 'GET' : 'POST', credentials: 'same-origin', cache: 'no-store', signal,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' }, body: body === undefined ? undefined : JSON.stringify(body) })
  const data = await response.json()
  if (!response.ok || data.code !== 0) throw new Error(currentLocale() === 'zh' && data.messageCode === 'campus.rulesInvalid' && typeof data.params?.reason === 'string'
    ? data.params.reason : i18n.global.t(Object.hasOwn(messages, data.messageCode) ? messages[data.messageCode]! : 'campus.errorRules'))
  return data.result as T
}
export const calendarRulesAPI = {
  read: (admin = false, signal?: AbortSignal) => request<CampusCalendarSettings>(`/api/${admin ? 'admin/campus' : 'campus'}/calendar-rules`, undefined, signal),
  save: (settings: CampusCalendarSettings, signal?: AbortSignal) => request<CampusCalendarSettings>('/api/admin/campus/calendar-rules', settings, signal),
  parse: (input: CampusCalendarParseRequest, signal?: AbortSignal) => request<CampusCalendarDraft>('/api/admin/campus/calendar-rules/parse', input, signal),
}
export function teachingDateLabel(value: string, locale: string = currentLocale()): string {
  const d = new Date(`${value}T12:00:00+08:00`)
  return Number.isNaN(d.valueOf()) ? value : new Intl.DateTimeFormat(locale, { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', weekday: 'short' }).format(d)
}
