import { describe, expect, test } from 'vitest'
import {
  DEFAULT_APPEARANCE_SETTINGS,
  isFontPristine,
  MAX_CUSTOM_CSS_LENGTH,
  normalizeAppearanceSettings,
  resolveFontFamily,
} from '../src/runtime/appearance-settings'

const def = () => ({ ...DEFAULT_APPEARANCE_SETTINGS })

describe('normalizeAppearanceSettings', () => {
  test('returns defaults for garbage / null / empty', () => {
    for (const raw of [null, undefined, 'x', [], {}, 42]) {
      expect(normalizeAppearanceSettings(raw)).toEqual(def())
    }
  })

  test('normalizes zones with clamping 12..24 and per-zone defaults', () => {
    const s = normalizeAppearanceSettings({ zones: {} })
    expect(s.zones.ui.size).toBe(16)
    expect(s.zones.body.size).toBe(16)
    expect(s.zones.code.size).toBe(14)
    expect(s.zones.ui.familyPreset).toBe('system')
    expect(s.zones.code.familyPreset).toBe('mono')
    const s2 = normalizeAppearanceSettings({ zones: { ui: { size: 30, familyPreset: 'kai' } } })
    expect(s2.zones.ui.size).toBe(24)
    expect(s2.zones.ui.familyPreset).toBe('kai')
    expect(s2.zones.body.size).toBe(16)
  })

  test('blank custom preset falls back to system', () => {
    const s = normalizeAppearanceSettings({ zones: { body: { familyPreset: 'custom', customFamily: '   ' } } })
    expect(s.zones.body.familyPreset).toBe('system')
  })

  test('migrates legacy single font shape to all zones', () => {
    const s = normalizeAppearanceSettings({ fontSize: 18, fontFamilyPreset: 'serif', customFontFamily: '' })
    expect(s.zones.ui.size).toBe(18)
    expect(s.zones.body.size).toBe(18)
    expect(s.zones.code.size).toBe(18)
    expect(s.zones.ui.familyPreset).toBe('serif')
    expect(s.zones.code.familyPreset).toBe('serif')
  })

  test('truncates customFamily and customCss', () => {
    const s = normalizeAppearanceSettings({
      zones: { ui: { customFamily: 'x'.repeat(500) } },
      customCss: 'y'.repeat(MAX_CUSTOM_CSS_LENGTH + 10),
    })
    expect(s.zones.ui.customFamily).toHaveLength(200)
    expect(s.customCss).toHaveLength(256 * 1024)
  })

  test('clickAnimation must be exactly true', () => {
    expect(normalizeAppearanceSettings({ clickAnimation: true }).clickAnimation).toBe(true)
    expect(normalizeAppearanceSettings({ clickAnimation: 1 }).clickAnimation).toBe(false)
  })
})

describe('resolveFontFamily', () => {
  test('returns preset stacks for each family', () => {
    for (const p of ['system', 'serif', 'kai', 'hei', 'mono'] as const) {
      expect(resolveFontFamily({ size: 16, familyPreset: p, customFamily: '' })).toMatch(/^["a-z-]/)
    }
    expect(resolveFontFamily({ size: 16, familyPreset: 'serif', customFamily: '' })).toContain('Songti SC')
    expect(resolveFontFamily({ size: 16, familyPreset: 'mono', customFamily: '' })).toContain('ui-monospace')
  })

  test('custom uses trimmed value, blank custom falls back to system stack', () => {
    expect(resolveFontFamily({ size: 16, familyPreset: 'custom', customFamily: '  KaiTi  ' })).toBe('KaiTi')
    expect(resolveFontFamily({ size: 16, familyPreset: 'custom', customFamily: '   ' })).toContain('-apple-system')
  })
})

describe('isFontPristine', () => {
  test('true when all zones at defaults', () => {
    expect(isFontPristine(def())).toBe(true)
  })
  test('false when any zone differs', () => {
    expect(isFontPristine({ ...def(), zones: { ...def().zones, ui: { size: 18, familyPreset: 'system', customFamily: '' } } })).toBe(false)
    expect(isFontPristine({ ...def(), zones: { ...def().zones, code: { size: 14, familyPreset: 'serif', customFamily: '' } } })).toBe(false)
  })
})
