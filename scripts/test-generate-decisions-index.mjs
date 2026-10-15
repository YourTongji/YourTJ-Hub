import assert from 'node:assert/strict';
import { mkdtemp, mkdir, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { test } from 'node:test';
import { generateIndex } from './generate-decisions-index.mjs';

const record = (title, status = 'Accepted') => `# ${title}

## Status
${status}
Class: process

## Context and Problem Statement
Context.
## Decision Drivers
Drivers.
## Considered Options
Options.
## Decision Outcome
Outcome.
## Pros and Cons of the Options
Tradeoffs.
## Links
None.
`;

test('discovers parallel records without changing the shared README', async (t) => {
  const dir = await mkdtemp(path.join(tmpdir(), 'yourtj-decision-index-'));
  t.after(() => rm(dir, { recursive: true, force: true }));
  await writeFile(path.join(dir, 'README.md'), '# Stable entry point\n');
  await mkdir(path.join(dir, 'nested'));
  await writeFile(path.join(dir, '0088-beta.md'), record('Beta | decision'));
  await writeFile(path.join(dir, '0084-alpha.md'), record('Alpha decision', 'Proposed'));
  const report = await generateIndex(dir, 'https://example.com/blob/sha/docs/decisions/');
  assert.ok(report.indexOf('0084') < report.indexOf('0088'));
  assert.match(report, /\[0084: Alpha decision\]\(https:\/\/example.com\/blob\/sha\/docs\/decisions\/0084-alpha.md\)/);
  assert.match(report, /Beta \| decision/);
  assert.match(report, /Proposed/);
  assert.match(report, /Accepted/);
  assert.equal(await readFile(path.join(dir, 'README.md'), 'utf8'), '# Stable entry point\n');
  await writeFile(path.join(dir, '0088-duplicate.md'), record('Duplicate'));
  await assert.rejects(generateIndex(dir), /duplicate decision number 0088/);
});

test('refuses to report an invalid decision as current', async (t) => {
  const dir = await mkdtemp(path.join(tmpdir(), 'yourtj-decision-index-'));
  t.after(() => rm(dir, { recursive: true, force: true }));
  await writeFile(path.join(dir, '0088-invalid.md'), record('Invalid', 'Unknown'));
  await assert.rejects(generateIndex(dir), /invalid status/);
});
