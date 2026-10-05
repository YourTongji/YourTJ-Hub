# Testing Strategy & Commands

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-05

## Principles

- Verification strength scales with change risk (auth/PII/governance/points/search must include
  negative, replay, privacy, failure, and reconciliation cases).
- A local subset is not CI passing; report the commands actually run and their results.
- Upstream already has solid Go unit tests (controller layer, i18n rendering, SEO meta); keep them and
  add tests when modifying.
- Bug fixes start red: write the smallest failing test that reproduces the bug, run it to confirm the
  failure, then implement the fix and turn it green. Mechanical changes (rename, formatting, dependency
  bump, docs-only) are exempt; the failing test stays as a regression test.

## Commands

```bash
# Backend
cd apps/gooseforum && go vet ./... && go test ./...

# Frontend (`pnpm check` = i18n gate: four-locale key consistency, static t() refs,
# and serverMessages → mobile server_message_catalog.dart mirror freshness)
cd apps/gooseforum/resource && pnpm typecheck && pnpm test && pnpm check && pnpm build

# Browser layout regressions (install Chromium once; Linux CI adds --with-deps)
cd apps/gooseforum/resource && pnpm exec playwright install chromium && pnpm test:browser

# PostgreSQL: use a disposable database; tests may DROP SCHEMA public CASCADE.
cd apps/gooseforum
export YOURTJ_TEST_PG_URL="host=127.0.0.1 port=5432 user=postgres password=postgres dbname=postgres sslmode=disable"
export TEST_PG_DSN="$YOURTJ_TEST_PG_URL"
go test -p 1 -parallel 1 ./app/... -run 'PostgreSQL|Postgres' -count=1 -v

# Full
make test

# Contract: lint, bundle, and regenerate the committed OpenAPI TypeScript output
cd packages/api-contract && pnpm install --frozen-lockfile && pnpm run check
# Equivalent Make targets: make contract-lint, make contract-generate-ts, make contract-check

# Build smoke
make build && ./bin/yourtj-hub serve   # then curl http://localhost:5234
```

## Layers

| Layer | Test type | Tool |
|---|---|---|
| bundles | unit tests (utilities) | go test |
| models | model/migration tests | go test |
| service | business unit + transaction cases | go test + sqlmock or testcontainers (when decided) |
| http/controllers | handler + rendering tests (upstream has some) | go test + httptest |
| resource (frontend) | typecheck + component tests | vue-tsc + Vitest |
| contract | OpenAPI lint/bundle/type generation plus real Gin route-chain fixture assertions | pnpm + go test + httptest |
| mobile | widget/unit | flutter test (melos analyze + test; behavior, layout, accessibility and token assertions) |
| mobile OIDC | controller chain unit + E2E script | `auth/test/oidc_controller_test.dart` (authorize→exchange 调用链) + `scripts/oidc_e2e.sh` (本地内建 Provider → AppAuth 模拟器回跳 → exchange 验证) |

## Test layout

测试与被测代码放在一起，按语言惯例分层；**不要为迁移而迁移**，新测试遵守以下归属：

| 层 | 位置 | 约定 |
|---|---|---|
| Go 单元测试（bundles/models/service 内部逻辑） | 与被测文件同包同目录的 `*_test.go` | Go 工具链/覆盖率/重构联动依赖同包布局；白盒测试访问包内未导出符号 |
| Go 黑盒测试（外部契约/集成行为） | 同目录 `*_test.go`，`package xxx_test` | 只通过导出 API 验证行为；需要外部文件（样例输入、golden 输出）时放同包 `testdata/` |
| 前端组件/单元测试 | `apps/gooseforum/resource/test/*.test.ts` | Vitest 独立目录，避免混入 `src/`；fixtures 就近放 `test/fixtures/` |
| Flutter 测试 | 各包 `apps/mobile/packages/<pkg>/test/` | `widget_test.dart` / `*_test.dart`，fixtures 放同目录 |
| 契约测试（路由级 HTTP 断言） | `apps/gooseforum/app/http/routes/*_test.go` | 位于 Go module 内，随 `go test ./...` 运行 |
| 契约 fixtures | `packages/api-contract/fixtures/` | 只放 JSON fixture 数据；该目录不在 Go module 内，测试代码不放这里 |
| 生成 TS 类型 | `apps/gooseforum/resource/packages/client/src/gen/`（`index.ts` / `openapi.ts`） | OpenAPI 生成并提交的输出，契约变更时随 PR 同步 |

