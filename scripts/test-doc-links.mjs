import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdtemp, mkdir, writeFile, rm, unlink, symlink } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { test } from 'node:test';
import { verifyLinks } from './verify-doc-links.mjs';

async function repository(t, files) {
  const root = await mkdtemp(path.join(tmpdir(), 'yourtj-doc-links-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  execFileSync('git', ['init', '-q', root]);
  for (const [name, content] of Object.entries(files)) {
    await mkdir(path.dirname(path.join(root, name)), { recursive: true });
    await writeFile(path.join(root, name), content);
  }
  return root;
}

test('checks root and nested READMEs and fork docs, including new files', async (t) => {
  const root = await repository(t, {
    'README.md': '[missing](missing.md)',
    'README-course.md': '[missing](missing.md)',
    'apps/forum/README_ZH.md': '[missing](missing.md)',
    'services/search/readme.md': '[missing](missing.md)',
    'apps/gooseforum/docs/user/configuration.md': '[missing](missing.md)',
    'docs/README.md': '# Documentation',
    'apps/forum/source.dart': '[ignored](missing.md)',
    '.gitignore': 'node_modules/\nresearch/\n',
    'node_modules/dependency/README.md': '[ignored](missing.md)',
    'research/README.md': '[ignored](missing.md)',
  });
  execFileSync('git', ['add', 'README.md'], { cwd: root });
  const { errors, files } = await verifyLinks(root);
  assert.deepEqual(files.sort(), [
    'README-course.md', 'README.md', 'apps/forum/README_ZH.md',
    'apps/gooseforum/docs/user/configuration.md', 'docs/README.md',
    'services/search/readme.md',
  ].sort());
  assert.equal(errors.length, 5);
  assert.ok(errors.every((error) => error.includes('missing.md')));
});

test('removed tracked documentation does not fail the gate', async (t) => {
  const root = await repository(t, { 'README.md': '# Readme', 'docs/old.md': '# Old' });
  execFileSync('git', ['add', '.'], { cwd: root });
  await unlink(path.join(root, 'docs/old.md'));
  assert.deepEqual((await verifyLinks(root)).errors, []);
});

test('validates Chinese anchors while rejecting stale decorated anchors', async (t) => {
  const root = await repository(t, {
    'docs/README.md': '[开始](guide.md#快速开始)\n[配置](guide.md#configuration)',
    'docs/guide.md': '# Guide\n## 快速开始\n## Configuration\n',
  });
  assert.deepEqual((await verifyLinks(root)).errors, []);
  await writeFile(path.join(root, 'docs/README.md'), '[旧链接](guide.md#🚀-快速开始)');
  assert.match((await verifyLinks(root)).errors[0], /fragment.*not found/);
});

test('does not follow symlinked documentation outside the repository', async (t) => {
  const root = await repository(t, { 'README.md': '# Readme' });
  const external = await mkdtemp(path.join(tmpdir(), 'yourtj-external-doc-'));
  t.after(() => rm(external, { recursive: true, force: true }));
  await writeFile(path.join(external, 'README.md'), '[missing](missing.md)');
  await mkdir(path.join(root, 'docs'));
  await symlink(path.join(external, 'README.md'), path.join(root, 'docs/linked.md'));
  const { files, errors } = await verifyLinks(root);
  assert.deepEqual(files, ['README.md']);
  assert.deepEqual(errors, []);
});

test('fails explicitly if Git cannot enumerate the repository', async (t) => {
  const root = await repository(t, { 'README.md': '# Readme' });
  await rm(path.join(root, '.git'), { recursive: true });
  await assert.rejects(verifyLinks(root), /git ls-files/);
});
