export type FontZone = 'ui' | 'body' | 'code'
export type FontFamilyPreset = 'system' | 'serif' | 'kai' | 'hei' | 'mono' | 'custom'

export interface ZoneFont {
  size: number
  familyPreset: FontFamilyPreset
  customFamily: string
}

export interface AppearanceSettings {
  zones: Record<FontZone, ZoneFont>
  clickAnimation: boolean
  customCss: string
}

const STORAGE_KEY = 'goose-appearance-settings'
export const FONT_SIZE_MIN = 12
export const FONT_SIZE_MAX = 24
export const MAX_CUSTOM_FONT_LENGTH = 200
export const MAX_CUSTOM_CSS_LENGTH = 256 * 1024
export const CUSTOM_CSS_STYLE_ID = 'goose-custom-css'

const DEFAULT_SIZE: Record<FontZone, number> = { ui: 16, body: 16, code: 14 }
const DEFAULT_FAMILY: Record<FontZone, FontFamilyPreset> = { ui: 'system', body: 'system', code: 'mono' }

export const FONT_ZONES: readonly FontZone[] = ['ui', 'body', 'code']

export const FONT_FAMILY_PRESETS: readonly FontFamilyPreset[] = [
  'system',
  'serif',
  'kai',
  'hei',
  'mono',
  'custom',
]

export const FONT_STACKS: Record<Exclude<FontFamilyPreset, 'custom'>, string> = {
  system: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "PingFang SC", "Hiragino Sans GB", "Segoe UI", "Microsoft YaHei", "Noto Sans CJK SC", Arial, sans-serif',
  serif: '"Songti SC", "SimSun", serif',
  kai: '"Kaiti SC", "KaiTi", "STKaiti", serif',
  hei: '"Heiti SC", "SimHei", "Noto Sans CJK SC", sans-serif',
  mono: 'ui-monospace, SFMono-Regular, Menlo, Consolas, "Courier New", monospace',
}

export const DEFAULT_APPEARANCE_SETTINGS: AppearanceSettings = Object.freeze({
  zones: Object.freeze({
    ui: Object.freeze({ size: 16, familyPreset: 'system', customFamily: '' }),
    body: Object.freeze({ size: 16, familyPreset: 'system', customFamily: '' }),
    code: Object.freeze({ size: 14, familyPreset: 'mono', customFamily: '' }),
  }),
  clickAnimation: false,
  customCss: '',
})

export function normalizeAppearanceSettings(raw: unknown): AppearanceSettings {
  const source = isRecord(raw) ? raw : {}
  const zones = normalizeZones(source.zones, source)
  const clickAnimation = source.clickAnimation === true
  const customCss = typeof source.customCss === 'string'
    ? source.customCss.slice(0, MAX_CUSTOM_CSS_LENGTH)
    : ''
  return { zones, clickAnimation, customCss }
}

function normalizeZones(rawZones: unknown, legacy: Record<string, unknown>): Record<FontZone, ZoneFont> {
  if (isRecord(rawZones)) {
    return {
      ui: normalizeZone(rawZones.ui, 'ui'),
      body: normalizeZone(rawZones.body, 'body'),
      code: normalizeZone(rawZones.code, 'code'),
    }
  }
  if (typeof legacy.fontSize !== 'number') {
    return {
      ui: normalizeZone(undefined, 'ui'),
      body: normalizeZone(undefined, 'body'),
      code: normalizeZone(undefined, 'code'),
    }
  }
  const legacySize = clampNumber(legacy.fontSize, FONT_SIZE_MIN, FONT_SIZE_MAX, 16)
  const legacyFamily = legacy.fontFamilyPreset
  const legacyCustom = legacy.customFontFamily
  return {
    ui: normalizeZone({ size: legacySize, familyPreset: legacyFamily, customFontFamily: legacyCustom }, 'ui'),
    body: normalizeZone({ size: legacySize, familyPreset: legacyFamily, customFontFamily: legacyCustom }, 'body'),
    code: normalizeZone({ size: legacySize, familyPreset: legacyFamily, customFontFamily: legacyCustom }, 'code'),
  }
}