模型/迁移测试必须同时满足 PG 门禁（见下方 CI mapping 的 `ci-backend-pg`）。

## CI mapping

All CI `push` triggers are limited to `dev` and `main`, so a push to an in-repository
PR branch is validated once by its `pull_request` run rather than again by a duplicate
`push` run. CI runs for the same PR or branch supersede older in-progress runs.

[CI / Verify](../../.github/workflows/ci.yml) is the single PR/dev/main entry. The
[domain selector](../../scripts/ci/select_inputs.py) compares complete Git changes, including both
sides of renames and deleted files. Unknown executable inputs or unavailable comparisons select
conservative validation. Domain workflows are reusable and consume an explicit source SHA.
`ci-required` always runs and rejects missing, skipped, failed or cancelled selected domains;
unselected domains carry reasons. Backend's reusable result includes PG/race and mobile includes
all test shards. The old `ci-backend`, `ci-frontend`, `ci-contract` names remain compatibility aliases
while required-check settings migrate. Docs/governance and current govulncheck run on every change.

- ci-backend.yml: changed backend or contract-fixture paths run go vet + go test + go build
  (apps/gooseforum/app/**, main.go, go.mod, go.sum, embedded resource Go/GoHTML files,
  markdown compatibility fixtures, packages/api-contract/**).
- ci-backend.yml also runs `ci-backend-pg` for model, migration, service, SQL-connection or Go module
  changes. It uses a disposable `postgres:16-alpine` service, sets both `YOURTJ_TEST_PG_URL` and
  `TEST_PG_DSN`, and runs every app package's tests matching `PostgreSQL|Postgres`. New PG tests
  must include one of those names; they need no package allowlist or `TestSchema` prefix.
  Packages and tests run serially (`-p 1 -parallel 1`) because migration suites reset the public schema.
  **Any model/migration change must pass this PG gate**: SQLite cannot validate PostgreSQL types,
  JSON comparisons, index upgrades, uniqueness or concurrency behavior. Without a test DSN these
  integration cases skip locally, so ordinary `go test ./...` is not equivalent evidence.
- ci-frontend.yml: changed frontend paths or the Go `page_component.go` registry run pnpm typecheck +
  client/site unit tests + Chromium layout tests + build. `pnpm test` includes `packages/client/test`,
  which checks the public component registry against Go, including order and exclusion of `admin.shell`.
  Browser regressions live in `resource/test/*.browser.mjs`, render the production Vue components
  and CSS through Vite, and stub API responses. These are separate from happy-dom component tests.
  The campus-map suite uses a separate temporary Vite `cacheDir` and removes it after closing its
  server, because its dependency optimizer differs from the other suites. The Node runner runs
  suites concurrently; sharing the default cache across different optimizer configurations can
  replace Vue chunks during imports and prevent another fixture from mounting.
  The campus-map location regression checks offline dictionary status, uncovered text, semester isolation, scoped time conditions and multiple-member choices with synthetic public course responses. It writes desktop/375px screenshots and a source receipt
  to `CAMPUS_MAP_BROWSER_EVIDENCE_DIR` (a temporary directory by default). CI uploads
  `campus-map-browser-evidence-<workflow SHA>` for 30 days; link the current run's artifact in a PR
  when reviewing those flows. Captures remain outside Git and are browser evidence, not dev acceptance.
- ci-contract.yml: changed contract inputs install the locked `packages/api-contract` pnpm tooling, run OpenAPI
  lint + bundle + TypeScript generation, then rejects an uncommitted diff below
  `apps/gooseforum/resource/packages/client/src/gen`. Its inputs are the contract package, generated
  TypeScript, client package manifest, and its own workflow configuration. The route-level HTTP
  contract fixture tests run inside the backend `go test ./...` gate.
- ci-mobile.yml: Flutter source, tests, assets, dependency/tooling or native files trigger independent
  analysis and test jobs. Dart paths include `integration_test`, `test_driver` and package-local `tool`
  so these sources still receive analysis; device integration tests remain a separate manual run.
  Each package tests on its own runner; `forum_app` has two disjoint Flutter
  shards. All package suites run for a shared-code change, including dependent packages. The local
  `melos run test` remains serial to avoid sharing one SDK's startup lock across Flutter processes.
  Release-tool Python tests run in `ci-automation.yml` on Ubuntu without installing Flutter. Store metadata
  and documentation alone do not run Flutter checks.
- ci-mobile-native.yml: Android or iOS source/configuration changes build only that platform; shared
  pubspecs/lockfiles, build hooks/scripts, release tooling or the native workflow itself build both.
  Plain Dart/test/asset changes do not compile either native app. The reusable workflow accepts an explicit platform list
  from the shared selector. Android retains R8/all-push-adapter
  compilation and the arm64 size artifact, with a Gradle user-home cache. iOS runs the simulator
  build, native `RunnerTests` storage-policy and widget schedule regressions, and effective distribution-signing checks.
  Clean checkouts use `flutter pub get` before native compilation. Signed releases separately verify the fixed application source
  and build only the selected native platforms.
- Mobile tests assert behavior, layout constraints, accessibility and design tokens without screenshot
  baselines. Screenshot golden tests and their PNG fixtures are not maintained or run locally, in PR CI,
  or during release verification. For visual changes, inspect the affected screens in a simulator or
  on a device in both themes; keep acceptance captures outside Git.

The trade-off between fast source checks and native compile coverage is recorded in
[the mobile verification decision](../decisions/0047-mobile-visual-acceptance.md).

Both mobile workflows pin external actions to full commit SHAs and disable checkout credential
persistence before running PR-controlled code. `node --test scripts/test-mobile-ci-*.mjs` checks
the Flutter input selection and these workflow security settings in automation CI.

## Documentation and governance

`ci-docs.yml` is called by the unified entry for every PR and every push to `dev` or `main`, so renaming or deleting a linked
source file, image or configuration also triggers the gate. It needs only Git and Node, without
application dependencies:

```bash
node --test scripts/test-doc-links.mjs
node scripts/run-gates.mjs
git diff --check
```

The link gate covers root/nested READMEs, `docs/`, fork-owned GooseForum docs and governance entry
files. It checks relative link targets and heading anchors, including Chinese and repeated headings; it does not
establish that documented behavior or external URLs are current. Review prose against the owning code.

## Independent status application

`apps/status` has an isolated pnpm workspace. Run `pnpm install --frozen-lockfile`, `pnpm test`,
`pnpm build`, and `pnpm test:browser` there. `pnpm contract:generate` regenerates its local OpenAPI
TypeScript; `ci-status.yml` rejects generated drift, runs provider/snapshot/component/contract tests,
Chromium layout and fault cases, and builds both the frontend and Cloudflare Worker. It does not need
the forum, database, live provider credentials or a Cloudflare account. For real-source local verification
and production collection checks, follow the [Cloudflare runbook](../operations/status-cloudflare.md).

## Smoke checklist

```bash
curl http://localhost:5234/            # homepage HTML (three-mode rendering, GoHTML)
curl http://localhost:5234/api/...     # JSON API (per upstream routes)
# Frontend dev: http://localhost:3010
```
