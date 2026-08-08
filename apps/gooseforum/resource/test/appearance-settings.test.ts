import { describe, expect, test } from 'vitest'
import {
  DEFAULT_APPEARANCE_SETTINGS,
  normalizeAppearanceSettings,
  resolveFontFamily,
} from '../src/runtime/appearance-settings'

describe('normalizeAppearanceSettings', () => {
  test('returns defaults for null/undefined/garbage/empty', () => {
    for (const raw of [null, undefined, 'nope', [], {}, 42]) {
      expect(normalizeAppearanceSettings(raw)).toEqual(DEFAULT_APPEARANCE_SETTINGS)
    }
  })

  test('clamps fontSize into [14, 20] and rejects non-numbers', () => {
    expect(normalizeAppearanceSettings({ fontSize: 12 }).fontSize).toBe(14)
    expect(normalizeAppearanceSettings({ fontSize: 24 }).fontSize).toBe(20)
    expect(normalizeAppearanceSettings({ fontSize: 18 }).fontSize).toBe(18)
    expect(normalizeAppearanceSettings({ fontSize: '18' }).fontSize).toBe(16)
    expect(normalizeAppearanceSettings({ fontSize: Number.NaN }).fontSize).toBe(16)
  })

  test('falls back to system preset for unknown fontFamilyPreset', () => {
    expect(normalizeAppearanceSettings({ fontFamilyPreset: 'times' }).fontFamilyPreset).toBe('system')
    expect(normalizeAppearanceSettings({ fontFamilyPreset: 'kai' }).fontFamilyPreset).toBe('kai')
  })

  test('truncates overlong customFontFamily and defaults non-strings', () => {
    expect(normalizeAppearanceSettings({ customFontFamily: 'x'.repeat(500) }).customFontFamily).toHaveLength(200)
    expect(normalizeAppearanceSettings({ customFontFamily: 42 }).customFontFamily).toBe('')
    expect(normalizeAppearanceSettings({ customFontFamily: 'Noto Serif SC' }).customFontFamily).toBe('Noto Serif SC')
  })

  test('clickAnimation must be exactly true', () => {
    expect(normalizeAppearanceSettings({ clickAnimation: true }).clickAnimation).toBe(true)
    expect(normalizeAppearanceSettings({ clickAnimation: false }).clickAnimation).toBe(false)
    expect(normalizeAppearanceSettings({ clickAnimation: 1 }).clickAnimation).toBe(false)
  })

  test('preserves a fully valid object', () => {
    const input = { fontSize: 18, fontFamilyPreset: 'custom', customFontFamily: 'Kaiti SC', clickAnimation: true }
    expect(normalizeAppearanceSettings(input)).toEqual(input)
  })
})

describe('resolveFontFamily', () => {
  test('system preset resolves to empty string', () => {
    expect(resolveFontFamily({ ...DEFAULT_APPEARANCE_SETTINGS, fontFamilyPreset: 'system' })).toBe('')
  })

  test('serif/kai/hei resolve to preset stacks', () => {
    expect(resolveFontFamily({ ...DEFAULT_APPEARANCE_SETTINGS, fontFamilyPreset: 'serif' })).toContain('Songti SC')
    expect(resolveFontFamily({ ...DEFAULT_APPEARANCE_SETTINGS, fontFamilyPreset: 'kai' })).toContain('Kaiti SC')
    expect(resolveFontFamily({ ...DEFAULT_APPEARANCE_SETTINGS, fontFamilyPreset: 'hei' })).toContain('Heiti SC')
  })

  test('custom preset uses trimmed customFontFamily', () => {
    const base = { ...DEFAULT_APPEARANCE_SETTINGS, fontFamilyPreset: 'custom' as const }
    expect(resolveFontFamily({ ...base, customFontFamily: '  Kaiti SC  ' })).toBe('Kaiti SC')
    expect(resolveFontFamily({ ...base, customFontFamily: '   ' })).toBe('')
  })
})
