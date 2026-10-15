import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { copyFile, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { test } from "node:test";

test("parallel route additions regenerate only the temporary report and omissions still fail", async (t) => {
  const dir = await mkdtemp(path.join(tmpdir(), "yourtj-route-coverage-"));
  t.after(() => rm(dir, { recursive: true, force: true }));
  for (const name of ["scripts", "fixtures", ".tmp"]) await mkdir(path.join(dir, name));
  await copyFile(new URL("./check-route-coverage.mjs", import.meta.url), path.join(dir, "scripts/check-route-coverage.mjs"));
  await writeFile(path.join(dir, "coverage-matrix.md"), "# Stable entry point\n");
  await writeFile(path.join(dir, "route-coverage.json"), JSON.stringify({ excluded: [], knownUncovered: [] }));
  const routes = [
    { method: "GET", path: "/api/first" },
    { method: "POST", path: "/api/second/:id" },
    { method: "GET", path: "/.well-known/example" },
  ];
  await writeFile(path.join(dir, "fixtures/routes-snapshot.json"), JSON.stringify(routes));
  const bundle = { paths: {
    "/api/first": { get: { operationId: "first" } },
    "/api/second/{id}": { post: { operationId: "second" } },
    "/.well-known/example": { get: { operationId: "wellKnown" } },
  } };
  await writeFile(path.join(dir, ".tmp/openapi.bundle.json"), JSON.stringify(bundle));
  const run = () => spawnSync(process.execPath, [path.join(dir, "scripts/check-route-coverage.mjs")], { encoding: "utf8" });
  const success = run();
  assert.equal(success.status, 0, success.stderr);
  const report = path.join(dir, ".tmp/coverage-matrix.md");
  const matrix = await readFile(report, "utf8");
  assert.match(matrix, /快照路由总数：3/);
  assert.match(matrix, /\/api JSON 路由：2，已入契约：2（100%）/);
  assert.match(matrix, /second/);
  assert.equal(await readFile(path.join(dir, "coverage-matrix.md"), "utf8"), "# Stable entry point\n");

  delete bundle.paths["/api/second/{id}"];
  await writeFile(path.join(dir, ".tmp/openapi.bundle.json"), JSON.stringify(bundle));
  const failure = run();
  assert.equal(failure.status, 1);
  assert.match(failure.stderr, /no OpenAPI operation/);
  await assert.rejects(readFile(report), { code: "ENOENT" });
});
