#!/usr/bin/env node
// Route coverage gate (issue #277): every Go route registered via RegisterByGin
// (snapshot: fixtures/routes-snapshot.json, produced by TestRoutesSnapshot) must
// either appear as an operation in the bundled OpenAPI contract or be listed in
// route-coverage.json as excluded / knownUncovered. Also regenerates the
// committed coverage matrix (coverage-matrix.md). Run after `pnpm run bundle:json`.
import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const contractDir = join(here, "..");

const snapshot = JSON.parse(readFileSync(join(contractDir, "fixtures", "routes-snapshot.json"), "utf8"));
const bundle = JSON.parse(readFileSync(join(contractDir, ".tmp", "openapi.bundle.json"), "utf8"));
const coverage = JSON.parse(readFileSync(join(contractDir, "route-coverage.json"), "utf8"));

const HTTP_METHODS = new Set(["get", "post", "put", "delete", "patch", "head", "options", "trace"]);

// gin uses :param / *wildcard, OpenAPI uses {param}; normalize to the OpenAPI
// form so method+path comparisons line up.
const toOpenApiPath = (p) => p.replace(/[:*]([^/]+)/g, (_, name) => `{${name}}`);
const keyOf = (method, path) => `${method} ${path}`;

// covered: OpenAPI operation key (normalized) -> { method, path, operationId }
const covered = new Map();
for (const [path, ops] of Object.entries(bundle.paths ?? {})) {
  for (const [method, op] of Object.entries(ops)) {
    if (!HTTP_METHODS.has(method)) continue;
    covered.set(keyOf(method.toUpperCase(), path), {
      method: method.toUpperCase(),
      path,
      operationId: op?.operationId ?? "(no operationId)",
    });
  }
}

const excluded = coverage.excluded ?? [];
const knownUncovered = coverage.knownUncovered ?? [];

const matchesEntry = (route, entry) =>
  (entry.method === "*" || entry.method === route.method) && entry.path === route.path;

const failures = [];

// --- classify every snapshot route ---
const coveredRows = [];
const knownUncoveredRows = [];
const excludedRows = [];
const undocumented = [];

for (const route of snapshot) {
  const opKey = keyOf(route.method, toOpenApiPath(route.path));
  if (covered.has(opKey)) {
    coveredRows.push({ route, operationId: covered.get(opKey).operationId });
    continue;
  }
  const excl = excluded.find((e) => matchesEntry(route, e));
  if (excl) {
    excludedRows.push({ route, reason: excl.reason ?? "" });
    continue;
  }
  const uncov = knownUncovered.find((e) => matchesEntry(route, e));
  if (uncov) {
    knownUncoveredRows.push({ route, reason: uncov.reason ?? "" });
    continue;
  }
  undocumented.push(route);
}

// (a) new routes with no contract operation and no coverage-list entry.
if (undocumented.length > 0) {
  failures.push(
    `[check-route-coverage] snapshot routes with no OpenAPI operation and no route-coverage.json entry:\n` +
      undocumented.map((r) => `  - ${r.method} ${r.path}`).join("\n") +
      `\nAdd the operation to packages/api-contract (paths/), or record it in route-coverage.json (excluded / knownUncovered).`
  );
}

// (b) knownUncovered entries that are now covered by the contract (stale).
const staleUncovered = knownUncovered.filter((entry) => {
  if (entry.method === "*") {
    return [...covered.keys()].some((k) => k.endsWith(` ${toOpenApiPath(entry.path)}`));
  }
  return covered.has(keyOf(entry.method, toOpenApiPath(entry.path)));
});
if (staleUncovered.length > 0) {
  failures.push(
    `[check-route-coverage] knownUncovered entries already covered by the contract; remove them from route-coverage.json:\n` +
      staleUncovered.map((e) => `  - ${e.method} ${e.path}`).join("\n")
  );
}

