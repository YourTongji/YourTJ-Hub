# Testing Strategy & Commands

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-28

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
| mobile | widget/unit | flutter test (melos analyze + test; pixel goldens are tagged `golden` and excluded from this gate; see local-development.md) |
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

The required backend, frontend, and contract workflows start for every PR so their
required status checks cannot remain pending. Each required job performs its own path
detection, so a detection failure fails that required check; its heavy Go or pnpm steps
run only when its owned inputs changed. `ci-mobile` and `ci-mobile-native` are optional workflows
with path filters; unrelated PRs do not start Flutter runners. Their path-detection jobs gate the heavier jobs.

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
- ci-contract.yml: changed contract inputs install the locked `packages/api-contract` pnpm tooling, run OpenAPI
  lint + bundle + TypeScript generation, then rejects an uncommitted diff below
  `apps/gooseforum/resource/packages/client/src/gen`. Its inputs are the contract package, generated
  TypeScript, client package manifest, and its own workflow configuration. The route-level HTTP
  contract fixture tests run inside the backend `go test ./...` gate.
- ci-mobile.yml: Flutter source, tests, assets, dependency/tooling or native files trigger independent
  analysis and test jobs. Each package tests on its own runner; `forum_app` has two disjoint Flutter
  shards. All package suites run for a shared-code change, including dependent packages. The local
  `melos run test` remains serial to avoid sharing one SDK's startup lock across Flutter processes.
  Release-tool Python tests run in a separate Ubuntu job without installing Flutter. Store metadata
  and documentation alone do not run Flutter checks.
- ci-mobile-native.yml: Android or iOS source/configuration changes build only that platform; shared
  pubspecs/lockfiles, build hooks/scripts, release tooling or the native workflow itself build both.
  Plain Dart/test/asset changes do not compile either native app. Manual dispatch builds both platforms
  when compile or Android size evidence is needed for any ref. Android retains R8/all-push-adapter
  compilation and the arm64 size artifact, with a Gradle user-home cache. iOS retains the simulator
  build and effective distribution-signing checks. Clean checkouts use `flutter pub get` before native
  compilation. Signed releases still perform full verification and both platform builds.
- Pixel golden tests are tagged `golden`, excluded from all CI behavior-test shards, and have no
  refresh workflow. For an intentional baseline update, use a matching Linux/Flutter environment
  and run `flutter test --update-goldens test/golden/pages_golden_test.dart` in `forum_app`, or
  `flutter test --update-goldens test/golden/components_golden_test.dart` in `ui_kit`.

The trade-off between fast source checks and native compile coverage is recorded in
[the mobile CI decision](../decisions/0045-mobile-ci-by-input.md).

## Documentation and governance

`ci-docs.yml` runs for every PR and every push to `dev` or `main`, so renaming or deleting a linked
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
Chromium layout and fault cases, and builds both the frontend and Netlify Functions. It does not need
the forum, database, live provider credentials or a Netlify account. For real-source local verification
and production scheduled-function checks, follow the [Netlify runbook](../operations/status-netlify.md).

## Smoke checklist

```bash
curl http://localhost:5234/            # homepage HTML (three-mode rendering, GoHTML)
curl http://localhost:5234/api/...     # JSON API (per upstream routes)
# Frontend dev: http://localhost:3010
```
