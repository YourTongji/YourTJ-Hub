import type { CampusCalendarSettings, CampusCalendarDraft, CampusCalendarParseRequest } from '@gooseforum/client'
const messages: Record<string, string> = {
  'campus.rulesInvalid': '规则有冲突或日期无效，请检查后重试。',
  'campus.rulesChanged': '其他管理员已更新规则，请重新加载后编辑。当前草稿尚未保存。',
  'campus.rulesAIUnavailable': 'AI 暂不可用，请检查“AI 课程总结”的模型连接配置，或手动添加规则。',
  'campus.rulesAIOutput': 'AI 未生成有效规则，请补充完整日期和教学对应关系后重试，或手动编辑。',
  'campus.rulesLimited': 'AI 调用较频繁，请稍后再试。',
}
async function request<T>(path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  const response = await fetch(path, { method: body === undefined ? 'GET' : 'POST', credentials: 'same-origin', cache: 'no-store', signal,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' }, body: body === undefined ? undefined : JSON.stringify(body) })
  const data = await response.json()
  if (!response.ok || data.code !== 0) throw new Error(data.messageCode === 'campus.rulesInvalid' && typeof data.params?.reason === 'string'
    ? data.params.reason : messages[data.messageCode] || '无法读取或保存调休规则，请稍后重试。')
  return data.result as T
}
export const calendarRulesAPI = {
  read: (admin = false, signal?: AbortSignal) => request<CampusCalendarSettings>(`/api/${admin ? 'admin/campus' : 'campus'}/calendar-rules`, undefined, signal),
  save: (settings: CampusCalendarSettings, signal?: AbortSignal) => request<CampusCalendarSettings>('/api/admin/campus/calendar-rules', settings, signal),
  parse: (input: CampusCalendarParseRequest, signal?: AbortSignal) => request<CampusCalendarDraft>('/api/admin/campus/calendar-rules/parse', input, signal),
}
export function teachingDateLabel(value: string): string {
  const d = new Date(`${value}T12:00:00+08:00`)
  return Number.isNaN(d.valueOf()) ? value : `${value}（${new Intl.DateTimeFormat('zh-CN', { timeZone: 'Asia/Shanghai', weekday: 'short' }).format(d)}）`
}
