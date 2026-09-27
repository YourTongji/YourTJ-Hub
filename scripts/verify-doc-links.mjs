#!/usr/bin/env node
// verify-doc-links.mjs — documentation link gate.
// Checks relative Markdown links and anchors in repository documentation:
// governance docs, root/nested READMEs and fork-owned GooseForum docs.
// Git excludes ignored local artifacts and dependency trees from discovery.
// Zero npm dependencies; Git and Node >= 18. Exits non-zero on any violation.
import { readFile, stat, lstat } from 'node:fs/promises';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import path from 'node:path';
import process from 'node:process';
import { pathToFileURL } from 'node:url';
import { readRepoManifest, safeRepositoryRelativePath } from './governance-config.mjs';

export const SCOPED_PATHS = ['AGENTS.md', 'CLAUDE.md', 'docs', 'CONTRIBUTING.md', 'SECURITY.md', 'CODE_OF_CONDUCT.md', 'apps/gooseforum/docs'];
const execFileAsync = promisify(execFile);

// Links inside fenced code blocks and inline code are examples, not real
// links (e.g. `[0001](0001-title.md)` in a format description); strip them
// before extraction so they do not false-positive.
export function stripCode(text) {
  return text
    .replace(/```[\s\S]*?```/g, (m) => ' '.repeat(m.length))
    .replace(/`[^`]*`/g, (m) => ' '.repeat(m.length));
}

// Extract markdown links [text](target) and fragment references.
export function extractLinks(text) {
  const links = [];
  const re = /\[[^\]]*\]\(([^)]+)\)/g;
  let m;
  while ((m = re.exec(stripCode(text))) !== null) {
    const target = m[1].trim();
    if (target.startsWith('http://') || target.startsWith('https://') || target.startsWith('mailto:')) continue;
    if (target.startsWith('<') && target.endsWith('>')) continue; // autolink
    const [filePart, fragment] = target.split('#');
    links.push({ filePart: filePart || null, fragment: fragment || null, raw: target });
  }
  return links;
}

// Collect headings (as slug anchors) from a markdown file.
export function headingAnchors(text) {
  const anchors = new Set();
  const re = /^(#{1,6})\s+(.+)$/gm;
  let m;
  while ((m = re.exec(text)) !== null) {
    anchors.add(slugify(m[2]));
  }
  return anchors;
}

export function slugify(heading) {
  return heading
    .trim()
    .toLowerCase()
    .replace(/[^\p{L}\p{M}\p{N}_\s-]/gu, '')
    .replace(/\s+/g, '-');
}

// Check a single file: all links resolve.
export async function checkFile(fileAbs, repoRoot, { allowedBareDirectories = new Set() } = {}) {
  const errors = [];
  let text;
  try {
    text = await readFile(fileAbs, 'utf8');
  } catch {
    errors.push(`${path.relative(repoRoot, fileAbs)}: unreadable`);
    return errors;
  }
  const fileDir = path.dirname(fileAbs);
  const links = extractLinks(text);
  for (const l of links) {
    if (l.filePart === null) {
      // Fragment-only link: must resolve in this file.
      const anchors = headingAnchors(text);
      if (l.fragment && !anchors.has(l.fragment)) {
        errors.push(`${path.relative(repoRoot, fileAbs)}: fragment "#${l.fragment}" not found in file`);
      }
      continue;
    }
    // Relative link: resolve against fileDir, allow ../
    const targetAbs = path.resolve(fileDir, l.filePart);
    // Reject escaping the repo root
    const rel = path.relative(repoRoot, targetAbs);
    if (rel.startsWith('..') || path.isAbsolute(rel) || !safeRepositoryRelativePath(rel)) {
      errors.push(`${path.relative(repoRoot, fileAbs)}: link "${l.raw}" has an unsafe repository target`);
      continue;
    }
    let st;
    try {
      st = await stat(targetAbs);
    } catch {
      errors.push(`${path.relative(repoRoot, fileAbs)}: link "${l.raw}" -> missing ${l.filePart}`);
      continue;
    }
    if (st.isDirectory()) {
      if (allowedBareDirectories.has(path.resolve(targetAbs))) continue;
      // Directory link: check for README.md inside
      const idx = path.join(targetAbs, 'README.md');
      try {
        await stat(idx);
      } catch {
        errors.push(`${path.relative(repoRoot, fileAbs)}: link "${l.raw}" -> directory ${l.filePart} without README.md`);
      }
      continue;
    }
    if (l.fragment) {
      const targetText = await readFile(targetAbs, 'utf8');
      const anchors = headingAnchors(targetText);
      if (!anchors.has(l.fragment)) {
        errors.push(`${path.relative(repoRoot, fileAbs)}: fragment "#${l.fragment}" not found in ${l.filePart}`);
      }
    }
  }
  return errors;
}

export async function collectScopedFiles(repoRoot) {
  // Include tracked and new non-ignored docs without walking build/SDK caches.
  // A Git failure must fail the gate rather than silently checking no files.
  const { stdout } = await execFileAsync('git', [
    'ls-files', '--cached', '--others', '--exclude-standard', '-z',
  ], { cwd: repoRoot, maxBuffer: 16 * 1024 * 1024 });
  const candidates = [...new Set(stdout.split('\0').filter((file) =>
    file.endsWith('.md') && (
      /^readme.*\.md$/i.test(path.basename(file)) ||
      SCOPED_PATHS.some((scope) => file === scope || file.startsWith(`${scope}/`))
    )
  ))].sort();
  const files = [];
  for (const file of candidates) {
    try {
      // Deleted tracked files are absent from the proposed tree; do not follow
      // symlinks into external checkouts or local dependency directories.
      if ((await lstat(path.join(repoRoot, file))).isFile()) files.push(file);
    } catch (error) {
      if (error.code !== 'ENOENT') throw error;
    }
  }
  return files;
}

export async function verifyLinks(repoRoot) {
  const errors = [];
  const files = await collectScopedFiles(repoRoot);
  const manifest = await readRepoManifest(repoRoot);
  const allowedBareDirectories = new Set(
    Object.values(manifest?.governance?.externalSources ?? {})
      .filter((source) => source?.type === 'repository' && source.location)
      .map((source) => path.resolve(repoRoot, source.location))
  );
  for (const f of files) {
    const fileErrors = await checkFile(path.join(repoRoot, f), repoRoot, { allowedBareDirectories });
    errors.push(...fileErrors);
  }
  return { errors, files };
}

async function main() {
  const repoRoot = path.resolve(process.argv[2] ?? '.');
  const { errors, files } = await verifyLinks(repoRoot);
  if (errors.length) {
    for (const e of errors) console.error(`verify-doc-links: ${e}`);
    process.exit(1);
  }
  console.log(`verify-doc-links: OK (${files.length} files)`);
}

const isMain =
  process.argv[1] &&
  pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url;
if (isMain) {
  main().catch((e) => {
    console.error('verify-doc-links: fatal', e);
    process.exit(2);
  });
}