function normalizeZone(raw: unknown, zone: FontZone): ZoneFont {
  const source = isRecord(raw) ? raw : {}
  const customFamily = (
    typeof source.customFamily === 'string'
      ? source.customFamily
      : typeof source.customFontFamily === 'string'
        ? source.customFontFamily
        : ''
  ).slice(0, MAX_CUSTOM_FONT_LENGTH)
  const size = clampNumber(source.size, FONT_SIZE_MIN, FONT_SIZE_MAX, DEFAULT_SIZE[zone])
  let familyPreset = FONT_FAMILY_PRESETS.includes(source.familyPreset as FontFamilyPreset)
    ? (source.familyPreset as FontFamilyPreset)
    : DEFAULT_FAMILY[zone]
  if (familyPreset === 'custom' && customFamily.trim() === '') familyPreset = 'system'
  return { size, familyPreset, customFamily }
}

export function resolveFontFamily(zone: ZoneFont): string {
  if (zone.familyPreset === 'custom') return zone.customFamily.trim() || FONT_STACKS.system
  return FONT_STACKS[zone.familyPreset]
}

export function isFontPristine(settings: AppearanceSettings): boolean {
  return FONT_ZONES.every(zone => {
    const zf = settings.zones[zone]
    return zf.size === DEFAULT_SIZE[zone] && zf.familyPreset === DEFAULT_FAMILY[zone]
  })
}

let clickAnimationEnabled = false
export function isClickAnimationEnabled(): boolean {
  return clickAnimationEnabled
}

export function loadAppearanceSettings(): AppearanceSettings {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (raw === null) return copySettings(DEFAULT_APPEARANCE_SETTINGS)
    return normalizeAppearanceSettings(JSON.parse(raw))
  } catch {
    return copySettings(DEFAULT_APPEARANCE_SETTINGS)
  }
}

export function applyAppearanceSettings(settings: AppearanceSettings) {
  clickAnimationEnabled = settings.clickAnimation
  applyCustomCss(settings.customCss)
  if (isFontPristine(settings)) {
    removeFontOverrides()
    return
  }
  applyFontZones(settings.zones)
}

function applyFontZones(zones: Record<FontZone, ZoneFont>) {
  document.documentElement.style.fontSize = `${zones.ui.size}px`
  setFontVar('--gf-font-family-ui', zones.ui)
  setFontVar('--gf-font-family-body', zones.body)
  setFontVar('--gf-font-family-code', zones.code)
  document.documentElement.style.setProperty('--gf-font-size-body', `${zones.body.size}px`)
  document.documentElement.style.setProperty('--gf-font-size-code', `${zones.code.size}px`)
}

function setFontVar(name: string, zone: ZoneFont) {
  const family = resolveFontFamily(zone)
  if (family) document.documentElement.style.setProperty(name, family)
  else document.documentElement.style.removeProperty(name)
}

function removeFontOverrides() {
  const root = document.documentElement.style
  root.removeProperty('font-size')
  root.removeProperty('--gf-font-family-ui')
  root.removeProperty('--gf-font-family-body')
  root.removeProperty('--gf-font-family-code')
  root.removeProperty('--gf-font-size-body')
  root.removeProperty('--gf-font-size-code')
}

function applyCustomCss(css: string) {
  let el = document.getElementById(CUSTOM_CSS_STYLE_ID) as HTMLStyleElement | null
  if (!css) {
    el?.remove()
    return
  }
  if (!el) {
    el = document.createElement('style')
    el.id = CUSTOM_CSS_STYLE_ID
    document.head.appendChild(el)
  }
  el.textContent = css
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

function copySettings(settings: AppearanceSettings): AppearanceSettings {
  return {
    zones: {
      ui: { ...settings.zones.ui },
      body: { ...settings.zones.body },
      code: { ...settings.zones.code },
    },
    clickAnimation: settings.clickAnimation,
    customCss: settings.customCss,
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function clampNumber(value: unknown, min: number, max: number, fallback: number): number {
  if (typeof value !== 'number' || Number.isNaN(value)) return fallback
  return Math.min(max, Math.max(min, Math.round(value)))
}
