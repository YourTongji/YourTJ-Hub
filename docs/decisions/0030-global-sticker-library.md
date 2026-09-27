# 全局表情包库：管理员维护 + 自定义 token + 服务端展开渲染

## Status
Superseded by [0038](0038-personal-sticker-library.md)
Class: feature

## Context and Problem Statement

论坛需要一个表情包系统。产品范围：纯管理员维护的全局表情包库（不含用户自有上传/收藏）、自定义 token 语法、管理面板统一管理（单个上传 + zip 压缩包批量导入解析）、预设一批开源表情包（如 GitHub 上的 flower「花」）作为开箱内容。

实现需要回答四个设计问题：产品形态、内容标记方式、渲染归属层、预设来源与文件链路。同时必须服从既有基础设施的硬约束：单二进制 go:embed 部署（MADR 0006 形态）、图片经 `/file/img` 服务端内网代理读（MADR 0009 成本模型）、file_usage 引用门禁（删内容后文件可被 GC）、OpenAPI 合同同 PR 门禁。

## Decision Drivers

- 管理员体验：库管理必须像 badges 一样是独立面板 CRUD，不能靠手写 Markdown。
- 内容持久性：帖子里的表情不能因管理员删图/换图而破坏渲染或留下死链。
- 多端一致性：Web（SSR 渲染链）与移动端（客户端 markdown 渲染）都要有明确的表情显示路径。
- 预设合规：嵌入二进制的素材必须有明确许可证（CC BY 4.0 / Apache-2.0 / MIT 级别），真人肖像与零授权仓库排除。
- 存储治理：贴纸文件必须纳入既有 file_usage 门禁与 GC 生命周期，不允许旁路。

## Considered Options

**产品形态**
- a) 全局库（管理员维护，全员可用）（选定）
- b) 用户自有上传 + 收藏（涉及审核队列、配额、肖像权风险，不在支持范围）

**内容标记**
- a) 自定义 token `[:sticker:name:]`，服务端渲染前展开（选定）
- b) 直接写 Markdown 图片语法 `![](url)`——编辑器零改动，但管理员换图/删图会破坏全部历史帖，且无法追溯管理
- c) 前端渲染期替换 token（三端各自实现，行为漂移）

**渲染归属**
- a) 服务端 `RenderPostHTML` 展开（mention 管线同构）（选定）
- b) 三端各自替换（web/mobile 各维护一份解析器，token 语义漂移风险）

**预设来源**
- a) flowerhd（CC BY 4.0）+ EmojiPackage 精选子集（Apache-2.0）+ WXMemeStickers（MIT）（选定）
- b) ChineseBQB（★16k 最活跃）——零授权声明 + 大量第三方 IP 内容，排除出预设；其内容可由管理员经 zip 导入通道自行扩充（自担风险）

**文件链路**
- a) 走标准 filedata 存储 + `/file/img` 代理 + file_usage `sticker` 引用（选定）
- b) 独立 embed 静态路由直读——绕开 GC 与门禁，破坏存储治理一致性

## Decision Outcome

