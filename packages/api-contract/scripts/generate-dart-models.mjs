import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import yaml from 'js-yaml';

const packageDir = fileURLToPath(new URL('..', import.meta.url));
const output = path.resolve(packageDir, '../../apps/mobile/packages/core/lib/src/gen/openapi_models.dart');
const types = { string: 'String', integer: 'int', boolean: 'bool' };
const objectKeys = new Set(['type', 'properties', 'required', 'additionalProperties', 'description', 'title']);
// These constraints stay server-validated; they do not change the wire type.
const propertyKeys = new Set(['type', 'description', 'title', 'enum', 'const', 'minimum', 'maximum', 'minLength', 'maxLength']);
const reserved = new Set(('abstract as assert async await base break case catch class const continue covariant default deferred do dynamic else enum export extends extension external factory false final finally for Function get hide if implements import in interface is late library mixin native new null of on operator part required rethrow return sealed set show static super switch sync this throw true try typedef var void when while with yield').split(' '));

function requireKeys(value, allowed, label) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error(`${label}: expected an object`);
  for (const key of Object.keys(value)) {
    if (!allowed.has(key)) throw new Error(`${label}: unsupported keyword ${key}`);
  }
}

function identifier(name, pattern, label) {
  if (typeof name !== 'string' || !pattern.test(name) || reserved.has(name)) {
    throw new Error(`${label}: unsupported Dart identifier ${name}`);
  }
}

export function renderDartModels(schemas, models) {
  if (!Array.isArray(models) || models.length === 0) throw new Error('Dart model registry is empty');
  const names = new Set();
  const blocks = models.map((model) => {
    requireKeys(model, new Set(['schema', 'className', 'fromJson', 'toJson']), 'registry entry');
    identifier(model.schema, /^[A-Z][A-Za-z0-9]*$/, 'schema');
    identifier(model.className, /^[A-Z][A-Za-z0-9]*$/, 'class');
    if (names.has(model.className)) throw new Error(`duplicate Dart class ${model.className}`);
    names.add(model.className);
    if (typeof model.fromJson !== 'boolean' || typeof model.toJson !== 'boolean' || !(model.fromJson || model.toJson)) {
      throw new Error(`${model.schema}: select fromJson and/or toJson explicitly`);
    }
    const schema = schemas[model.schema];
    requireKeys(schema, objectKeys, model.schema);
    if (schema.type !== 'object' || schema.additionalProperties !== false) {
      throw new Error(`${model.schema}: only closed object schemas are supported`);
    }
    if (!schema.properties || typeof schema.properties !== 'object' || Array.isArray(schema.properties)) {
      throw new Error(`${model.schema}: expected properties`);
    }
    const required = schema.required ?? [];
    if (!Array.isArray(required) || required.some((key) => !Object.hasOwn(schema.properties, key))) {
      throw new Error(`${model.schema}: invalid required fields`);
    }
    const fields = Object.entries(schema.properties).map(([name, property]) => {
      const label = `${model.schema}.${name}`;
      identifier(name, /^[a-z][A-Za-z0-9]*$/, label);
      if (['fromJson', 'toJson', 'hashCode', 'runtimeType', 'toString', 'noSuchMethod'].includes(name)) {
        throw new Error(`${label}: conflicts with a Dart member`);
      }
      requireKeys(property, propertyKeys, label);
      if (typeof property.type !== 'string' || !Object.hasOwn(types, property.type)) throw new Error(`${label}: unsupported type ${JSON.stringify(property.type)}`);
      const isRequired = required.includes(name);
      return { name, required: isRequired, type: types[property.type] + (isRequired ? '' : '?') };
    });
    if (!fields.length) throw new Error(`${model.schema}: empty model`);
    const args = fields.map((field) => `${field.required ? 'required ' : ''}this.${field.name}`);
    const ctor = args.length === 1
      ? `  const ${model.className}({${args[0]}});`
      : `  const ${model.className}({\n${args.map((arg) => `    ${arg},`).join('\n')}\n  });`;
    const lines = [`/// OpenAPI schema: ${model.schema}.`, `class ${model.className} {`, ctor, '',
      ...fields.map((field) => `  final ${field.type} ${field.name};`)];
    if (model.fromJson) {
      lines.push('', `  factory ${model.className}.fromJson(Map<String, dynamic> json) {`, `    return ${model.className}(`,
        ...fields.map((field) => `      ${field.name}: json['${field.name}'] as ${field.type},`), '    );', '  }');
    }
    if (model.toJson) {
      const entries = fields.map((field) => `${field.required ? '' : `if (${field.name} != null) `}'${field.name}': ${field.name}`);
      lines.push('', '  Map<String, dynamic> toJson() {');
      if (entries.length === 1 && entries[0].length < 60) {
        lines.push(`    return {${entries[0]}};`);
      } else {
        lines.push('    return {', ...entries.map((entry) => `      ${entry},`), '    };');
      }
      lines.push('  }');
    }
    lines.push('}');
    return lines.join('\n');
  });
  return '// GENERATED CODE - DO NOT MODIFY BY HAND.\n'
    + '// Source: packages/api-contract/components/schemas.yaml + dart-models.json.\n'
    + '// Regenerate: cd packages/api-contract && pnpm generate:dart\n'
    + '// Wire models only; enum/range validation remains on the server.\n\n'
    + blocks.join('\n\n') + '\n';
}

export function writeOrCheck(file, expected, check) {
  if (check) {
    if (!fs.existsSync(file) || fs.readFileSync(file, 'utf8') !== expected) {
      throw new Error('Generated Dart models are missing or stale; run pnpm generate:dart in packages/api-contract');
    }
  } else {
    fs.writeFileSync(file, expected);
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const args = process.argv.slice(2);
    if (args.length > 1 || (args.length === 1 && args[0] !== '--check')) throw new Error('Usage: generate-dart-models.mjs [--check]');
    const schemas = yaml.load(fs.readFileSync(path.join(packageDir, 'components/schemas.yaml'), 'utf8'));
    const models = JSON.parse(fs.readFileSync(path.join(packageDir, 'dart-models.json'), 'utf8'));
    writeOrCheck(output, renderDartModels(schemas, models), args.includes('--check'));
    console.log(args.includes('--check') ? 'Dart models are current' : 'Generated Dart models');
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
