import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
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

for (const target of ['manifest', 'Dart map', 'asset list', 'Unicode defaults']) {
  test(`--check rejects stale ${target} without rewriting either generated output`, () => {
    const fixture = mkdtempSync(path.join(tmpdir(), 'twemoji-freshness-'))
    try {
      const tool = path.join(fixture, 'tool/generate_twemoji_manifest.mjs')
      const data = path.join(fixture, 'tool/data/unicode-emoji-17-default-text.txt')
      const images = path.join(fixture, 'assets/twemoji-public/twemoji/72x72')
      const manifest = path.join(fixture, 'assets/twemoji-manifest.json')
      const dart = path.join(fixture, 'lib/src/twemoji/twemoji_asset_map.g.dart')
      mkdirSync(path.dirname(data), { recursive: true })
      mkdirSync(images, { recursive: true })
      copyFileSync(path.join(toolDir, 'generate_twemoji_manifest.mjs'), tool)
      writeFileSync(data, '00a9\n')
      // The manifest generator consumes inventory names, not PNG pixel contents.
      writeFileSync(path.join(images, '1f600.png'), '')
      execFileSync(process.execPath, [tool], { stdio: 'pipe' })
      if (target === 'manifest') writeFileSync(manifest, '{}\n')
      if (target === 'Dart map') writeFileSync(dart, '// stale\n')
      if (target === 'asset list') writeFileSync(path.join(images, '1f601.png'), '')
      if (target === 'Unicode defaults') writeFileSync(data, '00a9\n00ae\n')
      const before = [readFileSync(manifest, 'utf8'), readFileSync(dart, 'utf8')]
      assert.throws(
        () => execFileSync(process.execPath, [tool, '--check'], { stdio: 'pipe' }),
        (error) => error.status !== 0 && /is stale; regenerate it/.test(error.stderr.toString()),
      )
      assert.deepEqual([readFileSync(manifest, 'utf8'), readFileSync(dart, 'utf8')], before)
    } finally {
      rmSync(fixture, { recursive: true, force: true })
    }
  })
}
