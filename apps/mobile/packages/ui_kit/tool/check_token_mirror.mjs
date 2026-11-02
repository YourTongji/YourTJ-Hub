import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// These Web-only course slots and typography sizes are not part of the mobile
// palette/geometry mirror. Mobile component roles remain independently owned.
const webOnlyKeys = new Set([
  ...Array.from({ length: 8 }, (_, i) => [`color-course-${i + 1}`, `color-course-${i + 1}-content`]).flat(),
  'text-xs', 'text-sm', 'text-base', 'text-lg',
])
const decimal = String.raw`(?:\d+(?:\.\d+)?|\.\d+)`
const oklchPattern = new RegExp(`^oklch\\((${decimal})(%?)\\s+(${decimal})\\s+(-?${decimal})\\)$`)
const numericPattern = new RegExp(`^(${decimal})(rem|px)?$`)

export function normalizeTokenValue(key, value) {
  if (key.startsWith('color-')) {
    if (/^#[\da-f]{6}$/i.test(value)) return value.toLowerCase()
    const match = oklchPattern.exec(value)
    if (match) {
      const lightness = Number(match[1]) / (match[2] ? 100 : 1)
      const chroma = Number(match[3])
      const hue = Number(match[4]) * Math.PI / 180
      if (lightness >= 0 && lightness <= 1 && Number.isFinite(chroma) && Number.isFinite(hue)) {
        const a = chroma * Math.cos(hue)
        const b = chroma * Math.sin(hue)
        // Oklab inverse matrices: Björn Ottosson, public-domain implementation
        // https://bottosson.github.io/posts/oklab/#converting-from-linear-srgb-to-oklab
        const l = (lightness + 0.3963377774 * a + 0.2158037573 * b) ** 3
        const m = (lightness - 0.1055613458 * a - 0.0638541728 * b) ** 3
        const s = (lightness - 0.0894841775 * a - 1.2914855480 * b) ** 3
        const linear = [
          4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s,
          -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s,
          -0.0041960863 * l - 0.7034186147 * m + 1.7076147010 * s,
        ]
        // Match the existing 8-bit sRGB mirror: clip channels, then round. This
        // is not a promise of browser/display-P3 gamut-mapping equivalence.
        // Transfer function: https://www.w3.org/TR/css-color-4/#color-conversion-code
        return `#${linear.map((channel) => {
          const srgb = channel <= 0.0031308 ? 12.92 * channel : 1.055 * channel ** (1 / 2.4) - 0.055
          return Math.round(Math.max(0, Math.min(1, srgb)) * 255).toString(16).padStart(2, '0')
        }).join('')}`
      }
    }
  } else {
    const match = numericPattern.exec(value)
    if (match && (key === 'depth' ? !match[2] : (match[2] || Number(match[1]) === 0))) {
      // The shared token mirror uses the Web design baseline, 1rem = 16px;
      // Flutter's runtime text scaling and mobile component roles are separate.
      const number = Number(match[1]) * (match[2] === 'rem' ? 16 : 1)
      if (Number.isFinite(number)) return number
    }
  }
  throw new Error(`Unsupported token value ${key}: ${value}`)
}

function readWebThemes(css) {
  const themes = {}
  // This intentionally accepts the repository's flat token declaration format,
  // not arbitrary CSS. Unsupported/missing theme declarations fail closed.
  const source = css.replace(/\/\*[\s\S]*?\*\//g, '')
  const flatBlock = /([^{}]+)\{([^{}]*)\}/g
  if (source.replace(flatBlock, '').trim()) throw new Error('Unsupported nested or malformed token CSS')
  for (const [, selectors, body] of source.matchAll(flatBlock)) {
    const names = selectors.split(',').map((selector) => selector.trim())
    const theme = names.includes('[data-theme="gf-light"]') ? 'light'
      : names.includes('[data-theme="gf-dark"]') ? 'dark' : null
    if (!theme) {
      if (/(?:^|;)\s*--gf-/.test(body)) {
        throw new Error(`Unsupported token theme selector: ${selectors.trim()}`)
      }
      continue
    }
    if (themes[theme]) throw new Error(`Duplicate ${theme} theme block`)
    if (theme === 'dark' && !themes.light) throw new Error('Token theme order must be root/light before dark')
    if (theme === 'light' ? names.length !== 2 || !names.includes(':root') : names.length !== 1) {
      throw new Error(`Unsupported ${theme} theme selector: ${selectors.trim()}`)
    }
    const values = {}
    for (const declaration of body.split(';').map((line) => line.trim()).filter(Boolean)) {
      if (!declaration.startsWith('--gf-')) continue
      const match = /^--gf-([\w-]+)\s*:\s*(.+)$/.exec(declaration)
      if (!match) throw new Error(`Unsupported token declaration: ${declaration}`)
      const [, key, value] = match
      if (Object.hasOwn(values, key)) throw new Error(`Duplicate ${theme}/${key}`)
      values[key] = value
    }
    themes[theme] = values
  }
  if (!themes.light || !themes.dark) throw new Error('Both light and dark token theme blocks are required')
  return { light: themes.light, dark: { ...themes.light, ...themes.dark } }
}

export function checkTokenMirror(css, mirror) {
  if (!mirror || Object.keys(mirror).sort().join(',') !== 'dark,light') {
    throw new Error('Token mirror must contain exactly light and dark themes')
  }
  const themes = readWebThemes(css)
  const errors = []
  for (const [theme, values] of Object.entries(themes)) {
    const expected = Object.fromEntries(Object.entries(values)
      .filter(([key]) => !webOnlyKeys.has(key))
      .map(([key, value]) => [key, normalizeTokenValue(key, value)]))
    for (const key of new Set([...Object.keys(expected), ...Object.keys(mirror[theme] ?? {})])) {
      if (!Object.hasOwn(expected, key) || !Object.hasOwn(mirror[theme] ?? {}, key) || expected[key] !== mirror[theme][key]) {
        errors.push(`${theme}/${key}: web=${JSON.stringify(expected[key])}, mobile=${JSON.stringify(mirror[theme]?.[key])}`)
      }
    }
  }
  if (errors.length) throw new Error(`Token mirror drift:\n${errors.join('\n')}\nUpdate tokens.json and the corresponding Dart constants together.`)
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const css = readFileSync(new URL('../../../../gooseforum/resource/src/styles/tokens.css', import.meta.url), 'utf8')
  const mirror = JSON.parse(readFileSync(new URL('../lib/src/theme/tokens.json', import.meta.url), 'utf8'))
  checkTokenMirror(css, mirror)
  console.log('Token mirror: light and dark values match')
}
