import type { DecorationStyle } from '@gooseforum/client'

export function isSafeDecorationAssetUrl(value: string | undefined): value is string {
  return !!value && /^\/static\/decorations\/[A-Za-z0-9][A-Za-z0-9._-]{0,120}\.(?:png|webp)$/.test(value)
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

export function hasNicknameContrast(colors: string[], background: readonly number[]): boolean {
  if (!colors.length || colors.some(color => !/^#[0-9a-f]{6}$/i.test(color))) return false
  const rgb = (hex: string) => [1, 3, 5].map(offset => parseInt(hex.slice(offset, offset + 2), 16))
  const luminance = (values: readonly number[]) => values.map(value => { const c = value / 255; return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4 }).reduce((sum, value, index) => sum + value * [0.2126, 0.7152, 0.0722][index], 0)
  const bg = luminance(background)
  const safe = (values: number[]) => { const fg = luminance(values); return (Math.max(fg, bg) + 0.05) / (Math.min(fg, bg) + 0.05) >= 4.5 }
  const stops = colors.map(rgb)
  if (stops.some(stop => !safe(stop))) return false
  return stops.slice(1).every((stop, index) => Array.from({ length: 15 }, (_, step) => (step + 1) / 16).every(t => safe(stop.map((value, channel) => stops[index][channel] * (1 - t) + value * t))))
}
