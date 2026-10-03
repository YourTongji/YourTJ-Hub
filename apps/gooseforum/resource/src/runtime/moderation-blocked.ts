import { reactive } from 'vue'
import { ApiResponseError } from './api'

// 发布被 AI 图文审查拦截时的友好提示（issue #975）：全站单例状态，
// AppShell 只挂载一个 ModerationBlockedDialog，各编辑入口共用。
export type ModerationBlockKind = 'policy' | 'externalImage'

const BLOCK_CODES: Record<string, ModerationBlockKind> = {
  'content.aiModeration.blocked': 'policy',
  'content.aiModeration.externalImageBlocked': 'externalImage',
}

export const moderationBlockedState = reactive({
  open: false,
  kind: 'policy' as ModerationBlockKind,
  message: '',
})

/** 错误属于 AI 审查拦截时返回拦截类型，否则返回 null。 */
export function moderationBlockKind(error: unknown): ModerationBlockKind | null {
  if (!(error instanceof ApiResponseError) || !error.messageCode) return null
  return BLOCK_CODES[error.messageCode] ?? null
}

/** 若为 AI 审查拦截则弹出提示并返回 true；调用方仍保留编辑器内容与行内错误。 */
export function showModerationBlocked(error: unknown): boolean {
  const kind = moderationBlockKind(error)
  if (!kind) return false
  moderationBlockedState.kind = kind
  moderationBlockedState.message = error instanceof Error ? error.message : ''
  moderationBlockedState.open = true
  return true
}

export function closeModerationBlocked() {
  moderationBlockedState.open = false
}
