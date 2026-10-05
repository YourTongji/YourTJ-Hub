# YourTJ Hub 当前能力与缺口

> Doc type: status matrix
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-03

本页只汇总具体能力、缺口和维护文档。产品规则在对应规范中维护，接口字段以
[契约覆盖矩阵](../../packages/api-contract/coverage-matrix.md)为准；历史交付过程由 Git 保存。
状态词定义见[文档中心](../README.md#implementation-status-words)，代码存在不代表部署和真机验收完成。

## Supported capabilities

| 能力 | 状态 | 当前边界与维护入口 |
|---|---|---|
| 主题、回复与互动 | `Current` | 提问／瞬间／文章、图集、草稿、楼层视图、点赞收藏、关注频道、通知与私信；首页最新排序提供置顶标题区，默认单行显示置顶标签、标题与总数，可展开其余标题；见[论坛体验](forum.md)。 |
| Web 导航外壳 | `Current` | 桌面侧栏可折叠并记忆状态，窄屏使用抽屉；见[论坛阅读规范](forum.md#feed-and-reading)。 |
| Markdown 与外链 | `Partial` | 阅读支持 Mermaid、数学与受控链接预览；Vditor 替代数学分隔符预览尚缺，见[论坛体验](forum.md#links-markdown-and-stickers)。 |
| 表情库 | `Partial` | 官方目录、个人素材 API、Web token 渲染及 Flutter 管理／输入可用；Web 个人库管理未实现，见[表情规范](mobile-experience.md#sticker-library)。 |
| 内容治理与账号生命周期 | `Current` | 审核、举报、敏感词、限流、可恢复删除及账号清除；见[论坛体验](forum.md#governance-and-public-exports)与[身份规范](identity-and-access.md)。 |
| 课程目录、课评与排课 | `Current` | Web／Flutter 目录、评价、收藏、多方案排课和逐方案云同步；匿名边界、沿革及导入规则见[课程规范](courses-and-scheduling.md)。 |
| Wiki 内容投影 | `Current` | GitHub 为内容源，站内只读页面、搜索与评论，编辑／历史外链 GitHub；见[Wiki 规范](wiki-authoring.md)。 |
| 官方校园连接与数据 | `Partial` | Web／Flutter 身份连接、课表、学业与消息页面可用；体测等上游数据及真机学校认证仍有缺口，见[校园规范](campus.md)。 |
| 校园设备快照与桌面课表 | `Partial` | 白名单持久化、Android／iOS 原生 Widget 和本地时间推进已实现；OEM、省电、重启与真机矩阵仍需验收，见[校园小组件](campus.md#home-screen-widgets)。 |
| 校园地图 | `Partial` | Web 六校区／基地导览可用；张江示意图未校准，官方设施核验和 Flutter 原生页尚缺，见[地图规范](campus-map.md)。 |
| 登录、TOTP 与会话 | `Current` | 密码、GitHub／可选 Google OAuth、同济统一认证、内建 OIDC Provider、TOTP 和可撤销会话；见[身份规范](identity-and-access.md)。 |
| 外部登录 MFA 策略 | `Decision needed` | OIDC／OAuth 是否复用论坛 TOTP 仍需明确，见[身份规范](identity-and-access.md)。 |
| Agent API 与 MCP | `Partial` | Agent 生命周期、REST／MCP、持久化互动与广播、签名 Webhook、幂等回复和管理员评论策略已实现；外部运行器／真机验收及 Agent OAuth 仍未完成，见[身份规范](identity-and-access.md)与[架构](../architecture/system-overview.md)。 |
| AI 可读公开导出 | `Current` | 索引、全文和单篇 Markdown 独立开关，只导出可见内容并标记截断；见[契约与数据](../architecture/contracts-and-data.md)。 |
| 聚合搜索与索引管理 | `Partial` | 检索与管理可用；部分用户／生命周期变更在提交后异步入队，不能视为全链路事务同步。故障与重建边界见[契约与数据](../architecture/contracts-and-data.md#managed-search-reconciliation)。 |
| 移动端页面与语言 | `Current` | 首页／校园／通知／私信四个持久分支，课程／排课／Wiki 原生页面，中英日德四语言；见[移动端体验](mobile-experience.md)。 |
| 移动端正式分发 | `Current` | [官网](https://yourtj.de/#download)提供 iPhone／iPad App Store 和 Android 正式 APK 入口；公开分发与更新边界见[移动端体验](mobile-experience.md#distribution-and-updates)。 |
| 原生推送与设备验收 | `Partial` | APNs／JPush／OEM 生产投递、安装后签名升级、校园小组件及学校认证仍需相应真机证据；按[移动发布指南](../operations/mobile-releases.md)验收，已上架不代表所有设备链路完成验证。 |
| 受控 API 契约 | `Partial` | 所有非排除 `/api` 路由已纳管；OIDC 标准端点、文本导出与自动 Dart 生成不在完整生成链路内，见[契约状态](../architecture/contracts-and-data.md#contract-status)。 |
| 数据库与文件 | `Current` | 部署默认 PostgreSQL，本地默认 SQLite；SQLite BLOB／S3 文件存储，见[架构](../architecture/system-overview.md)与[对象存储](../operations/object-storage.md)。 |
| 论坛内积分 | `Partial` | 本地账本幂等入账和删除回滚可用；内存事件丢失后的持久补偿尚缺，见[积分规范](credit-and-escrow.md#current-boundary)。 |
| 跨服务积分结算 | `Planned` | credit 尚未部署或接入论坛；见[积分规范](credit-and-escrow.md)。 |
| 独立运行状态站 | `Partial` | Vue／Cloudflare 快照读取与 GitHub Actions 采集已实现；生产项目、DNS 与定时采集仍需配置验收，见[状态站规范](server-status.md)。 |

## Development and operations

- 环境、服务地址与启动命令：[本地开发](../development/local-development.md)。
- 自动化覆盖和设备验收边界：[测试指南](../development/testing.md)。
- 单二进制、开发实例与生产发布：[部署指南](../operations/deployment.md)。
- 架构选择及理由：[MADR 决策索引](../decisions/README.md)。
