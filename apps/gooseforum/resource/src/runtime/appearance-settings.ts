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
  if (typeof legacy.fontSize === 'number') {
    const size = clampNumber(legacy.fontSize, FONT_SIZE_MIN, FONT_SIZE_MAX, 16)
    return {
      ui: normalizeZone({ size, familyPreset: legacy.fontFamilyPreset, customFontFamily: legacy.customFontFamily }, 'ui'),
      body: normalizeZone({ size, familyPreset: legacy.fontFamilyPreset, customFontFamily: legacy.customFontFamily }, 'body'),
      code: normalizeZone({}, 'code'),
    }
  }
  return {
    ui: normalizeZone({}, 'ui'),
    body: normalizeZone({}, 'body'),
    code: normalizeZone({}, 'code'),
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
  if (zone.familyPreset === 'custom') return quoteFontFamily(zone.customFamily) || FONT_STACKS.system
  return FONT_STACKS[zone.familyPreset]
}

/**
 * 把用户字体名包成安全的 CSS font-family 值。
 * 字体名含空格（如 "Noto Serif SC"）时必须加引号，否则会被拆成多个族名；
 * 同时剔除可能破坏 CSS 结构的引号字符。
 */
export function quoteFontFamily(name: string): string {
  const trimmed = name.trim()
  if (!trimmed) return ''
  if ((trimmed.startsWith('"') && trimmed.endsWith('"')) || (trimmed.startsWith("'") && trimmed.endsWith("'"))) {
    return trimmed
  }
  return `"${trimmed.replace(/["']/g, '')}"`
}

export interface LocalFontInfo {
  family: string
  fullName: string
}

export type LocalFontsResult =
  | { status: 'ok'; fonts: LocalFontInfo[] }
  | { status: 'unsupported' }
  | { status: 'error' }

/** 枚举本机字体（Local Font Access API，Chrome/Edge 103+）。 */
export async function loadLocalFonts(): Promise<LocalFontsResult> {
  if (typeof window === 'undefined' || typeof window.queryLocalFonts !== 'function') {
    return { status: 'unsupported' }
  }
  try {
    const fonts = await window.queryLocalFonts()
    const seen = new Set<string>()
    const list: LocalFontInfo[] = []
    for (const font of fonts) {
      const family = font.family?.trim()
      if (!family || seen.has(family)) continue
      seen.add(family)
      list.push({ family, fullName: font.fullName?.trim() || family })
    }
    list.sort((a, b) => a.family.localeCompare(b.family, undefined, { sensitivity: 'base' }))
    return { status: 'ok', fonts: list }
  } catch {
    // 用户拒绝授权或 API 异常。
    return { status: 'error' }
  }
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

function isZoneCustomized(zone: FontZone, zones: Record<FontZone, ZoneFont>): boolean {
  const zf = zones[zone]
  return zf.size !== DEFAULT_SIZE[zone] || zf.familyPreset !== DEFAULT_FAMILY[zone]
}

function applyFontZones(zones: Record<FontZone, ZoneFont>) {
  const root = document.documentElement
  // UI zone: root font-size + body font-family
  if (zones.ui.size !== DEFAULT_SIZE.ui) root.style.fontSize = `${zones.ui.size}px`
  else root.style.removeProperty('font-size')
  setFontVar('--gf-font-family-ui', zones.ui)

  applyZoneFont('body', zones, root)
  applyZoneFont('code', zones, root)
}

function applyZoneFont(zone: 'body' | 'code', zones: Record<FontZone, ZoneFont>, root: HTMLElement) {
  const zf = zones[zone]
  if (isZoneCustomized(zone, zones)) {
    root.setAttribute(`data-gf-font-${zone}`, '')
    root.style.setProperty(`--gf-font-size-${zone}`, `${zf.size}px`)
    if (zf.familyPreset !== DEFAULT_FAMILY[zone]) setFontVar(`--gf-font-family-${zone}`, zf)
    else root.style.removeProperty(`--gf-font-family-${zone}`)
  } else {
    root.removeAttribute(`data-gf-font-${zone}`)
    root.style.removeProperty(`--gf-font-size-${zone}`)
    root.style.removeProperty(`--gf-font-family-${zone}`)
  }
}

function setFontVar(name: string, zone: ZoneFont) {
  const family = resolveFontFamily(zone)
  if (family) document.documentElement.style.setProperty(name, family)
  else document.documentElement.style.removeProperty(name)
}

function removeFontOverrides() {
  const root = document.documentElement
  root.removeAttribute('data-gf-font-body')
  root.removeAttribute('data-gf-font-code')
  root.style.removeProperty('font-size')
  root.style.removeProperty('--gf-font-family-ui')
  root.style.removeProperty('--gf-font-family-body')
  root.style.removeProperty('--gf-font-family-code')
  root.style.removeProperty('--gf-font-size-body')
  root.style.removeProperty('--gf-font-size-code')
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
  const normalized = normalizeAppearanceSettings(settings)
  applyAppearanceSettings(normalized)
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(normalized))
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
