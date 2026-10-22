import assert from 'node:assert/strict';
import { test } from 'node:test';
import { selectedScopes } from '../.github/oryn/validate.mjs';

test('docs need governance, without application suites', () => {
  assert.deepEqual(selectedScopes(['docs/operations/example.md']), ['governance']);
});
test('database changes require backend and PostgreSQL', () => {
  assert.deepEqual(selectedScopes(['apps/gooseforum/app/models/forum/topic/topic.go']), ['governance', 'backend', 'postgres']);
});
test('HTTP changes also verify the controlled API contract', () => {
  assert.deepEqual(selectedScopes(['apps/gooseforum/app/http/controllers/api/post.go']), ['governance', 'backend', 'contract']);
});
test('web and shared contract changes include their consumers', () => {
  assert.deepEqual(selectedScopes(['packages/api-contract/openapi.yaml']), ['governance', 'backend', 'frontend', 'contract', 'mobile']);
});
test('unknown executable surfaces fail closed by checking every application domain', () => {
  assert.deepEqual(selectedScopes(['scripts/new-build.sh']), ['governance', 'backend', 'postgres', 'frontend', 'contract', 'mobile']);
});

test('scope includes committed PR changes, staged repair and untracked files', async () => {
  const { mkdtempSync, writeFileSync, rmSync } = await import('node:fs');
  const { tmpdir } = await import('node:os');
  const { join } = await import('node:path');
  const { execFileSync } = await import('node:child_process');
  const { changedPaths } = await import('../.github/oryn/validate.mjs');
  const directory = mkdtempSync(join(tmpdir(), 'oryn-scope-'));
  const previous = process.cwd();
  const git = (...args) => execFileSync('git', args, { cwd: directory, encoding: 'utf8' }).trim();
  try {
    git('init', '-q');
    git('config', 'user.name', 'test');
    git('config', 'user.email', 'test@example.invalid');
    writeFileSync(join(directory, 'base.txt'), 'base');
    git('add', '.');
    git('commit', '-qm', 'base');
    const base = git('rev-parse', 'HEAD');
    writeFileSync(join(directory, 'committed.txt'), 'PR');
    git('add', '.');
    git('commit', '-qm', 'PR');
    writeFileSync(join(directory, 'staged.txt'), 'repair');
    git('add', '.');
    writeFileSync(join(directory, 'untracked.txt'), 'repair');
    process.chdir(directory);
    assert.deepEqual(changedPaths(base).sort(), ['committed.txt', 'staged.txt', 'untracked.txt']);
    assert.throws(() => changedPaths('0'.repeat(40)), /Cannot establish/);
  } finally {
    process.chdir(previous);
    rmSync(directory, { recursive: true, force: true });
  }
});

// The checked-in concurrency expression uses the JS-compatible boolean/format subset of
// GitHub expressions. Evaluate that exact text with typed workflow-dispatch fixtures.
async function orynConcurrency(inputs, event = 'workflow_dispatch', runId = 101) {
  const { readFileSync } = await import('node:fs');
  const workflow = readFileSync(new URL('../.github/workflows/oryn.yml', import.meta.url), 'utf8');
  const expression = workflow.match(/^  group: \$\{\{ (.+) \}\}$/m)?.[1];
  assert.ok(expression, 'Oryn must declare its workflow concurrency group');
  const evaluate = new Function('github', 'inputs', 'vars', 'format', `return (${expression});`);
  const format = (template, ...values) => template.replace(/\{(\d+)\}/g, (_, index) => String(values[index]));
  return evaluate({ event_name: event, run_id: runId, event: { comment: { user: { type: 'User' } } } },
    { release_notes_smoke: false, preflight: false, publish: false, ...inputs },
    { ORYN_ENABLED: 'true', ORYN_EVENT_ENABLED: 'true' }, format);
}

test('non-publishing manual probes can run without waiting for the publication queue', async () => {
  assert.equal(await orynConcurrency({}), 'oryn-probe-101');
  assert.equal(await orynConcurrency({}, 'workflow_dispatch', 102), 'oryn-probe-102');
  assert.equal(await orynConcurrency({ preflight: true, publish: true }), 'oryn-probe-101');
  assert.equal(await orynConcurrency({ release_notes_smoke: true }), 'oryn-release-smoke-101');
});

test('publishing manual work and automatic events keep their shared receipt-writer queue', async () => {
  assert.equal(await orynConcurrency({ publish: true }), 'oryn-sweep');
  for (const event of ['schedule', 'pull_request_target', 'issues', 'issue_comment']) {
    assert.equal(await orynConcurrency({}, event), 'oryn-sweep');
  }
});
