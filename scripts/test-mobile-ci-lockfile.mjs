import assert from 'node:assert/strict'
import { mkdtempSync, mkdirSync, readFileSync, writeFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { spawnSync } from 'node:child_process'
import test from 'node:test'

const root = new URL('../', import.meta.url)
const read = path => readFileSync(new URL(path, root), 'utf8')

test('hosted Flutter locks use the same canonical registry as CI', () => {
  const blocks = read('apps/mobile/pubspec.lock').split(/(?=^  \w+:\n)/m)
  const hosted = blocks.filter(block => /^    source: hosted$/m.test(block))
  assert.ok(hosted.length > 0)
  for (const block of hosted) assert.match(block, /^      url: "https:\/\/pub.dev"$/m,
    'Registry rewrites make pub resolve new versions instead of honoring the checked-in lock')
})

function runScript(step) {
  const block = step.match(/^        run: \|\n((?:          .*\n|\n)+)/m)
  assert.ok(block, 'Bootstrap needs an executable, fail-closed block')
  return block[1].replace(/^          /gm, '')
}
function git(cwd, ...args) {
  const r = spawnSync('git', args, { cwd, encoding: 'utf8' })
  assert.equal(r.status, 0, r.stderr)
}
for (const workflow of ['ci-mobile.yml', 'ci-mobile-native.yml']) {
  const source = read('.github/workflows/' + workflow)
  const bootstraps = source.split(/^      - /m).slice(1).filter(step => step.startsWith('name: Bootstrap\n'))
  assert.equal(bootstraps.length, workflow === 'ci-mobile.yml' ? 2 : 1)
  for (const [index, step] of bootstraps.entries()) {
    test(`${workflow} bootstrap ${index} preserves locked versions and rejects resolver drift`, t => {
      const script = runScript(step)
      assert.match(script, /^flutter pub get --enforce-lockfile$/m)
      assert.match(script, /^git diff --exit-code -- ':\(top\)apps\/mobile\/pubspec.lock'$/m)
      const directory = mkdtempSync(join(tmpdir(), 'mobile-lock-bootstrap-'))
      t.after(() => rmSync(directory, { recursive: true, force: true }))
      const working = join(directory, step.match(/^        working-directory: (.+)$/m)[1])
      const bin = join(directory, 'bin')
      const lock = join(directory, 'apps/mobile/pubspec.lock')
      mkdirSync(working, { recursive: true }); mkdirSync(bin)
      writeFileSync(lock, 'packages:\n  pinned:\n    version: "1.0.0"\n')
      git(directory, 'init', '--quiet'); git(directory, 'add', '.')
      git(directory, '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', 'commit', '--quiet', '-m', 'fixture')
      writeFileSync(join(bin, 'flutter'), `#!/usr/bin/env bash
set -euo pipefail
test "$*" = 'pub get --enforce-lockfile'
if [ "$LOCK_MODE" = fail ]; then exit 23; fi
if [ "$LOCK_MODE" = flutter ]; then printf 'resolver changed dependency version\n' > "$TASK_LOCKFILE"; fi
`, { mode: 0o755 })
      writeFileSync(join(bin, 'dart'), `#!/usr/bin/env bash
set -euo pipefail
test "$*" = 'run melos bootstrap'
if [ "$LOCK_MODE" = dart ]; then printf 'resolver changed package source\n' > "$TASK_LOCKFILE"; fi
`, { mode: 0o755 })
      for (const mode of ['fresh', 'fail', 'flutter', ...(workflow === 'ci-mobile.yml' ? ['dart'] : [])]) {
        git(directory, 'restore', '--', 'apps/mobile/pubspec.lock')
        const r = spawnSync('bash', ['-euo', 'pipefail', '-c', script], { cwd: working,
          env: { ...process.env, PATH: `${bin}:${process.env.PATH}`, TASK_LOCKFILE: lock, LOCK_MODE: mode }, encoding: 'utf8' })
        if (mode === 'fresh') assert.equal(r.status, 0, r.stderr)
        else if (mode === 'fail') assert.equal(r.status, 23, 'pub failure must stop bootstrap')
        else assert.notEqual(r.status, 0, mode + ' drift must fail rather than silently test new dependencies')
      }
    })
  }
  if (workflow === 'ci-mobile-native.yml') test('native Android compile cannot resolve dependencies again', () => {
    const apk = source.split('\n').find(line => /^\s+flutter build apk /.test(line))
    assert.match(apk, /(?:^| )--no-pub(?: |$)/)
  })
}

test('the dependency policy is checked before CI job selection', () => {
  const source = read('.github/workflows/ci.yml')
  const guard = source.indexOf('node --test scripts/test-mobile-ci-lockfile.mjs')
  assert.ok(guard >= 0 && guard < source.indexOf('- id: select'))
})

test('analysis and filtered generation also reject implicit resolver drift', () => {
  const source = read('.github/workflows/ci-mobile.yml')
  const steps = source.split(/^      - /m).slice(1)
  for (const title of ['Analyze', 'Require fresh generated login contract']) {
    const step = steps.find(step => step.startsWith('name: ' + title + '\n'))
    assert.ok(step)
    assert.ok(runScript(step).includes("':(top)apps/mobile/pubspec.lock'"))
  }
  assert.match(source, /run: flutter test --no-pub --total-shards/)
})
