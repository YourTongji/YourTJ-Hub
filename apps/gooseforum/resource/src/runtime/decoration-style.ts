// Pure decoration presentation helpers shared by the site and admin entrypoints.
import type { DecorationStyle } from '@gooseforum/client'

export function isSafeDecorationAssetUrl(value: string | undefined): value is string {
  return !!value && /^\/static\/decorations\/[A-Za-z0-9][A-Za-z0-9._-]{0,120}\.(?:png|webp|gif|svg)$/.test(value)
}

export function decorationAssetScale(style?: DecorationStyle | null): number {
  const value = style?.assetScale
  return typeof value === 'number' && Number.isFinite(value) && value >= 0.5 && value <= 2 ? value : 1.25
}

export function decorationVariables(style?: DecorationStyle | null) {
  if (!style?.isEnabled) return {}
  const valid = (colors: string[]) => Array.isArray(colors) && colors.length >= 1 && colors.length <= 4 && colors.every(color => /^#[0-9a-f]{6}$/i.test(color))
  if (!valid(style.lightColors) || !valid(style.darkColors)) return {}
  const gradient = (colors: string[]) => `linear-gradient(120deg, ${colors.length === 1 ? [colors[0], colors[0]].join(', ') : colors.join(', ')})`
  return { '--decoration-light': gradient(style.lightColors), '--decoration-dark': gradient(style.darkColors) }
}

// WCAG large-text floor; the Go authority (NicknameContrastMin) and Flutter mirror this value.
export const NICKNAME_CONTRAST_MIN = 3

const luminance = (values: readonly number[]) => values.map(value => { const c = value / 255; return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4 }).reduce((sum, value, index) => sum + value * [0.2126, 0.7152, 0.0722][index], 0)

// Lowest contrast across every gradient stop and every background; invalid input floors at 0.
export function nicknameContrastFloor(colors: string[], backgrounds: readonly (readonly number[])[]): number {
  if (!colors.length || !backgrounds.length || colors.some(color => !/^#[0-9a-f]{6}$/i.test(color))) return 0
  const rgb = (hex: string) => [1, 3, 5].map(offset => parseInt(hex.slice(offset, offset + 2), 16))
  const stops = colors.map(color => luminance(rgb(color)))
  return Math.min(...backgrounds.flatMap(background => { const bg = luminance(background); return stops.map(fg => (Math.max(fg, bg) + 0.05) / (Math.min(fg, bg) + 0.05)) }))
}

export function hasNicknameContrast(colors: string[], background: readonly number[]): boolean {
  return nicknameContrastFloor(colors, [background]) >= NICKNAME_CONTRAST_MIN
}
