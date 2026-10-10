# Issue #1071 实施检查单

状态：实现及本地验证完成（未提交）。

## 预计修改面

- [x] `apps/gooseforum/app/models/forum/sticker/`：审核状态字段、迁移兼容测试及个人资产查询/状态迁移。
- [x] `apps/gooseforum/app/service/stickerservice/`：事务内建待审资产和 task_queue 任务，扩展状态响应、解析守卫、AI worker 与人工决议。
- [x] `apps/gooseforum/app/service/moderationservice/`、`app/console/serve.go`：表情主体标识及注册审核 worker。
- [x] `apps/gooseforum/app/http/controllers/api/`：站点管理队列和决议 API；拒绝版主读取及越权测试。
- [x] 管理端审核队列、操作日志、Web/Flutter 私人库状态和不可插入行为。
- [x] Web/Flutter 私信预览本地化占位和回归测试。
- [x] `packages/api-contract/`：schema、路径、fixtures、生成 TypeScript 类型和 Dart 镜像。
- [x] `docs/decisions/0079-*.md`、决策索引及产品文档。

## 验收步骤

- [x] Go 测试覆盖审核任务、AI 决议、人工决议、迁移及版主越权拒绝。
- [x] Web 全量测试通过：185 个测试文件、1407 项测试；`pnpm typecheck` 和 `pnpm run check` 通过。
- [x] Flutter 3.44.9 下 `melos run analyze`、core 全量测试及 forum_app 全量测试通过。
- [x] `node scripts/verify-decisions.mjs` 与 `git diff --check` 通过。
- [x] `pnpm run check` 的契约检查通过（fixtures、TS 生成和路由覆盖）。根 `make contract-check` 的生成文件洁净度守卫因本次预期的 `openapi.ts` 差异退出；生成文件保留为未提交差异。
- [x] `apps/mobile/scripts/build_dev_apk.sh` 生成 armeabi-v7a、arm64-v8a、x86_64 Debug APK；x86_64 包已安装并在 `yourtj-course-widget-api31` 前台启动。

## 未执行

- 未部署 Go 后端；APK 使用仓库脚本配置的共享 dev 服务端，因此服务端尚未包含的审核 API/行为不能通过本次模拟器启动验证。
- 未提交、推送、创建 PR 或投递 NAS。
