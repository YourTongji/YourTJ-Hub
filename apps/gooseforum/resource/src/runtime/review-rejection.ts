export const rejectionPresets = ['rules', 'spam', 'privacy', 'offtopic', 'other'] as const
export type RejectionPreset = typeof rejectionPresets[number]
export interface RejectionDraft { preset: RejectionPreset | ''; note: string }
export const emptyRejectionDraft = (): RejectionDraft => ({ preset: '', note: '' })

/** Existing author-visible plain text; no model evidence or internal notes. */
export function rejectionReason(t: (key: string) => string, draft: RejectionDraft): string {
  if (!rejectionPresets.includes(draft.preset as RejectionPreset)) return ''
  const note = draft.note.trim()
  if (draft.preset === 'other') return note
  const standard = t(`reviewRejection.reasons.${draft.preset}`)
  return note ? `${standard}\n${note}` : standard
}
export function rejectionError(t: (key: string) => string, reason: string): string {
  if (!reason.trim()) return t('reviewRejection.required')
  return Array.from(reason).length > 512 ? t('reviewRejection.tooLong') : ''
}
