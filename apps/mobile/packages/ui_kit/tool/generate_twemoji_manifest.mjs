import { mkdir, readFile, readdir, writeFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const packageDir = path.resolve(scriptDir, '..')
const imageDir = path.join(
  packageDir,
  'assets/twemoji-public/twemoji/72x72',
)
const manifestPath = path.join(packageDir, 'assets/twemoji-manifest.json')
const dartMapPath = path.join(
  packageDir,
  'lib/src/twemoji/twemoji_asset_map.g.dart',
)
const version = '17.0.3'

const files = (await readdir(imageDir))
  .filter((file) => file.endsWith('.png'))
  .filter((file) => {
    const codepoint = Number.parseInt(file.split('-')[0], 16)
    return !(codepoint >= 0xe000 && codepoint <= 0xf8ff)
  })
  .sort()

const entriesBySequence = new Map()
let maximumCodepoints = 0

for (const file of files) {
  const codepoints = file
    .slice(0, -'.png'.length)
    .split('-')
    .map((hex) => Number.parseInt(hex, 16))

  if (codepoints.some((codepoint) => !Number.isInteger(codepoint))) {
    throw new Error(`Invalid Twemoji asset name: ${file}`)
  }
  // The upstream archive also carries one legacy Twitter bird PUA glyph. It
  // is not a Unicode Emoji character and is outside this renderer's scope.
  if (codepoints.some((codepoint) => codepoint >= 0xe000 && codepoint <= 0xf8ff)) {
    continue
  }

  const sequence = String.fromCodePoint(...codepoints)
  const hasJoiner = codepoints.includes(0x200d)
  const canonicalCodepoints = hasJoiner
    ? codepoints
    : codepoints.filter((codepoint) => codepoint !== 0xfe0f)
  const canonicalSequence = String.fromCodePoint(...canonicalCodepoints)
  const existing = entriesBySequence.get(canonicalSequence)

  if (existing && existing.asset !== file) {
    throw new Error(
      `Twemoji assets map to the same sequence: ${existing.asset}, ${file}`,
    )
  }

  entriesBySequence.set(canonicalSequence, { sequence: canonicalSequence, asset: file })
  maximumCodepoints = Math.max(maximumCodepoints, codepoints.length)
  if (sequence.includes('\u200d')) {
    maximumCodepoints = Math.max(maximumCodepoints, Array.from(sequence).length)
  }
}

const entries = [...entriesBySequence.values()].sort((left, right) =>
  left.sequence < right.sequence ? -1 : left.sequence > right.sequence ? 1 : 0,
)
const manifest = `${JSON.stringify({ version, entries }, null, 2)}\n`
const dartMap = [
  '// GENERATED CODE - DO NOT MODIFY BY HAND.',
  '// Regenerate with: node tool/generate_twemoji_manifest.mjs',
  `const String twemojiAssetVersion = '${version}';`,
  `const int twemojiMaximumCodepoints = ${maximumCodepoints + 4};`,
  'const Map<String, String> twemojiAssetBySequence = <String, String>{',
  ...entries.map(({ sequence, asset }) => {
    const entry = `  '${sequence}': '${asset}',`
    return entry.length > 80
      ? `  '${sequence}':\n      '${asset}',`
      : entry
  }),
  '};',
  '',
].join('\n')

const checkOnly = process.argv.includes('--check')

async function writeOrCheck(filePath, expected) {
  let actual
  try {
    actual = await readFile(filePath, 'utf8')
  } catch (error) {
    if (checkOnly || error.code !== 'ENOENT') throw error
  }

  if (actual === expected) return
  if (checkOnly) {
    throw new Error(`${path.relative(packageDir, filePath)} is stale; regenerate it`)
  }
  await writeFile(filePath, expected)
}

await mkdir(path.dirname(dartMapPath), { recursive: true })
await writeOrCheck(manifestPath, manifest)
await writeOrCheck(dartMapPath, dartMap)
console.log(
  `Twemoji ${version}: ${files.length} PNGs, ${entries.length} mapped sequences${checkOnly ? ' (checked)' : ''}`,
)
