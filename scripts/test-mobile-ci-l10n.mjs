import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { test } from 'node:test';

const workflow = readFileSync(new URL('../.github/workflows/ci-mobile.yml', import.meta.url), 'utf8');
const analyze = workflow.split(/^  test:/m)[0];
const steps = analyze.split(/^      - /m).slice(1);
const packagePath = 'apps/mobile/packages/forum_app';

function freshnessStep() {
  const step = steps.find(step => step.startsWith('name: Check generated localizations\n'));
  assert.ok(step, 'mobile analysis must check ARB-generated localization freshness');
  assert.match(step, /^        working-directory: apps\/mobile\/packages\/forum_app$/m);
  const run = step.match(/^        run: \|\n((?:          .*\n|\n)+)/m);
  assert.ok(run, 'freshness gate must have an executable shell block');
  return run[1].replace(/^          /gm, '');
}

function assertGenerationOrder(steps) {
  const freshness = steps.findIndex(step => step.startsWith('name: Check generated localizations\n'));
  for (const command of ['dart run melos bootstrap', 'dart run melos run analyze']) {
    const index = steps.findIndex(step => step.includes(command));
    assert.ok(index >= 0 && index < freshness, `${command} must precede localization generation`);
  }
  // Filtered build_runner can remove unrelated generated files on its first run.
  // If present, it must not run before either analysis or localization generation.
  for (const [index, step] of steps.entries()) {
    if (/^\s+dart run build_runner\b/m.test(step)) assert.ok(index > freshness);
  }
}

test('localization generation runs after bootstrap/analyze and before filtered build_runner', () => {
  freshnessStep();
  assertGenerationOrder(steps);
  const config = readFileSync(new URL(`../${packagePath}/l10n.yaml`, import.meta.url), 'utf8');
  assert.match(config, /^arb-dir: lib\/l10n$/m);
  assert.doesNotMatch(config, /^output-dir:/m, 'update the gate when moving generated output');
});

test('adjacent filtered generation comments do not change command ordering', () => {
  const filteredGeneration = `      # Filtered build_runner cleanup may remove other generated outputs.
      # Keep this last so analysis checks the complete committed workspace.
      - name: Require fresh generated login contract
        working-directory: apps/mobile/packages/core
        run: |
          dart run build_runner build --delete-conflicting-outputs --build-filter='lib/src/gen/auth.*.dart'
          git diff --exit-code -- lib/src/gen/auth.freezed.dart lib/src/gen/auth.g.dart
`;
  const correct = `${analyze}${filteredGeneration}`.split(/^      - /m).slice(1);
  assertGenerationOrder(correct);
  const incorrect = analyze.replace('      - name: Check generated localizations', `${filteredGeneration}      - name: Check generated localizations`).split(/^      - /m).slice(1);
  assert.throws(() => assertGenerationOrder(incorrect));
});

function git(cwd, ...args) {
  const result = spawnSync('git', args, { cwd, encoding: 'utf8' });
  assert.equal(result.status, 0, result.stderr);
  return result.stdout;
}

function runGate(t, mutation = '', prepare = () => {}) {
  const script = freshnessStep();
  const root = mkdtempSync(join(tmpdir(), 'mobile-l10n-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const pkg = join(root, packagePath);
  const output = join(pkg, 'lib/l10n');
  mkdirSync(output, { recursive: true });
  const bin = join(root, 'bin');
  mkdirSync(bin);
  writeFileSync(join(output, 'app_localizations.dart'), '// fresh generated output\n');
  writeFileSync(join(output, 'app_en.arb'), '{}\n');
  git(root, 'init', '--quiet');
  git(root, 'add', '.');
  git(root, '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', 'commit', '--quiet', '-m', 'fixture');
  writeFileSync(join(bin, 'flutter'), `#!/usr/bin/env bash
set -euo pipefail
test "$*" = gen-l10n
test -f lib/l10n/app_en.arb
case "$L10N_MUTATION" in
  tracked) printf '// regenerated output\\n' > lib/l10n/app_localizations.dart ;;
  untracked) printf '// new locale\\n' > lib/l10n/app_localizations_es.dart ;;
  fail) echo 'invalid ARB fixture' >&2; exit 23 ;;
esac
`, { mode: 0o755 });
  prepare(root, output);
  return {
    result: spawnSync('bash', ['-e', '-o', 'pipefail', '-c', script], {
      cwd: pkg,
      env: { ...process.env, PATH: `${bin}:${process.env.PATH}`, L10N_MUTATION: mutation },
      encoding: 'utf8',
    }),
    root,
    output,
  };
}

test('fresh generated outputs pass despite unrelated untracked files', t => {
  const { result } = runGate(t, '', root => writeFileSync(join(root, 'unrelated.txt'), 'local note\n'));
  assert.equal(result.status, 0, result.stderr);
});

for (const mutation of ['tracked', 'untracked']) {
  test(`${mutation} generated drift fails and preserves the output for inspection`, t => {
    const { result, output } = runGate(t, mutation);
    assert.equal(result.status, 1, result.stdout + result.stderr);
    assert.match(result.stdout + result.stderr, /app_localizations/);
    assert.match(result.stdout + result.stderr, /flutter gen-l10n/);
    const file = mutation === 'tracked' ? 'app_localizations.dart' : 'app_localizations_es.dart';
    assert.match(readFileSync(join(output, file), 'utf8'), /regenerated output|new locale/);
  });
}

for (const state of ['staged', 'deleted']) {
  test(`${state} output drift left by earlier commands cannot pass`, t => {
    const { result } = runGate(t, '', (root, output) => {
      const file = join(output, 'app_localizations.dart');
      if (state === 'deleted') rmSync(file);
      else {
        writeFileSync(file, '// staged drift\n');
        git(root, 'add', file);
      }
    });
    assert.equal(result.status, 1, result.stdout + result.stderr);
    assert.match(result.stdout + result.stderr, /app_localizations\.dart/);
  });
}

test('generator failure cannot be hidden by a clean Git status', t => {
  const { result } = runGate(t, 'fail');
  assert.equal(result.status, 23, result.stdout + result.stderr);
  assert.match(result.stderr, /invalid ARB fixture/);
});

for (const path of [`${packagePath}/lib/l10n/app_en.arb`, `${packagePath}/lib/l10n/app_localizations.dart`, `${packagePath}/l10n.yaml`]) {
  test(`localization input selects the mobile gate: ${path}`, () => {
    const result = spawnSync('python3', ['scripts/ci/select_inputs.py', '--paths-json'], { input: JSON.stringify([path]), encoding: 'utf8' });
    assert.equal(result.status, 0, result.stderr);
    assert.equal(JSON.parse(result.stdout).mobile.run, true);
  });
}
