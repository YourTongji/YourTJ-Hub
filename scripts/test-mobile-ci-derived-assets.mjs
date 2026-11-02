import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import test from 'node:test'

const root = new URL('../', import.meta.url)

test('mobile CI runs dependency-free derived-asset gates with the pinned Node version', () => {
  const workflow = readFileSync(new URL('.github/workflows/ci-mobile.yml', root), 'utf8')
  assert.match(workflow, /node-version-file: \.node-version/)
  assert.match(workflow, /node --test apps\/mobile\/packages\/ui_kit\/tool\/\*\.test\.mjs/)
})

test('frontend tests include both Twemoji and token mirror gates', () => {
  const pkg = JSON.parse(readFileSync(new URL('apps/gooseforum/resource/package.json', root), 'utf8'))
  assert.match(pkg.scripts.test, /node --test \.\.\/\.\.\/mobile\/packages\/ui_kit\/tool\/\*\.test\.mjs/)
})

for (const path of [
  'apps/gooseforum/resource/src/styles/tokens.css',
  'apps/mobile/packages/ui_kit/lib/src/theme/tokens.json',
  'apps/mobile/packages/ui_kit/assets/twemoji-public/twemoji/72x72/1f600.png',
  'apps/mobile/packages/ui_kit/assets/twemoji-manifest.json',
  'apps/mobile/packages/ui_kit/lib/src/twemoji/twemoji_asset_map.g.dart',
  'apps/mobile/packages/ui_kit/tool/data/unicode-emoji-17-default-text.txt',
  'apps/mobile/packages/ui_kit/tool/check_token_mirror.mjs',
]) {
  test(`mobile gates cannot be bypassed by changing only ${path}`, () => {
    const run = spawnSync('python3', ['-c', 'import sys,json; sys.path.insert(0,"scripts/ci"); from select_inputs import select; print(json.dumps(select([sys.argv[1]])))', path], { cwd: root, encoding: 'utf8' })
    assert.equal(run.status, 0, run.stderr)
    assert.equal(JSON.parse(run.stdout).mobile.run, true)
  })
}
