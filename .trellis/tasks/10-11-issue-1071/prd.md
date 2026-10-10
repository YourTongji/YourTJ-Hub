# Issue #1071：私信表情预览与上传审核

状态：implementation complete（本地验证完成）
来源：https://github.com/YourTongji/YourTJ-Hub/issues/1071

## 目标

修复私信会话列表把个人表情随机 token 名显示给收件人的问题；个人上传表情必须先通过异步审核，才能被选择、解析或传播。

## 背景与已确认事实

- Issue 验收要求会话预览使用短占位文案；用户确认“先审后用”。
- 用户确认：AI 返回需人工复核，或 AI 审核关闭、shadow 模式无可用通过结论时，增加仅站点管理员可见的表情人工复核队列。
- 当前基线为 `origin/dev` 的 `c81c076dfce9e12f9266f4faf615a28e10d7d435`，独立分支 `fix/issue-1071`。
- Web `stickerPreviewLabel` 和 Flutter 同名镜像当前都会暴露 token 名：`apps/gooseforum/resource/src/site/utils/sticker-token.ts:53`、`apps/mobile/packages/core/lib/src/markdown/sticker_token.dart:181`。
- 新个人表情当前以 `IsEnabled: true` 写入；解析只查询启用表情。私有库响应包含禁用资产，便于保留并移除：`apps/gooseforum/app/service/stickerservice/library.go`、`sticker_service.go`。
- 现有异步审核复用 `task_queue`、后台 worker 和 `moderationservice.EvaluateSubmission`；当前人工内容队列只处理主题与回复修订。
- 管理端表情页与内容复核队列属于 `permission.SiteManager` 管理权限；版主入口另在论坛审核 API。表情队列只使用管理端站点管理权限，不向版主开放。
- 表情 API 由 OpenAPI 契约及手写 Dart 镜像共同维护；CI 固定 Flutter 3.44.9。存在 AVD `yourtj-course-widget-api31`，构建脚本为 `apps/mobile/scripts/build_dev_apk.sh`。

## 用户故事与验收标准

### US-1：私信收件人快速理解最后一条消息

- AC-1：会话列表预览把所有有效、未知或停用的表情 token 显示为统一且本地化的“动画表情”短文案，不显示 token 名。
- AC-2：Web 与 Flutter 对同一预览使用相同文案；普通文本及私信正文不变。
- AC-3：Web、Flutter 测试覆盖官方、个人、未知及停用 token，确认 `u_…` 随机名不泄漏到预览。

### US-2：个人上传表情先审核再使用

- AC-4：新上传资产写入 `pending` 审核状态且不可用；未通过前不进入可选表情列表、不被 token 解析或消息渲染。
- AC-5：AI 明确允许或站点管理员批准后才转为 `approved` 并启用；AI 明确拒绝或管理员拒绝后保持禁用。
- AC-6：上传者能区分待审、已拒绝和一般不可用状态；仍可移除个人库条目，且私有标签、排序、配额及历史素材保留行为不变。
- AC-7：AI 需人工复核、服务关闭、shadow 模式无可用通过结论时，资产保持待审并显示在站点管理员专属表情队列。
- AC-8：队列与审核动作只允许具有 `permission.SiteManager`（或隐含该权限的 `permission.Admin`）的人员访问；版主和普通用户不能列出或审核他人上传图片。
- AC-9：上传记录与异步任务在同一事务创建；重试、重复执行或与人工操作竞态均不能把被拒绝的资产重新启用。
- AC-10：已有个人表情不回溯审核，迁移后继续保持现有可用状态。

## 范围与影响

- Web 与 Flutter 的会话预览文案和四语言资源及测试。
- Go 表情模型、异步任务、AI 结果处理、站点管理员复核 API 和现有管理队列页面。
- 上传者私有库状态响应与 Web/Flutter 展示；OpenAPI、fixture、生成 TypeScript 类型及手写 Dart 镜像同步。
- 更新表情及审核产品/接口文档和 MADR；覆盖 SQLite 与 PostgreSQL 兼容的迁移/契约惯例。
- 获得实现批准后，使用 CI 固定 Flutter 3.44.9 编译 dev Debug APK，安装到已发现的 Android AVD 并启动。

## 不在范围

- 修改私信正文、存储 token、历史消息或全局官方表情管理。
- 回溯审核已有个人表情，新增审核供应商/配置页，或扩展论坛版主权限。
- NAS 投递、部署或修改远程 Issue 状态。提交、推送和创建 PR 已由用户后续明确授权。

## 验证计划（实现获批后）

- Go 表情服务、worker、权限/HTTP 契约和迁移测试。
- Web sticker-token 测试与类型检查；Flutter token、个人库状态和上传相关测试。
- 如修改 API 契约，运行 `make contract-check` 及相关路由契约测试。
- `git diff --check`；使用 pinned Flutter 运行 `apps/mobile/scripts/build_dev_apk.sh`，安装匹配 ABI APK 并启动现有 AVD。
