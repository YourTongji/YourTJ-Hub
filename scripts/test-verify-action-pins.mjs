import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';

import { findActionPinDrift, formatFinding, parseUsesPins } from './verify-action-pins.mjs';

const script = fileURLToPath(new URL('./verify-action-pins.mjs', import.meta.url));

const workflowSha = 'a'.repeat(40);
const compositeSha = 'b'.repeat(40);
const otherSha = 'c'.repeat(40);

// Build a repository-shaped fixture: only the files a case needs.
function fixture(files) {
  const root = mkdtempSync(join(tmpdir(), 'yourtj-action-pins-'));
  for (const [rel, content] of Object.entries(files)) {
    const absolute = join(root, ...rel.split('/'));
    mkdirSync(join(absolute, '..'), { recursive: true });
    writeFileSync(absolute, content);
  }
  return root;
}

function workflow(pins) {
  return ['name: fixture', 'on:', '  workflow_call:', 'jobs:', '  build:', '    runs-on: ubuntu-latest', '    steps:', ...pins.map((pin) => `      - uses: ${pin}`), ''].join('\n');
}

function composite(pins) {
  return ['name: fixture', 'description: fixture', 'runs:', '  using: composite', '  steps:', ...pins.map((pin) => `    - uses: ${pin}`), ''].join('\n');
}

// Derive the expected report line from the fixture itself instead of hardcoding it,
// so the assertion still means "points at the pin" if the fixture shape changes.
function pinLine(content) {
  return content.split('\n').findIndex((line) => line.includes('uses:')) + 1;
}

