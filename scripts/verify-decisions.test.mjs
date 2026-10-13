import assert from 'node:assert/strict';
import { mkdtemp, mkdir, writeFile, rm } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { test } from 'node:test';
import { verifyDir } from './verify-decisions.mjs';

const decision = (status = 'Proposed') => `## Status
${status}

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

async function withRecords(names, fn) {
  const root = await mkdtemp(path.join(os.tmpdir(), 'yourtj-madr-'));
  try {
    const dir = path.join(root, 'decisions');
    await mkdir(dir);
    for (const name of names) await writeFile(path.join(dir, name), decision());
    await fn(dir);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
}

test('accepts gaps reserved by concurrent PRs', async () => {
  await withRecords(['0000-first.md', '0082-existing.md', '0084-reserved.md'], async dir => {
    assert.deepEqual(await verifyDir(dir), []);
  });
});

test('rejects multiple records claiming the same number', async () => {
  await withRecords(['0000-first.md', '0084-a.md', '0084-b.md'], async dir => {
    const errors = await verifyDir(dir);
    assert.ok(errors.some(error => error.includes('duplicate decision number 0084')), errors.join('\n'));
  });
});

test('keeps enforcing sections and superseded targets with gaps', async () => {
  await withRecords(['0000-first.md', '0084-proposed.md'], async dir => {
    await writeFile(path.join(dir, '0084-proposed.md'), decision('Superseded by [0083](0083-missing.md)'));
    const errors = await verifyDir(dir);
    assert.ok(errors.some(error => error.includes('superseded target missing')), errors.join('\n'));
  });
});