1. **数据模型**：新表 `stickers`（`name` varchar(64) 唯一、`file_name`、`sort_order`、`is_enabled`、`created_by`），落 `app/models/forum/sticker/`，注册进 `SchemaModels()`。name 是 token 的权威查找键，字符集限 `[\p{L}\p{N}_\-]{1,64}`（空白/冒号/方括号会破坏 token 与 Markdown 解析）。
2. **token 与渲染**：`[:sticker:name:]` 在服务端 `RenderPostHTML` 渲染前展开为标准图片语法 `![sticker:name](/file/img/...)`，继承既有 sanitize/math/heading-id 管线。`GetPostVersion()` 为 8；含贴纸 token 的帖子及可见历史版本每次读取按当前定义重渲，不写回 HTML 缓存，避免新增、改名、替图、启停、删除后历史帖子仍引用旧图片。每次渲染先去重名称、一次批量查询；数据库暂时不可用时保留原文，下次读取可重试。展开排除区间：行内代码/代码块/链接/图片/autolink/原始 HTML/数学公式内的 token 不展开。未知或停用表情保持原始 token 文本。
3. **API 面**：admin 4 条（`GET /api/admin/stickers`、`POST sticker-save`、`POST sticker-delete`、`POST sticker-import`，SiteManager 权限，badges CRUD 同构）+ 公开只读 `GET /api/forum/stickers`（启用列表 name→url，供编辑器选择器与客户端 token 替换；独立 `sticker.list` 默认每 IP 每分钟 60 次）。zip 导入用标准库 `archive/zip`（零新依赖），条目级失败不阻断整包，单文件校验复用 `imagepolicy.ValidateContent`，zip 上限 32 MiB / 500 条目 / 单图 4 MiB / 总实际解压 64 MiB，超过总量立即停止。新增必须提交本人上传的图片，编辑可替图或保留现图。
4. **文件生命周期**：贴纸图经 `filedata.SaveFileFromUpload`（path=`stickers`）落标准存储，同时挂 `file_usage` `UsageSticker`（TargetSticker，ACTIVE）——公开读取由既有 `/file/img` 门禁放行、GC 跳过。定义和引用在同一主库事务提交或回滚；文件库写入失败补偿清理。删除同步释放 usage；重名用唯一约束和冲突重试，避免并发导入丢图。导入与预设共用 stickerservice，HTTP 只处理请求和响应。
5. **预设包**：三包 223 张 / 7.3MB go:embed 进 `app/console/stickerpresets`，`seed-stickers` 命令幂等导入（按 name 跳过已存在），经标准存储管道落库（与管理员上传完全同构，可删可改可 GC）。选择面板与管理页链接至公开来源和许可说明，NOTICE.md 及许可证正文随包嵌入（flowerhd CC BY 4.0 署名义务）。EmojiPackage 子集仅收录非真人卡通/文字类目录；预设名经服务端清洗后入库；任一条导入失败使命令返回非零退出状态，供部署识别失败。
6. **范围边界**：帖子/回复链路由服务端展开；私信气泡与移动端（帖子 markdown + 私信气泡）走客户端 token 展开，复用公开列表 API（web 私信安全分段渲染不引入 v-html；移动端按 Markdown AST 排除代码、链接、HTML、公式后预展开 + 气泡 WidgetSpan 内联图；未知/停用 token 保持原文，与服务端语义对齐）。仍不做：移动端编辑器选择面板（输入入口后续按需）、SVG（imagepolicy XSS 面维持排除）、用户自有表情包/收藏。

### Consequences

- Good: 帖子与表情解耦（管理员换图不破坏历史帖）；零新第三方依赖（zip 走标准库）；存储治理零旁路；mention 管线同构复用降低维护面；预设开箱即用且合规。
- Trade-off: 单二进制 +7.3MB；含贴纸帖子每次读取多一次批量查询与渲染，换取无需额外引用投影的定义一致性；客户端列表使用会话缓存，重新进入应用后刷新；文件库与主库之间使用失败补偿而非跨库事务。
- 合同义务：5 条路由同 PR 进 OpenAPI + TS 生成 + fixtures + 路由测试 + routes-snapshot + 移动端 Dart 镜像（route coverage 门禁）。
- 许可证残留风险：EmojiPackage 的 Apache-2.0 授权其仓库整理成果，个别原图出处不可考（中文表情包仓库行业现实）；已按"宽松授权仓库 + 人工剔除真人目录"控制到可接受水平，NOTICE 保留溯源。

## Pros and Cons of the Options

### 全局库（选定）vs 用户自有
- Good: 无审核/配额/肖像新风险面；管理成本集中；实现边界清晰。
- Bad: 用户不能带自己的表情包。

### 自定义 token（选定）vs Markdown 直写
- Good: 内容与存储路径解耦；token 可全局统计/审计/管理；换图零破坏。
- Bad: 需要编辑器选择面板与渲染展开两处新代码。

### 服务端展开（选定）vs 客户端替换
- Good: 一处维护全端一致；复用 sanitize/math 管线；SSR/SEO 自然正确。
- Bad: 客户端展开（私信/移动端）需各端维护一份 token 解析（当前实现：web `sticker-token.ts` 分段 + 移动端 core 纯函数，规则对齐服务端正则）；帖子主链路仍以服务端展开为唯一权威。

## Links

- [MADR 0009：对象存储内网代理读](0009-image-egress-internal-proxy.md)
- [对象存储运维](../operations/object-storage.md)
- [预设来源、署名与许可证](../../apps/gooseforum/app/console/stickerpresets/preset_stickers/NOTICE.md)
- 来源仓库：[flowerhd](https://github.com/k4yt3x/flowerhd)、[EmojiPackage](https://github.com/getActivity/EmojiPackage)、[WXMemeStickers](https://github.com/anzhi0708/WXMemeStickers)
