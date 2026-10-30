import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { tmpdir } from 'node:os'
import { fileURLToPath } from 'node:url'
import { checkTokenMirror, normalizeTokenValue } from './check_token_mirror.mjs'

const cssPath = new URL('../../../../gooseforum/resource/src/styles/tokens.css', import.meta.url)
const tokenPath = new URL('../lib/src/theme/tokens.json', import.meta.url)
const css = readFileSync(cssPath, 'utf8')
const tokens = JSON.parse(readFileSync(tokenPath, 'utf8'))

// Known sRGB references, not values calculated by the implementation under test.
test('normalizes the supported CSS values into the existing mobile representation', () => {
  for (const [value, expected] of [
    ['#ABCDEF', '#abcdef'], ['oklch(100% 0 0)', '#ffffff'],
    ['oklch(0 0 0)', '#000000'], ['oklch(50% 0 0)', '#636363'],
    ['oklch(54.6% 0.215 262.88)', '#2563eb'],
    ['oklch(62.7% 0.194 149.214)', '#00a63e'],
  ]) assert.equal(normalizeTokenValue('color-primary', value), expected)
  assert.equal(normalizeTokenValue('radius-box', '0.5rem'), 8)
  assert.equal(normalizeTokenValue('border', '1px'), 1)
  assert.equal(normalizeTokenValue('depth', '1'), 1)
})

test('unsupported or invalid values fail rather than silently skipping equality', () => {
  for (const value of ['red', '#fff', 'var(--other)', 'oklch(50% 0.1 0 / .5)', 'oklch(101% 0 0)', 'oklch(50% -1 0)']) {
    assert.throws(() => normalizeTokenValue('color-primary', value), /Unsupported/)
  }
  for (const value of ['calc(1rem)', '1em', 'NaN', '1px junk', '8']) {
    assert.throws(() => normalizeTokenValue('radius-box', value), /Unsupported/)
  }
  assert.throws(() => normalizeTokenValue('depth', '1px'), /Unsupported/)
})

test('checked-in mirror equals both resolved web themes and CLI works outside the repository', () => {
  checkTokenMirror(css, tokens)
  const output = execFileSync(process.execPath, [fileURLToPath(new URL('check_token_mirror.mjs', import.meta.url))], { cwd: tmpdir(), encoding: 'utf8' })
  assert.match(output, /Token mirror: light and dark values match/)
})

for (const theme of ['light', 'dark']) {
  for (const key of ['color-primary', 'color-tooltip-bg', 'radius-box', 'depth']) {
    test(`detects a changed mobile value: ${theme}/${key}`, () => {
      const changed = structuredClone(tokens)
      changed[theme][key] = key.startsWith('color-') ? '#ffffff' : 9
      assert.throws(() => checkTokenMirror(css, changed), new RegExp(`${theme}/${key}`))
    })
  }
}

test('detects a web-only change without changing the mobile mirror', () => {
  assert.throws(() => checkTokenMirror(css.replace('oklch(54.6% 0.215 262.88)', '#ffffff'), tokens), /light\/color-primary/)
})

test('dark inherits root geometry and static colors, but explicit overrides win', () => {
  assert.throws(() => checkTokenMirror(css.replace('--gf-radius-box: 0.5rem', '--gf-radius-box: 1rem'), tokens), /light\/radius-box[\s\S]*dark\/radius-box/)
  assert.throws(() => checkTokenMirror(css.replace('[data-theme="gf-dark"] {', '[data-theme="gf-dark"] {\n --gf-radius-box: 1rem;'), tokens), /dark\/radius-box/)
})

test('missing and extra keys or themes fail; deleting a mirror key cannot bypass checks', () => {
  const missing = structuredClone(tokens)
  delete missing.dark['color-primary']
  assert.throws(() => checkTokenMirror(css, missing), /dark\/color-primary/)
  assert.throws(() => checkTokenMirror(css.replace('--gf-depth: 1;', ''), tokens), /depth/)
  assert.throws(() => checkTokenMirror(css.replace('--gf-depth: 1;', '--gf-depth: 1; --gf-new-token: 2px;'), tokens), /new-token/)
  const extra = structuredClone(tokens)
  extra.light['extra-token'] = 1
  assert.throws(() => checkTokenMirror(css, extra), /extra-token/)
  assert.throws(() => checkTokenMirror(css, { light: tokens.light }), /themes/)
  assert.throws(() => checkTokenMirror(css, { ...tokens, other: {} }), /themes/)
})

test('web-only course and typography tokens do not become mobile component roles', () => {
  checkTokenMirror(css.replace('--gf-text-xs: 0.6875rem', '--gf-text-xs: 1rem').replace('--gf-color-course-1: oklch(45% 0.19 268)', '--gf-color-course-1: #123456'), tokens)
})

test('missing theme blocks and duplicate declarations fail closed', () => {
  assert.throws(() => checkTokenMirror(css.replace('[data-theme="gf-dark"]', '[data-theme="other"]'), tokens), /theme/)
  assert.throws(() => checkTokenMirror(css.replace('--gf-depth: 1;', '--gf-depth: 1; --gf-depth: 2;'), tokens), /Duplicate/)
})

test('selector broadening and conditional theme rules cannot masquerade as global parity', () => {
  assert.throws(() => checkTokenMirror(css.replace('[data-theme="gf-dark"] {', ':root,\n[data-theme="gf-dark"] {'), tokens), /selector/)
  assert.throws(() => checkTokenMirror(css.replace('[data-theme="gf-light"] {', '[data-theme="gf-light"], body {'), tokens), /selector/)
  const nested = css.replace('[data-theme="gf-dark"] {', '@media print {\n[data-theme="gf-dark"] {').replace('\n@theme inline', '\n}\n@theme inline')
  assert.throws(() => checkTokenMirror(nested, tokens), /nested|Unsupported/)
})

test('dark overrides must follow root defaults in source order', () => {
  const blocks = css.match(/[^{}]+\{[^{}]*\}/g)
  assert.throws(() => checkTokenMirror([blocks[1], blocks[0], ...blocks.slice(2)].join('\n'), tokens), /order/)
})

test('Tailwind alias block cannot redefine the source token values', () => {
  assert.throws(() => checkTokenMirror(css.replace('@theme inline {', '@theme inline {\n --gf-depth: 2;'), tokens), /selector/)
})
