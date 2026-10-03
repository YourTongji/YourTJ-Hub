import { describe, expect, test } from 'vitest'
import { createI18n } from 'vue-i18n'
import zh from '../src/locales/zh'
import { reasonText } from '../src/admin/utils/aiModerationReasons'

const i18n = createI18n({ legacy: false, locale: 'zh', messages: { zh } })
const t = (key: string, values?: Record<string, unknown>) => i18n.global.t(key, values ?? {})

describe('AI moderation reasons (issue #975)', () => {
  test('explains why a block rule between the two lines still goes to human review', () => {
    const text = reasonText(t, { code: 'between_thresholds', policy: 'political_sensitive', score: 0.56, threshold: 0.5, limit: 0.9 })
    expect(text).toBe('政治敏感得分 0.56，在人工审核线 0.50 与拦截线 0.90 之间，进入人工审核。')
  })

  test('uses admin rule labels and localizes evidence details', () => {
    expect(reasonText(t, { code: 'block_threshold', policy: 'adult', score: 0.9, threshold: 0.9 }, { adult: '色情' }))
      .toBe('色情得分 0.90，达到拦截线 0.90，已拦截。')
    expect(reasonText(t, { code: 'evidence_incomplete', detail: 'unavailable' })).toBe('有图片未能识别，无法自动发布，进入人工审核。')
    expect(reasonText(t, { code: 'severity_escalation', policy: 'violence', score: 0.6, threshold: 0.5, severity: 2.8, limit: 2.5 }))
      .toBe('暴力血腥得分 0.60，达到人工审核线 0.50，且严重程度 2.8 达到提级线 2.5，已拦截。')
  })
})
