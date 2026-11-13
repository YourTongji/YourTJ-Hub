import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { renderDartModels, writeOrCheck } from './generate-dart-models.mjs';

const registry = [{ schema: 'Example', className: 'ExampleInput', fromJson: true, toJson: true }];
const schema = { type: 'object', additionalProperties: false, required: ['token'], properties: {
  token: { type: 'string' }, provider: { type: 'string' },
} };
const render = (input) => renderDartModels({ Example: input }, registry);

test('schema additions and requiredness change the generated public model', () => {
  const output = render(schema);
  assert.match(output, /required this.token/);
  assert.match(output, /final String\? provider/);
  assert.match(output, /if \(provider != null\) 'provider': provider/);
  assert.match(output, /token: json\['token'\] as String/);
  const changed = structuredClone(schema);
  changed.properties.count = { type: 'integer' };
  changed.required.push('count', 'provider');
  const next = render(changed);
  assert.match(next, /required this.count/);
  assert.match(next, /final int count/);
  assert.match(next, /final String provider/);
  assert.doesNotMatch(next, /if \(provider != null\)/);
});

test('unsupported schema semantics fail instead of guessing a Dart representation', () => {
  for (const property of [{ type: 'array', items: { type: 'string' } }, { $ref: '#/Example' },
    { type: ['string', 'null'] }, { type: 'string', default: 'auto' }, { type: 'string', format: 'date-time' },
    { type: 'number' }, { type: 'string', readOnly: true }]) {
    assert.throws(() => render({ ...schema, properties: { token: property } }), /unsupported/);
  }
  assert.throws(() => render({ ...schema, allOf: [] }), /unsupported keyword allOf/);
  assert.throws(() => render({ ...schema, additionalProperties: true }), /closed object/);
  assert.throws(() => render({ ...schema, required: ['missing'] }), /invalid required/);
  assert.throws(() => render({ ...schema, properties: { token: { type: 'string' }, class: { type: 'string' } } }), /identifier/);
  assert.throws(() => renderDartModels({ Example: schema }, [...registry, ...registry]), /duplicate Dart class/);
});

test('freshness check rejects missing or stale files without rewriting them', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'yourtj-dart-models-'));
  try {
    const file = path.join(dir, 'models.dart');
    const expected = render(schema);
    assert.throws(() => writeOrCheck(file, expected, true), /missing or stale/);
    assert.equal(fs.existsSync(file), false);
    writeOrCheck(file, expected, false);
    writeOrCheck(file, expected, true);
    fs.appendFileSync(file, '// manual edit\n');
    assert.throws(() => writeOrCheck(file, expected, true), /missing or stale/);
    assert.equal(fs.readFileSync(file, 'utf8'), expected + '// manual edit\n');
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
});
