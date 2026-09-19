import { courseColorSlotFor, courseSlotVar } from './courseColors'

export function timetableCourseStyle(seed: string, custom = false): Record<string, string> {
  if (custom) {
    return {
      '--card-accent': 'var(--gf-color-base-content)',
      '--card-bg': 'color-mix(in oklab, var(--gf-color-base-200) 80%, var(--gf-color-base-100))',
      '--card-bg-hover': 'var(--gf-color-base-200)',
      '--card-border': 'var(--gf-color-line)',
      '--card-title': 'var(--gf-color-base-content)',
      '--card-sub': 'color-mix(in oklab, var(--gf-color-base-content) 70%, transparent)',
      '--card-shadow-hover': '0 2px 8px -2px rgba(0, 0, 0, 0.08), 0 1px 3px -1px rgba(0, 0, 0, 0.04)',
      backgroundColor: 'var(--card-bg)',
      borderColor: 'var(--card-border)',
      color: 'var(--card-title)',
    }
  }
  const slot = courseColorSlotFor(seed)
  const slotVar = courseSlotVar(slot)
  return {
    '--card-accent': `var(${slotVar})`,
    // 借鉴参考图的柔和莫兰迪/马卡龙粉彩色底（11% 槽位色轻盈融合）
    '--card-bg': `color-mix(in oklab, var(${slotVar}) 11%, var(--gf-color-base-100))`,
    '--card-bg-hover': `color-mix(in oklab, var(${slotVar}) 17%, var(--gf-color-base-100))`,
    // 极轻微的同色系半透细边框，呈现「不包裹」的自然悬浮感
    '--card-border': `color-mix(in oklab, var(${slotVar}) 18%, transparent)`,
    // 标题文字：以 base-content 为底混入 45% 槽位色，确保与浅色/深色底对比度均 ≥8:1
    '--card-title': `color-mix(in oklab, var(${slotVar}) 45%, var(--gf-color-base-content))`,
    // 次级文字（教师、周次）
    '--card-sub': `color-mix(in oklab, var(${slotVar}) 25%, var(--gf-color-base-content))`,
    '--card-badge-bg': `color-mix(in oklab, var(${slotVar}) 12%, transparent)`,
    // 自然柔和环境光沉降阴影
    '--card-shadow-hover': '0 3px 10px -2px rgba(0, 0, 0, 0.08), 0 1px 3px -1px rgba(0, 0, 0, 0.04)',
    backgroundColor: 'var(--card-bg)',
    borderColor: 'var(--card-border)',
    color: 'var(--card-title)',
  }
}