// (c) coverage-list entries that no longer exist in the snapshot. "*"-method
// entries are exempt: they intentionally collapse gin's Any()/StaticFS method
// expansions and are validated implicitly through classification above.
const snapshotKeys = new Set(snapshot.map((r) => keyOf(r.method, r.path)));
const dangling = [...excluded, ...knownUncovered].filter(
  (entry) => entry.method !== "*" && !snapshotKeys.has(keyOf(entry.method, entry.path))
);
if (dangling.length > 0) {
  failures.push(
    `[check-route-coverage] route-coverage.json entries not present in the route snapshot; clean them up:\n` +
      dangling.map((e) => `  - ${e.method} ${e.path}`).join("\n") +
      `\nRegenerate the snapshot with YOURTJ_UPDATE_ROUTES_SNAPSHOT=1 go test ./app/http/routes/ -run TestRoutesSnapshot if the routes genuinely changed.`
  );
}

// (d) contract operations for routes the server does not register.
const snapshotOpenApiKeys = new Set(snapshot.map((r) => keyOf(r.method, toOpenApiPath(r.path))));
const phantomOps = [...covered.values()].filter((op) => !snapshotOpenApiKeys.has(keyOf(op.method, op.path)));
if (phantomOps.length > 0) {
  failures.push(
    `[check-route-coverage] OpenAPI operations with no matching route in the snapshot (contract describes a non-existent route):\n` +
      phantomOps.map((op) => `  - ${op.method} ${op.path} (${op.operationId})`).join("\n")
  );
}

if (failures.length > 0) {
  console.error(failures.join("\n\n"));
  process.exit(1);
}

// --- regenerate the committed coverage matrix ---
const apiTotal = snapshot.filter((r) => r.path.startsWith("/api/")).length;
const apiCovered = coveredRows.length;
const lines = [];
lines.push("# Route → Contract 覆盖矩阵");
lines.push("");
lines.push("<!-- 本文件由 `pnpm run check:coverage`（scripts/check-route-coverage.mjs）生成，请勿手改。 -->");
lines.push("");
lines.push("路由快照来自 `TestRoutesSnapshot`（`fixtures/routes-snapshot.json`，默认配置装配，不含 OIDC `/api/oauth/*` 端点——OIDC 另有专项）。");
lines.push("");
lines.push(`- 快照路由总数：${snapshot.length}`);
lines.push(`- /api JSON 路由：${apiTotal}，已入契约：${apiCovered}（${Math.round((apiCovered / apiTotal) * 100)}%），已知未覆盖：${knownUncoveredRows.length}`);
lines.push(`- 非 API 排除路由：${excludedRows.length}`);
lines.push("");
lines.push(`## 已覆盖（${coveredRows.length}）`);
lines.push("");
lines.push("| Method | Path | operationId |");
lines.push("| --- | --- | --- |");
for (const row of coveredRows) {
  lines.push(`| ${row.route.method} | \`${row.route.path}\` | \`${row.operationId}\` |`);
}
lines.push("");
lines.push(`## 已知未覆盖（${knownUncoveredRows.length}）`);
lines.push("");
lines.push("| Method | Path | 归属切片 |");
lines.push("| --- | --- | --- |");
for (const row of knownUncoveredRows) {
  lines.push(`| ${row.route.method} | \`${row.route.path}\` | ${row.reason} |`);
}
lines.push("");
lines.push(`## 排除（非 JSON API，${excludedRows.length}）`);
lines.push("");
lines.push("| Method | Path | 原因 |");
lines.push("| --- | --- | --- |");
for (const row of excludedRows) {
  lines.push(`| ${row.route.method} | \`${row.route.path}\` | ${row.reason} |`);
}
lines.push("");

writeFileSync(join(contractDir, "coverage-matrix.md"), lines.join("\n"));
console.log(
  `[check-route-coverage] OK: ${snapshot.length} routes = ${coveredRows.length} covered + ` +
    `${knownUncoveredRows.length} known-uncovered + ${excludedRows.length} excluded; coverage-matrix.md regenerated.`
);