test('a composite action that matches the workflows produces no finding', async (t) => {
  const root = fixture({
    '.github/workflows/ci.yml': workflow([`actions/checkout@${workflowSha}`, `actions/setup-node@${workflowSha}`]),
    '.github/actions/build-binary/action.yml': composite([`actions/setup-node@${workflowSha}`]),
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const { findings, compositePins } = await findActionPinDrift(root);
  assert.deepEqual(findings, []);
  assert.equal(compositePins.length, 1);
});

test('a composite action left behind by a workflow bump is reported', async (t) => {
  const compositeContent = composite([`actions/upload-artifact@${compositeSha}`]);
  const root = fixture({
    '.github/workflows/ci.yml': workflow([`actions/upload-artifact@${workflowSha}`]),
    '.github/actions/build-binary/action.yml': compositeContent,
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const { findings } = await findActionPinDrift(root);
  assert.equal(findings.length, 1);
  assert.equal(findings[0].kind, 'composite-behind');
  assert.equal(findings[0].key, 'actions/upload-artifact');
  assert.equal(findings[0].file, '.github/actions/build-binary/action.yml');
  assert.equal(findings[0].line, pinLine(compositeContent));
  assert.deepEqual(findings[0].workflowRefs, [workflowSha]);
  const report = formatFinding(findings[0]);
  assert.ok(report.includes(`build-binary/action.yml:${pinLine(compositeContent)}`), report);
  assert.ok(report.includes(`actions/upload-artifact@${compositeSha}`), report);
  assert.ok(report.includes(`workflows/** pins ${workflowSha}`), report);
});

test('an action pinned only inside a composite action has nothing to compare', async (t) => {
  const root = fixture({
    '.github/workflows/ci.yml': workflow([`actions/checkout@${workflowSha}`]),
    '.github/actions/setup-oryn/action.yml': composite([`oven-sh/setup-bun@${compositeSha}`]),
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const { findings } = await findActionPinDrift(root);
  assert.deepEqual(findings, []);
});

test('the workflows pinning one action at two refs is reported', async (t) => {
  const root = fixture({
    '.github/workflows/ci.yml': workflow([`actions/checkout@${workflowSha}`]),
    '.github/workflows/ci-docs.yml': workflow([`actions/checkout@${otherSha}`]),
    '.github/actions/build-binary/action.yml': composite([`actions/checkout@${workflowSha}`]),
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const { findings } = await findActionPinDrift(root);
  assert.deepEqual(findings.map((finding) => finding.kind), ['workflow-divergent']);
  assert.equal(findings[0].key, 'actions/checkout');
  assert.deepEqual(findings[0].refs, [workflowSha, otherSha].sort());
});

test('local, docker and expression uses values are not treated as pins', async (t) => {
  const root = fixture({
    '.github/workflows/ci.yml': workflow(['./.github/actions/build-binary', 'docker://alpine:3.19', 'actions/checkout@${{ inputs.sha }}']),
    '.github/actions/build-binary/action.yml': composite(['./.github/actions/nested', 'docker://alpine:3.19']),
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const { findings, workflowPins, compositePins } = await findActionPinDrift(root);
  assert.deepEqual(findings, []);
  assert.deepEqual(workflowPins, []);
  assert.deepEqual(compositePins, []);
});

test('only the repository-root workflows are in scope', async (t) => {
  const root = fixture({
    '.github/workflows/ci.yml': workflow([`actions/checkout@${workflowSha}`]),
    '.github/actions/build-binary/action.yml': composite([`actions/checkout@${workflowSha}`]),
    // A nested tree is never executed by GitHub and must not create a false finding.
    'apps/gooseforum/.github/workflows/ci.yml': workflow([`actions/upload-artifact@${compositeSha}`]),
  });
  t.after(() => rmSync(root, { recursive: true, force: true }));

  const { findings, workflowFiles } = await findActionPinDrift(root);
  assert.deepEqual(workflowFiles, ['.github/workflows/ci.yml']);
  assert.deepEqual(findings, []);
});

test('a trailing version comment does not change the parsed ref', () => {
  const pins = parseUsesPins('      - uses: actions/upload-artifact@' + workflowSha + ' # v7.0.2\n', '.github/workflows/ci.yml');
  assert.deepEqual(pins, [{ file: '.github/workflows/ci.yml', line: 1, key: 'actions/upload-artifact', ref: workflowSha }]);
});

test('the CLI exits 1 on drift and 0 when the pins agree', (t) => {
  const drifting = fixture({
    '.github/workflows/ci.yml': workflow([`actions/upload-artifact@${workflowSha}`]),
    '.github/actions/build-binary/action.yml': composite([`actions/upload-artifact@${compositeSha}`]),
  });
  const consistent = fixture({
    '.github/workflows/ci.yml': workflow([`actions/upload-artifact@${workflowSha}`]),
    '.github/actions/build-binary/action.yml': composite([`actions/upload-artifact@${workflowSha}`]),
  });
  t.after(() => {
    rmSync(drifting, { recursive: true, force: true });
    rmSync(consistent, { recursive: true, force: true });
  });

  const failed = spawnSync(process.execPath, [script, drifting], { encoding: 'utf8' });
  assert.equal(failed.status, 1, failed.stderr);
  assert.match(failed.stderr, /action pins must be consistent/);
  assert.match(failed.stderr, /build-binary\/action\.yml:\d+/);

  const passed = spawnSync(process.execPath, [script, consistent], { encoding: 'utf8' });
  assert.equal(passed.status, 0, passed.stderr);
  assert.match(passed.stdout, /verify-action-pins: OK/);
});

test('the summary header also fits a workflows-only finding', (t) => {
  // A workflow-divergent finding involves no composite action, so the header must not
  // describe it as a composite-vs-workflow mismatch.
  const divergent = fixture({
    '.github/workflows/ci.yml': workflow([`actions/checkout@${workflowSha}`]),
    '.github/workflows/ci-docs.yml': workflow([`actions/checkout@${otherSha}`]),
    '.github/actions/build-binary/action.yml': composite([`actions/checkout@${workflowSha}`]),
  });
  t.after(() => rmSync(divergent, { recursive: true, force: true }));

  const result = spawnSync(process.execPath, [script, divergent], { encoding: 'utf8' });
  assert.equal(result.status, 1, result.stderr);
  assert.match(result.stderr, /action pins must be consistent/);
  assert.doesNotMatch(result.stderr, /composite action pins must match/);
  assert.match(result.stderr, /pins actions\/checkout at more than one ref/);
});
