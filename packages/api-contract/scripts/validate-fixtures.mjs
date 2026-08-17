#!/usr/bin/env node
// Validate every external example fixture referenced from paths/ against the
// bundled OpenAPI response schema (issue #291: wiki-home-success.json carried
// editorId: 0 / editorName: "" which violated the WikiRecentPage editorId
// minimum:1 — the contract CI previously never validated fixtures against
// response schemas, so the drift went unnoticed).
//
// Runs after `pnpm run bundle`; consumes .tmp/openapi.bundle.yaml (gitignored),
// which inlines externalValue examples into `examples.<name>.value`.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import yaml from "js-yaml";
import Ajv2020 from "@redocly/ajv/dist/2020.js";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const bundlePath = join(root, ".tmp", "openapi.bundle.yaml");

const doc = yaml.load(readFileSync(bundlePath, "utf8"), { schema: yaml.CORE_SCHEMA });

// Resolve an internal JSON pointer like '#/components/schemas/WikiHomeResponse'.
function resolveRef(doc, ref) {
  if (!ref.startsWith("#/")) {
    throw new Error(`unsupported $ref "${ref}" (only internal pointers are expected in the bundle)`);
  }
  let node = doc;
  for (const rawPart of ref.slice(2).split("/")) {
    const part = decodeURIComponent(rawPart);
    node = node?.[part];
    if (node === undefined) {
      throw new Error(`unresolved $ref "${ref}"`);
    }
  }
  return node;
}

// Fully dereference a schema subtree so ajv can compile it standalone
// (the bundle keeps internal $refs; ajv cannot resolve them from an
// extracted schema object without the full document registered).
// Recursive schemas (e.g. WikiTreeNode.children → WikiTreeNode, PR #303)
// cannot be expanded into a finite object; fixtures are finite and shallow,
// so $ref expansion is depth-capped and anything below the cap matches the
// empty schema. Every reachable fixture level still validates against the
// real schema.
const maxDerefDepth = 50;

function deref(doc, node, depth = 0) {
  if (Array.isArray(node)) return node.map((n) => deref(doc, n, depth));
  if (!node || typeof node !== "object") return node;
  const out = {};
  for (const [key, value] of Object.entries(node)) {
    if (key === "$ref" && typeof value === "string") {
      if (depth >= maxDerefDepth) {
        // Recursion cap reached: emit the empty schema for this ref.
        continue;
      }
      const target = deref(doc, resolveRef(doc, value), depth + 1);
      for (const [tk, tv] of Object.entries(target)) {
        if (!(tk in out)) out[tk] = tv;
      }
      continue;
    }
    out[key] = deref(doc, value, depth);
  }
  return out;
}

const ajv = new Ajv2020({ strict: false, allErrors: true, logger: false });
let checked = 0;
let failed = 0;
const failures = [];

for (const [pathKey, pathItem] of Object.entries(doc.paths ?? {})) {
  for (const method of ["get", "post", "put", "patch", "delete"]) {
    const operation = pathItem?.[method];
    if (!operation) continue;
    for (const [status, response] of Object.entries(operation.responses ?? {})) {
      const content = response?.content?.["application/json"];
      const schema = content?.schema;
      if (!schema || !content?.examples) continue;
      for (const [exampleName, example] of Object.entries(content.examples)) {
        if (example.value === undefined) continue;
        checked++;
        let validate;
        try {
          validate = ajv.compile(deref(doc, schema));
        } catch (err) {
          failed++;
          failures.push(`${operation.operationId ?? `${method.toUpperCase()} ${pathKey}`} ${status} example "${exampleName}": schema compile failed: ${err.message}`);
          continue;
        }
        if (!validate(example.value)) {
          failed++;
          const detail = validate.errors
            .slice(0, 5)
            .map((e) => `${e.instancePath || "/"} ${e.message}`)
            .join("; ");
          failures.push(`${operation.operationId ?? `${method.toUpperCase()} ${pathKey}`} ${status} example "${exampleName}": ${detail}`);
        }
      }
    }
  }
}

if (failed > 0) {
  console.error(`[check-fixtures] ${failed} of ${checked} external fixtures violate their response schema:`);
  for (const f of failures) console.error(`  - ${f}`);
  console.error(`Run \`pnpm run bundle\` after editing fixtures or schemas, then re-run this check.`);
  process.exit(1);
}
console.log(`[check-fixtures] OK: ${checked} external fixtures validated against bundled response schemas.`);
