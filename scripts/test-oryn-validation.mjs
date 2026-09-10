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
  assert.deepEqual(selectedScopes(['packages/api-contract/openapi.yaml']), ['governance', 'frontend', 'contract', 'mobile']);
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
