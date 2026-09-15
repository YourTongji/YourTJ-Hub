# 全局表情包库：管理员维护 + 自定义 token + 服务端展开渲染

## Status
Accepted
Class: feature

## Context and Problem Statement

论坛需要一个表情包系统。产品拍板（2026-09-15）：纯管理员维护的全局表情包库（一期不做用户自有上传/收藏）、自定义 token 语法、管理面板统一管理（单个上传 + zip 压缩包批量导入解析）、预设一批开源表情包（如 GitHub 上的 flower「花」）作为开箱内容。

实现需要回答四个设计问题：产品形态、内容标记方式、渲染归属层、预设来源与文件链路。同时必须服从既有基础设施的硬约束：单二进制 go:embed 部署（MADR 0006 形态）、图片经 `/file/img` 服务端内网代理读（MADR 0009 成本模型）、file_usage 引用门禁（删内容后文件可被 GC）、OpenAPI 合同同 PR 门禁。

## Decision Drivers

- 管理员体验：库管理必须像 badges 一样是独立面板 CRUD，不能靠手写 Markdown。
- 内容持久性：帖子里的表情不能因管理员删图/换图而破坏渲染或留下死链。
- 多端一致性：Web（SSR 渲染链）与移动端（客户端 markdown 渲染）都要有明确的表情显示路径。
- 预设合规：嵌入二进制的素材必须有明确许可证（CC BY 4.0 / Apache-2.0 / MIT 级别），真人肖像与零授权仓库排除。
- 存储治理：贴纸文件必须纳入既有 file_usage 门禁与 GC 生命周期，不允许旁路。

## Considered Options

**产品形态**
- a) 全局库（管理员维护，全员可用）（选定，一期）
- b) 用户自有上传 + 收藏（涉及审核队列、配额、肖像权风险，二期再议）

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
2. **token 与渲染**：`[:sticker:name:]` 在服务端 `RenderPostHTML` 渲染前展开为标准图片语法 `![sticker:name](/file/img/...)`，继承既有 sanitize/math/heading-id 管线。`GetPostVersion()` 7→8 触发存量帖懒重渲。展开排除区间：行内代码/代码块/链接/图片/autolink/原始 HTML/数学公式内的 token 不展开。未知或停用表情保持原始 token 文本。
3. **API 面**：admin 4 条（`GET /api/admin/stickers`、`POST sticker-save`、`POST sticker-delete`、`POST sticker-import`，SiteManager 权限，badges CRUD 同构）+ 公开只读 `GET /api/forum/stickers`（启用列表 name→url，供编辑器选择器与客户端 token 替换）。zip 导入用标准库 `archive/zip`（零新依赖），条目级失败不阻断整包，单文件校验复用 `validateUploadedImage`，zip 上限 32MB / 500 条目 / 单图 4MB。
4. **文件生命周期**：贴纸图经 `filedata.SaveFileFromUpload`（path=`stickers`）落标准存储，同时挂 `file_usage` `UsageSticker`（TargetSticker，ACTIVE）——公开读取由既有 `/file/img` 门禁放行、GC 跳过。删除贴纸必须同步删 usage 行，否则文件永不可 GC（fileController 门禁语义）。
5. **预设包**：三包 223 张 / 7.3MB go:embed 进 `app/console/stickerpresets`，`seed-stickers` 命令幂等导入（按 name 跳过已存在），经标准存储管道落库（与管理员上传完全同构，可删可改可 GC）。NOTICE.md 随包嵌入（flowerhd CC BY 4.0 署名义务）。EmojiPackage 子集仅收录非真人卡通/文字类目录；预设名经服务端 SanitizeName/UniqueName 清洗后入库。
6. **范围边界**：帖子/回复链路由服务端展开；私信气泡与移动端（帖子 markdown + 私信气泡）为 2026-09-15 追加决策——走客户端 token 展开，复用公开列表 API（web 私信安全分段渲染不引入 v-html；移动端 markdown 渲染前预展开 + 气泡 WidgetSpan 内联图；未知/停用 token 保持原文，与服务端语义对齐）。仍不做：移动端编辑器选择面板（输入入口后续按需）、SVG（imagepolicy XSS 面维持排除）、用户自有表情包/收藏。

### Consequences

- Good: 帖子与表情解耦（管理员换图不破坏历史帖）；零新第三方依赖（zip 走标准库）；存储治理零旁路；mention 管线同构复用降低维护面；预设开箱即用且合规。
- Trade-off: 单二进制 +7.3MB；GetPostVersion bump 使存量帖首读有一次懒重渲写放大；删贴纸的 usage 行联动成为评审红线（忘删即 GC 泄漏）。
- 合同义务：5 条路由同 PR 进 OpenAPI + TS 生成 + fixtures + 路由测试 + routes-snapshot + 移动端 Dart 镜像（route coverage 门禁）。
- 许可证残留风险：EmojiPackage 的 Apache-2.0 授权其仓库整理成果，个别原图出处不可考（中文表情包仓库行业现实）；已按"宽松授权仓库 + 人工剔除真人目录"控制到可接受水平，NOTICE 保留溯源。

## Pros and Cons of the Options

### 全局库（选定）vs 用户自有
- Good: 无审核/配额/肖像新风险面；管理成本集中；一期交付快。
- Bad: 用户不能带自己的表情包（二期按需扩展）。

### 自定义 token（选定）vs Markdown 直写
- Good: 内容与存储路径解耦；token 可全局统计/审计/管理；换图零破坏。
- Bad: 需要编辑器选择面板与渲染展开两处新代码（本期已付）。

### 服务端展开（选定）vs 客户端替换
- Good: 一处维护全端一致；复用 sanitize/math 管线；SSR/SEO 自然正确。
- Bad: 客户端展开（私信/移动端）需各端维护一份 token 解析（本期已付：web `sticker-token.ts` 分段 + 移动端 core 纯函数，规则对齐服务端正则）；帖子主链路仍以服务端展开为唯一权威。

## Links

- MADR 0009（图片内网代理读成本模型——贴纸读取路径）
- `docs/operations/object-storage.md`（存储运维）
- 调研来源：k4yt3x/flowerhd、getActivity/EmojiPackage、anzhi0708/WXMemeStickers（NOTICE.md 有完整署名与收录日期）
