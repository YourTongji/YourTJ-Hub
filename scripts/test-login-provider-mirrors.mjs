import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';

const root = new URL('../', import.meta.url);
const read = (path) => readFileSync(new URL(path, root), 'utf8');
const sources = {
  go: read('apps/gooseforum/app/http/controllers/forum/payload.go'),
  web: read('apps/gooseforum/resource/packages/client/src/contracts/payload.ts'),
  dart: read('apps/mobile/packages/core/lib/src/gen/auth.dart'),
};

// /login is a page payload, not an OpenAPI operation. Keep this guard scoped to
// provider readiness; it does not claim full Dart contract generation/coverage.
function readinessFields(source, declaration, fields) {
  const body = source.match(declaration)?.[1];
  assert.ok(body, 'LoginPageProps declaration must be present');
  const entries = [...body.matchAll(fields)].map(([, name, type]) => [
    name,
    type === 'bool' ? 'boolean' : type,
  ]);
  assert.ok(entries.length > 0, 'LoginPageProps must declare provider readiness');
  return Object.fromEntries(entries.sort(([a], [b]) => a.localeCompare(b)));
}

function assertMirrors({ go, web, dart }) {
  // Reorder Go captures to share the name/type normalization below.
  const goBody = go.match(/type LoginPageProps struct \{([\s\S]*?)\n\}/)?.[1];
  assert.ok(goBody, 'Go LoginPageProps declaration must be present');
  const expected = Object.fromEntries(
    [...goBody.matchAll(/^\s*\w+\s+(\S+)\s+`json:"(\w+Ready)(?:,[^"]*)?"`/gm)]
      .map(([, type, name]) => [name, type === 'bool' ? 'boolean' : type])
      .sort(([a], [b]) => a.localeCompare(b)),
  );
  assert.ok(Object.keys(expected).length > 0, 'Go must declare provider readiness');
  assert.deepEqual(
    readinessFields(web, /interface LoginPageProps \{([\s\S]*?)\n\}/,
      /^\s*(\w+Ready)\??:\s*(\S+)/gm),
    expected,
    'Web LoginPageProps provider readiness must match Go',
  );
  // The source is hand-maintained; build_runner owns the Freezed/JSON outputs.
  const dartBody = dart.match(/factory LoginPageProps\(\{([\s\S]*?)\}\)/)?.[1];
  assert.ok(dartBody, 'Dart LoginPageProps factory must be present');
  const actual = Object.fromEntries(
    [...dartBody.matchAll(/^\s*(?:@Default\([^)]*\)\s+|required\s+)?(\w+)\s+(\w+Ready),/gm)]
      .map(([, type, name]) => [name, type === 'bool' ? 'boolean' : type])
      .sort(([a], [b]) => a.localeCompare(b)),
  );
  assert.deepEqual(actual, expected, 'Dart LoginPageProps provider readiness must match Go');
  return Object.keys(expected);
}

test('login provider readiness agrees across Go, Web and Dart', () => {
  assertMirrors(sources);
});

test('the JSON mirror reads and writes every provider readiness flag', () => {
  const generated = read('apps/mobile/packages/core/lib/src/gen/auth.g.dart');
  const login = generated.split('_$ResetPasswordPagePropsImpl')[0];
  const expected = assertMirrors(sources);
  const reads = [...login.matchAll(/^\s*(\w+Ready): json\['\1'\] as bool\??(?: \?\? false)?,/gm)]
    .map((match) => match[1]).sort();
  const writes = [...login.matchAll(/^\s*'(\w+Ready)': instance\.\1,/gm)]
    .map((match) => match[1]).sort();
  assert.deepEqual(reads, expected, 'Generated LoginPageProps JSON reader is stale');
  assert.deepEqual(writes, expected, 'Generated LoginPageProps JSON writer is stale');
});

test('the guard rejects a missing or incorrectly typed readiness mirror', () => {
  for (const language of ['web', 'dart']) {
    for (const mutate of [
      (source) => source.replace(/^.*githubReady.*\n/m, ''),
      (source) => source.replace(/^(.*githubReady.*)$/m, (line) => line.replace(/bool(?:ean)?/, 'String')),
    ]) {
      assert.throws(() => assertMirrors({ ...sources, [language]: mutate(sources[language]) }));
    }
  }
});

test('a newly published provider must be added to both mirrors', () => {
  const go = sources.go.replace('type LoginPageProps struct {',
    'type LoginPageProps struct {\n  NewReady bool `json:"newReady"`');
  assert.throws(() => assertMirrors({ ...sources, go }));
});

test('missing declarations cannot make the guard vacuously pass', () => {
  for (const language of ['go', 'web', 'dart']) {
    assert.throws(() => assertMirrors({ ...sources, [language]: '' }));
  }
});
