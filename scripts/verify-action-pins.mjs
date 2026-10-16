#!/usr/bin/env node
// verify-action-pins.mjs — action-pin consistency check (issue #1212).
//
// Every third-party action referenced from a local composite action
// (.github/actions/<name>/action.yml) must be pinned to the same ref as the same
// action in the repository-root .github/workflows/** files.
//
// Why this exists: Dependabot's `github-actions` ecosystem only scans
// .github/workflows/** plus the repository-root action.yml, so pins inside
// .github/actions/** are outside its scan set — they receive no scheduled
// version-update PR. (Dependabot alerts never cover them either: GitHub Actions
// manifests must live under .github/workflows/, and alerts are only generated for
// semantic-version references, not SHA pins.) Without this check they drift
// silently as soon as a workflow pin is bumped, and the drift is invisible in
// review because the Dependabot PR looks complete. See docs/development/testing.md
// and issue #1212.
//
// Scope: the repository-root .github/workflows directory only. A nested
// apps/**/.github/workflows tree is never executed by GitHub and is ignored here.
//
// Zero dependencies; Node >= 18. Exits 1 when a pin drifts, 0 when consistent.
import { readdir, readFile } from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';
import { pathToFileURL } from 'node:url';

const WORKFLOW_DIRECTORY = '.github/workflows';
const COMPOSITE_DIRECTORY = '.github/actions';
const COMPOSITE_MANIFEST_NAMES = new Set(['action.yml', 'action.yaml']);

// A step's `uses:` value, with or without a leading list dash.
const USES_PATTERN = /^\s*(?:-\s*)?uses:\s*(\S+)/;

function isWorkflowFile(name) {
  return /\.ya?ml$/.test(name);
}

export function isCompositeManifest(name) {
  return COMPOSITE_MANIFEST_NAMES.has(name);
}

// Extract `owner/repo[/path]@ref` pins from one file's content. Local actions (./),
// docker images and expression values carry no comparable pin and are skipped.
export function parseUsesPins(content, file) {
  const pins = [];
  content.split(/\r?\n/).forEach((line, index) => {
    const match = USES_PATTERN.exec(line);
    if (!match) return;
    const value = match[1].trim().replace(/^["']|["']$/g, '');
    if (value.startsWith('./') || value.startsWith('docker://') || value.includes('${{')) return;
    const at = value.lastIndexOf('@');
    if (at <= 0) return;
    const key = value.slice(0, at);
    const ref = value.slice(at + 1);
    if (!key.includes('/') || !ref) return;
    pins.push({ file, line: index + 1, key, ref });
  });
  return pins;
}

async function readDirectory(absolute) {
  try {
    return await readdir(absolute, { withFileTypes: true });
  } catch {
    return [];
  }
}

async function collectPins(repoRoot, relativeFiles) {
  const pins = [];
  for (const rel of relativeFiles) {
    const content = await readFile(path.join(repoRoot, rel), 'utf8');
    pins.push(...parseUsesPins(content, rel));
  }
  return pins;
}

export async function listWorkflowFiles(repoRoot) {
  return (await readDirectory(path.join(repoRoot, WORKFLOW_DIRECTORY)))
    .filter((entry) => entry.isFile() && isWorkflowFile(entry.name))
    .map((entry) => `${WORKFLOW_DIRECTORY}/${entry.name}`)
    .sort();
}

export async function listCompositeManifests(repoRoot) {
  const files = [];
  const directories = (await readDirectory(path.join(repoRoot, COMPOSITE_DIRECTORY)))
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .sort();
  for (const name of directories) {
    for (const entry of await readDirectory(path.join(repoRoot, COMPOSITE_DIRECTORY, name))) {
      if (entry.isFile() && isCompositeManifest(entry.name)) {
        files.push(`${COMPOSITE_DIRECTORY}/${name}/${entry.name}`);
      }
    }
  }
  return files;
}

// Findings:
//  - composite-behind: a composite action pins an action that the workflows pin at a
//    different ref (the drift Dependabot cannot see).
//  - workflow-divergent: the workflows themselves pin one action at more than one ref.
export async function findActionPinDrift(repoRoot) {
  const workflowFiles = await listWorkflowFiles(repoRoot);
  const compositeFiles = await listCompositeManifests(repoRoot);
  const workflowPins = await collectPins(repoRoot, workflowFiles);
  const compositePins = await collectPins(repoRoot, compositeFiles);

  const workflowRefs = new Map();
  for (const pin of workflowPins) {
    if (!workflowRefs.has(pin.key)) workflowRefs.set(pin.key, new Set());
    workflowRefs.get(pin.key).add(pin.ref);
  }

  const findings = [];
  for (const pin of compositePins) {
    const refs = workflowRefs.get(pin.key);
    if (!refs || refs.size === 0) continue; // composite-only action: nothing to compare
    if (!refs.has(pin.ref)) {
      findings.push({ kind: 'composite-behind', file: pin.file, line: pin.line, key: pin.key, ref: pin.ref, workflowRefs: [...refs].sort() });
    }
  }
  for (const [key, refs] of [...workflowRefs.entries()].sort(([a], [b]) => a.localeCompare(b))) {
    if (refs.size > 1) findings.push({ kind: 'workflow-divergent', key, refs: [...refs].sort() });
  }

  return { findings, workflowFiles, compositeFiles, workflowPins, compositePins };
}

export function formatFinding(finding) {
  if (finding.kind === 'composite-behind') {
    return `${finding.file}:${finding.line} pins ${finding.key}@${finding.ref}, but ${WORKFLOW_DIRECTORY}/** pins ${finding.workflowRefs.join(', ')}`;
  }
  return `${WORKFLOW_DIRECTORY}/** pins ${finding.key} at more than one ref: ${finding.refs.join(', ')}`;
}

async function main() {
  const repoRoot = path.resolve(process.argv[2] ?? '.');
  const { findings, workflowFiles, compositeFiles, workflowPins, compositePins } = await findActionPinDrift(repoRoot);
  if (findings.length === 0) {
    console.log(`verify-action-pins: OK (${compositePins.length} pins in ${compositeFiles.length} composite action(s), ${workflowPins.length} pins in ${workflowFiles.length} workflow(s))`);
    return;
  }
  console.error(`verify-action-pins: ${findings.length} finding(s) — action pins must be consistent`);
  for (const finding of findings) console.error(`  ${formatFinding(finding)}`);
  console.error(`  Land the matching Dependabot PR, or align the pins so every reference to one action uses the same ref (issue #1212).`);
  process.exit(1);
}

const isMain =
  process.argv[1] &&
  pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url;
if (isMain) {
  main().catch((error) => {
    console.error('verify-action-pins: fatal', error);
    process.exit(2);
  });
}
