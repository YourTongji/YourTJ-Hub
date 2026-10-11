# Issue #1071 实施检查单

状态：实现完成并合并前复核通过；审查追修已纳入（引用头线标记、待审图片字节门禁、契约描述、行内意见对应的测试与文档）。

## 修改面

- [x] `apps/gooseforum/app/models/forum/sticker/`：审核状态字段、迁移兼容测试及个人资产查询/状态迁移。
- [x] `apps/gooseforum/app/service/stickerservice/`：事务内建待审资产和 task_queue 任务，扩展状态响应、解析守卫、AI worker 与人工决议；批准后同事务激活文件引用，拒绝保持字节非公开。
- [x] `apps/gooseforum/app/service/moderationservice/`、`app/console/serve.go`：表情主体标识及注册审核 worker。
- [x] `apps/gooseforum/app/http/controllers/api/`：站点管理队列和决议 API；拒绝版主读取及越权测试。
- [x] 管理端审核队列、操作日志、Web/Flutter 私人库状态和不可插入行为。
- [x] Web/Flutter 私信预览本地化占位；Flutter 回复引用头烘焙语言中立 `[Sticker]` 线标记，渲染端本地化，正文不再携带随机名。
- [x] `packages/api-contract/`：schema、路径、fixtures、生成 TypeScript 类型和 Dart 镜像。
- [x] `docs/decisions/0079-*.md`、决策索引及产品文档。

## 验收记录

- [x] Go 测试覆盖审核任务、AI 决议、人工决议、文件引用生命周期、迁移及版主越权拒绝。
- [x] Web sticker-token 遮蔽测试覆盖官方/个人/未知 token。
- [x] Flutter widget 测试覆盖待审状态展示、仅移除操作与选择器隐藏。
- [x] 契约：schema 描述、fixtures、生成 TypeScript 与手写 Dart 镜像同步。
- [x] 本机 golangci-lint 受 Go 工具链版本限制未运行，以 CI 门禁为准；PostgreSQL 迁移门禁以 CI 的 `ci-backend-pg` 为准。

## 已知边界

- 未部署共享 dev；模拟器验证使用的是旧服务端，新审核 API/行为以 CI 与契约测试为证据。
- 待审/被拒表情的图片字节不对公开读取放行（file_usage PENDING）；管理员与上传者以登录态预览。
