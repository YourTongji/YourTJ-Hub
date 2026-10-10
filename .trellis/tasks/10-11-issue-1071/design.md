# Issue #1071 设计方案

状态：implementation complete（本地验证完成）

## 约束

- 预览只改私信会话列表的表情占位，不改正文或 token。
- 新上传资产必须默认不可用，且已有共享资产生命周期、个人收藏和配额语义不变。
- 待人工处理的上传图片只能通过站点管理权限访问；不能复用面向版主的内容队列授权。
- 继续复用现有 moderationservice、`task_queue`、文件引用、API 契约和管理端 UI 模式。

## 数据与状态

在 `stickers` 增加 `review_status`（`pending`、`approved`、`rejected`），数据库默认 `approved`，因此已有记录与官方表情不变。创建新的个人上传时在同一事务中写 `pending`、`is_enabled=false`、个人库条目、文件引用和审核任务。重复保存同一上传资产不重复入队。

只有 `review_status=approved` 且 `is_enabled=true` 的资产可解析或进入可用选择器。个人库响应附带审核状态，使所有者能辨别待审、拒绝和一般禁用；仍保留移除入口。迁移不回溯审核。

## 异步审核

- 新增个人表情审核 task type，负载只包含 sticker ID；在 `console/serve.go` 注册后台 worker。
- 扩展 `AIContentInput` 的主体类型到 sticker，传入上传图像供现有 AI 图文审核器判定。
- `allow` 原子更新为 `approved` 并启用；`block` 原子更新为 `rejected` 并保持禁用；`review`、禁用、shadow 或无结果继续保持 `pending`，由人工队列处理。
- worker 错误仍按 task_queue 的既有策略重试；不能因错误自动启用。
- worker 与管理动作都只允许从 `pending` 作状态迁移，避免重试覆盖人工终态。删除或移除会员不删除历史表情素材，符合既有生命周期。

## 人工复核与权限

在现有 `/admin/review-queue` 管理页面增加表情队列视图，展示待审图片、创建者和提交时间，并提供批准/拒绝。API 挂在 `permission.SiteManager` 管理路由组中，控制器/服务仍校验权限；`permission.Admin` 按仓库既有继承规则可访问。论坛版主和普通用户无法读取该队列。

人工动作对 `pending` 状态执行条件更新；成功后记录操作者和批准/拒绝结果到现有管理操作日志。队列只返回个人表情待审项，不枚举官方库或一般用户库。

## API 与客户端

- OpenAPI 更新 `StickerItem` / 私有库契约，并同步成功 fixture、TypeScript 类型和手写 Dart 镜像。
- Web 与 Flutter 私有库可见待审核/未通过状态；未批准图片不提供可用的插入操作，个人库移除与排序行为保留。
- Web/Flutter 会话预览 helper 将识别出的 sticker token 替换为同一语义的本地化短文案（建议 key `sticker.animation`），未知 token 不暴露随机 token 字符串。

## 文档

- 新增 Proposed MADR 0079，记录个人上传表情先审后用、AI 结果映射和站点管理员人工队列。
- 实现后更新决策索引、论坛/移动端表情行为及 API 契约说明。新决策说明 0061 的主题/回复审核机制继续适用，个人表情使用独立状态和队列。

## 风险与验收

- 上传者使用自己的既有图片上传接口；本任务限制表情 token 的使用，不能把文件上传/直链能力视为私有文件传输。
- 必须验证旧行迁移后仍可用，待审资产在解析和 UI 中不可用，站点管理权限是唯一审核入口，且重试/并发不会反转终态。
- 用户已要求 dev Debug APK 安装并启动；仅在代码实施、测试和构建成功后继续执行 AVD 安装/启动。
