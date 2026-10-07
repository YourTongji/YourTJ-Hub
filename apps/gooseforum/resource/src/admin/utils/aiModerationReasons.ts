import type { AiModerationReason } from '@/admin/types'

// AI 审查结论原因的本地化文案（issue #975）：决策记录与审核队列共用，
// 让管理员直接看到“为什么是这个结果”，而不是自己对照分数线推算。
type Translate = (key: string, values?: Record<string, unknown>) => string

const POLICY_KEYS: Record<string, string> = {
  adult: 'policyAdult',
  political_sensitive: 'policyPolitical',
  violence: 'policyViolence',
  illegal_or_dangerous: 'policyIllegal',
  other: 'policyOther',
}

const EVIDENCE_KEYS: Record<string, string> = {
  unavailable: 'evidenceUnavailable',
  external: 'evidenceExternal',
  too_many: 'evidenceTooMany',
  jev_failed: 'evidenceJevFailed',
  not_configured: 'evidenceNotConfigured',
  rate_limited: 'evidenceRateLimited',
  text_truncated: 'evidenceTextTruncated',
}

const fmt = (value?: number) => (value ?? 0).toFixed(2)

export function policyName(t: Translate, key: string, labels: Record<string, string> = {}) {
  return labels[key] || (POLICY_KEYS[key] ? t(`aiModerationAdmin.${POLICY_KEYS[key]}`) : key)
}

export function reasonText(t: Translate, reason: AiModerationReason, labels: Record<string, string> = {}) {
  const values = {
    policy: policyName(t, reason.policy ?? '', labels),
    score: fmt(reason.score),
    threshold: fmt(reason.threshold),
    limit: reason.code === 'severity_escalation' ? (reason.limit ?? 0).toFixed(1) : fmt(reason.limit),
    severity: (reason.severity ?? 0).toFixed(1),
    detail: reason.detail && EVIDENCE_KEYS[reason.detail] ? t(`aiModerationAdmin.${EVIDENCE_KEYS[reason.detail]}`) : reason.detail ?? '',
  }
  const keys: Record<string, string> = {
    block_threshold: 'reasonBlockThreshold',
    severity_escalation: 'reasonSeverityEscalation',
    between_thresholds: 'reasonBetweenThresholds',
    review_only_rule: 'reasonReviewOnlyRule',
    review_needed: 'reasonReviewNeeded',
    evidence_incomplete: 'reasonEvidenceIncomplete',
    external_image_blocked: 'reasonExternalImageBlocked',
  }
  return keys[reason.code] ? t(`aiModerationAdmin.${keys[reason.code]}`, values) : reason.code
}
