#!/usr/bin/env node
// Decision files own the list; the README stays a stable entry point.
import { readdir, readFile } from 'node:fs/promises';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { governancePaths, readRepoManifest } from './governance-config.mjs';
import { verifyDir } from './verify-decisions.mjs';

export async function generateIndex(dir, linkBase = 'docs/decisions/') {
  const errors = await verifyDir(dir);
  if (errors.length) throw new Error(errors.join('\n'));
  const entries = await readdir(dir, { withFileTypes: true });
  const files = entries.filter(e => e.isFile() && /^\d{4}-.+\.md$/.test(e.name)).map(e => e.name).sort();
  const lines = [
    '# Decision log index',
    '',
    'Generated from numbered MADR files by `node scripts/generate-decisions-index.mjs`.',
    '',
    `Source: [decision directory](${linkBase})`,
    '',
  ];
  for (const file of files) {
    const text = await readFile(path.join(dir, file), 'utf8');
    const title = (text.match(/^# (.+)$/m)?.[1] ?? file).replace(/[\[\]\\]/g, '\\$&');
    const status = text.match(/^## Status\n+([^\n]+)/m)[1].replace(/\[([^\]]+)\]\([^)]+\)/g, '$1');
    lines.push("- [" + file.slice(0, 4) + ": " + title + "](" + linkBase + file + ") (" + status + ")");
  }
  return lines.join('\n') + '\n';
}

async function main() {
  const manifest = await readRepoManifest(process.cwd());
  const dir = path.join(process.cwd(), governancePaths(manifest).decisions);
  process.stdout.write(await generateIndex(dir, process.argv[2]));
}

if (process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url) {
  main().catch(error => { console.error(error.message); process.exitCode = 1; });
}
