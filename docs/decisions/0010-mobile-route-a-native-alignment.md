# 移动端战略 Route A：原生 Flutter 全功能对齐

## Status
Superseded by 0011
Class: architecture

## Context and Problem Statement

`apps/mobile`（Flutter）此前处于维护模式：8 月以来提交几乎全是契约镜像修复，无功能开发；课表排课器（Web ~11k 行）、课程目录/课评、Wiki 只读均无移动端实现。

Web Push + PWA 刚合入 dev（[0008](0008-web-push-second-notification-channel.md)/issue #444），分析时 PWA-first（Route B）曾是推荐默认；用户权衡后优先原生体验与商店存在感，选 Route A（2026-09-06 用户决策）。

移动端的关键事实：`/u/:userId` 公开主页已建成；`gen/wiki.dart`（627 行）与 `gen/course_review.dart` 镜像已就绪只缺 repository + UI；iOS Info.plist 缺相册/相机权限声明；品牌资产滞留旧字标。

## Decision Drivers

- 原生体验与商店存在感优先（用户权衡）。
- 全部新功能走既有范式，不引入新框架。
- 课表算法行为必须与 Web 实现逐一对照（含边界差异）。

## Considered Options

- Route A：原生 Flutter 全功能对齐（选定）。
- Route B：PWA-first（Web Push/PWA 地基刚合入，杀手级功能零移植成本）。
- Route C：混合（Flutter 三件事 + 深链 Web）。

## Decision Outcome

1. 原生 Flutter 全功能对齐，全部新功能走既有范式（core 契约镜像 → repository → Riverpod provider → GoRoute → ConsumerStatefulWidget 页面）；不引入新状态管理/HTTP 框架，不动四 tab shell。
2. 课表纯算法一比一 Dart 移植（pkArrange/pkConflict/timetableGrid/timetable/pkCourseOrder/sectionTimes），断言值由 node 执行 Web 真实 TS 实现生成做行为对照（含 TS 正则容错差异逐一对齐）；localStorage schema 与 Web `pk.plans` 逐字段一致保证可携带性。
3. 后端新增三个公开/登录契约操作：`GET /api/pk/section-times`（pageConfig scheduleSettings 投递，此前唯一公开路径是 SSR props）、`GET /api/site-theme/tokens`（发布态主题 JSON 下发，此前仅 `/site-theme.css` CSS 通道）、`POST push/device/register|unregister` + `push_device` 表 + `nativepushservice`（APNs sideshow/apns2 token .p8 + FCM HTTP v1 OAuth，不引入 firebase-admin；复刻 webpush taskQueue outbox + 同点入队 + fail-closed 未配置即关闭，配置走 [0006](0006-deploy-config-as-artifact.md) Environments 管线）。
4. 主题同步走 GfRuntimeTheme 运行时覆盖层：内置 `GfColors`（tokens.json 镜像）保持编译期常量与 `tokens_test` 等值断言不变，服务端 tokens 逐键 hex 校验合并、非法回退内置。
5. 推送 mobile 侧 firebase_messaging + dart-define 注入 FirebaseOptions：仓库永不携带 Firebase 凭据（GoogleService plist/json 均不需要）；未配置构建 = unsupported 通道关闭，语义对齐 Web Push 门控。
6. 明确非目标：admin/审核界面、Wiki 写侧、HMS、商店上架流程、ja/de l10n、universal links、Web 端功能回改。

### Consequences

- Good: 原生推送与 Web Push（[0008](0008-web-push-second-notification-channel.md)）成双通道：同文案单源 `BuildPushCopy`、同触发点入队、同清理语义。
- Trade-off: FCM 大陆送达率差是已知接受风险，应用内 30s 轮询保持主通道语义。
- 课表数据仍是纯本地存储（无服务端同步）；跨端携带留待未来导出/导入决策。
- Web 端课表仍在演进，移植以 2026-09-06 dev（e523ab9a）为基准，后续按既有镜像同步纪律增量。
- golden 基线出自 Linux CI（`skipGoldens = !Platform.isLinux`），本地 mac 不重生成；品牌资产更新后的 golden 刷新走仓库 `mobile-golden-refresh.yml` workflow。

## Pros and Cons of the Options

### Route A 原生全功能（选定）
- Good: 原生体验、推送、离线与商店存在感；既有 Riverpod 范式零新概念。
- Bad: 双端维护成本；杀手级功能（课表）移植有对照成本。

### Route B PWA-first
- Good: 零移植成本、Web 单源。
- Bad: 用户已明确优先原生体验；iOS PWA 推送受安装约束；无商店存在。

### Route C 混合
- Good: 投入最省。
- Bad: 双会话/双 UX 割裂，仍要接原生推送通道，两头成本都付。

## Links

- 宿主 Blueprint「移动端原生对齐 Route A」（nte_074cbe2ac001obuMeMudqowD7n）
- 并行通道：[0008](0008-web-push-second-notification-channel.md)
- 配置管线：[0006](0006-deploy-config-as-artifact.md)
