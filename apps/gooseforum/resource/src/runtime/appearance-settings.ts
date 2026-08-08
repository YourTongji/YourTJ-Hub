export type FontFamilyPreset = 'system' | 'serif' | 'kai' | 'hei' | 'custom'

export interface AppearanceSettings {
  fontSize: number
  fontFamilyPreset: FontFamilyPreset
  customFontFamily: string
  clickAnimation: boolean
}

const STORAGE_KEY = 'goose-appearance-settings'
const FONT_SIZE_MIN = 14
const FONT_SIZE_MAX = 20
const DEFAULT_FONT_SIZE = 16
const MAX_CUSTOM_FONT_LENGTH = 200

export const DEFAULT_APPEARANCE_SETTINGS: AppearanceSettings = Object.freeze({
  fontSize: DEFAULT_FONT_SIZE,
  fontFamilyPreset: 'system',
  customFontFamily: '',
  clickAnimation: false,
})

export const FONT_FAMILY_PRESETS: readonly FontFamilyPreset[] = [
  'system',
  'serif',
  'kai',
  'hei',
  'custom',
]

export const FONT_PRESETS: Record<'serif' | 'kai' | 'hei', string> = {
  serif: '"Songti SC", "SimSun", serif',
  kai: '"Kaiti SC", "KaiTi", "STKaiti", serif',
  hei: '"Heiti SC", "SimHei", "Noto Sans CJK SC", sans-serif',
}

export function normalizeAppearanceSettings(raw: unknown): AppearanceSettings {
  const source = isRecord(raw) ? raw : {}
  const fontSize = clampNumber(source.fontSize, FONT_SIZE_MIN, FONT_SIZE_MAX, DEFAULT_FONT_SIZE)
  let fontFamilyPreset = FONT_FAMILY_PRESETS.includes(source.fontFamilyPreset as FontFamilyPreset)
    ? (source.fontFamilyPreset as FontFamilyPreset)
    : 'system'
  const customFontFamily = typeof source.customFontFamily === 'string'
    ? source.customFontFamily.slice(0, MAX_CUSTOM_FONT_LENGTH)
    : ''
  if (fontFamilyPreset === 'custom' && customFontFamily.trim() === '') {
    fontFamilyPreset = 'system'
  }
  const clickAnimation = source.clickAnimation === true
  return { fontSize, fontFamilyPreset, customFontFamily, clickAnimation }
}

export function resolveFontFamily(settings: AppearanceSettings): string {
  if (settings.fontFamilyPreset === 'custom') return settings.customFontFamily.trim()
  if (settings.fontFamilyPreset === 'system') return ''
  return FONT_PRESETS[settings.fontFamilyPreset]
}

let clickAnimationEnabled = false

export function isClickAnimationEnabled(): boolean {
  return clickAnimationEnabled
}

export function loadAppearanceSettings(): AppearanceSettings {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (raw === null) return { ...DEFAULT_APPEARANCE_SETTINGS }
    return normalizeAppearanceSettings(JSON.parse(raw))
  } catch {
    return { ...DEFAULT_APPEARANCE_SETTINGS }
  }
}

export function applyAppearanceSettings(settings: AppearanceSettings) {
  clickAnimationEnabled = settings.clickAnimation
  if (settings.fontSize === DEFAULT_FONT_SIZE) {
    document.documentElement.style.removeProperty('font-size')
  } else {
    document.documentElement.style.fontSize = `${settings.fontSize}px`
  }
  const family = resolveFontFamily(settings)
  if (family) {
    document.documentElement.style.setProperty('--gf-font-family', family)
  } else {
    document.documentElement.style.removeProperty('--gf-font-family')
  }
}

export function applyStoredAppearanceSettings() {
  applyAppearanceSettings(loadAppearanceSettings())
}

export function saveAppearanceSettings(settings: AppearanceSettings) {
  applyAppearanceSettings(settings)
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
  } catch {
    // Ignore storage failures in private or restricted browsing contexts.
  }
}

export function resetAppearanceSettings() {
  applyAppearanceSettings(DEFAULT_APPEARANCE_SETTINGS)
  try {
    window.localStorage.removeItem(STORAGE_KEY)
  } catch {
    // Ignore storage failures in private or restricted browsing contexts.
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function clampNumber(value: unknown, min: number, max: number, fallback: number): number {
  if (typeof value !== 'number' || Number.isNaN(value)) return fallback
  return Math.min(max, Math.max(min, Math.round(value)))
}
