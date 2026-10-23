import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'
import {
  buildTwemojiManifest,
  parseDefaultTextPresentationCodepoints,
} from './generate_twemoji_manifest.mjs'

const toolDir = path.dirname(fileURLToPath(import.meta.url))
const packageDir = path.resolve(toolDir, '..')
const unicodeDataPath = path.join(toolDir, 'data/unicode-emoji-17-default-text.txt')
const flagAssetPath = path.join(
  packageDir,
  'assets/twemoji-public/twemoji/72x72/1f1f9-1f1fc.png',
)

test('Unicode 17 text-presentation defaults include copyright, registered, and trademark', () => {
  const codepoints = parseDefaultTextPresentationCodepoints(
    readFileSync(unicodeDataPath, 'utf8'),
  )
  assert.ok(codepoints.includes(0x00a9))
  assert.ok(codepoints.includes(0x00ae))
  assert.ok(codepoints.includes(0x2122))
  assert.ok(codepoints.includes(0x2764))
  assert.ok(!codepoints.includes(0x1f600))
})

test('manifest generation omits the excluded Taiwan flag but retains other flags', () => {
  const { entries, excludedSequences } = buildTwemojiManifest([
    '1f1f9-1f1fc.png',
    '1f1fa-1f1f8.png',
    '1f600.png',
    'e000.png',
    'not-an-asset.txt',
  ])
  const sequences = new Set(entries.map(({ sequence }) => sequence))
  assert.ok(!sequences.has('🇹🇼'))
  assert.deepEqual(excludedSequences, ['🇹🇼'])
  assert.ok(sequences.has('🇺🇸'))
  assert.ok(sequences.has('😀'))
})

test('checked-in manifest and Dart map are reproducible and the flag file is absent', () => {
  execFileSync(
    process.execPath,
    [path.join(toolDir, 'generate_twemoji_manifest.mjs'), '--check'],
    { cwd: packageDir, stdio: 'pipe' },
  )
  assert.equal(existsSync(flagAssetPath), false)

  const manifest = JSON.parse(
    readFileSync(path.join(packageDir, 'assets/twemoji-manifest.json'), 'utf8'),
  )
  assert.equal(manifest.entries.some(({ sequence }) => sequence === '🇹🇼'), false)
  assert.deepEqual(manifest.excludedSequences, ['🇹🇼'])
  assert.ok(manifest.entries.some(({ sequence }) => sequence === '🇺🇸'))
  assert.ok(manifest.entries.some(({ sequence }) => sequence === '😀'))
})
