# Wiki Markdown 格式规范

> Doc type: product spec
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-15

wiki 以 GitHub 仓库 Markdown 为唯一真源（issue #256）：新仓库每页一个 `.md` 文件，文件头部可携带 YAML frontmatter 元数据。论坛侧写路径/同步引擎在渲染前剥离 frontmatter，公开接口、摘要、搜索、首图提取、TOC 均只基于剥离后的正文（issue #258）。

## 文件布局

- 每页一个 `.md` 文件，文件路径 = namespace 内相对路径（小写 slug 段，`namespace/slug[/slug...]`）。
- frontmatter 可选；存在时必须以文件首行 `---` 开始，并以独立的一行 `---` 闭合，随后空一行再写正文。
- 未闭合的 `---` 块不视为 frontmatter（按普通 Markdown 处理）。

## Frontmatter 字段

| 字段 | 必填 | 类型 | 上限 | 说明 |
|---|---|---|---|---|
| `title` | 是 | string | 512 字符 | 页面标题（与论坛 topics.title 同步，编辑器/导航/列表显示） |
| `order` | 否 | integer | — | 侧栏排序（数值越小越靠前；缺省按 path 排序） |
| `description` | 否 | string | 255 字符 | 页面摘要/描述（搜索分类与预览） |
| `tags` | 否 | list[string] | — | 搜索分类标签 |
| `draft` | 否 | boolean | — | 草稿开关：同步时 `draft: true` 的页面不写入公开投影 |

示例：

```yaml
---
title: 入门指南
order: 1
description: 新用户快速开始
tags: [入门, 教程]
draft: false
---
```

## 校验规则

- 有 frontmatter 时：`title` 必填、非空、≤512 字符；`description` ≤255 字符；`order` 必须整数；`tags` 必须字符串列表；`draft` 必须布尔值。
- YAML 语法错误 / 字段类型错误 / `title` 缺失或为空 → 创建/编辑被拒绝（接口返回 `common.request.invalidParams`）。
- 仅含 frontmatter、剥离后无正文 → 视为空内容被拒绝。
- 无 frontmatter 的普通 Markdown 照常支持（不触发校验）。
- 回滚到历史修订时采用宽松剥离：块存在即剥离，YAML 解析失败不阻断回滚（历史旧内容不阻塞管理员恢复）。

## 支持的 Markdown 子集

- GFM：段落、标题（自动生成 id）、强调、删除线、任务列表、自动链接、表格、围栏代码块。
- 图片：相对路径（`/uploads/...`、`/file/img/...`）或 http(s) 绝对地址。
- 数学：行内 `$...$` 与块级 `$$...$$`（服务端渲染与客户端预览行为一致，数学内容 HTML 转义，不执行原始 HTML）。
- 原始 HTML 不作为受信输出契约（服务端渲染过滤/转义）。

## 图片路径规则

- 优先使用相对路径，指向论坛文件服务（`/uploads/`、`/file/img/`）。
- 允许 http(s) 外链图片。
- `data:` / `blob:` URI 不参与首图提取与图库提取。

## 标题唯一性约束

- 页面唯一键是 path（`wiki_pages.path` 唯一索引；创建/移动/重命名时校验冲突与层级交叠）。
- `title` 不强制全局唯一，但同一 namespace 内应保持唯一，避免导航与搜索结果歧义。
- 页面内标题（`#`/`##`/…）由渲染器自动生成 id，重复标题自动追加后缀；TOC 与正文锚点基于该 id。

## 消费方

| 消费方 | 使用内容 |
|---|---|
| 渲染（`MarkdownToHTML`/`PostMarkdownToHTML`） | 剥离后的 body |
| 摘要（`ExtractDescription`） | 剥离后的 body |
| 首图（`ExtractFirstImageURL`） | 剥离后的 body |
| TOC（`ExtractHeadings`） | 剥离后的 body |
| 搜索（`ExtractSearchContent`） | 剥离后的 body（索引构建对 wiki 话题防御性剥离） |
| 编辑往返 | 修订表保留完整原文（含 frontmatter），编辑器加载原始 Markdown |

本规范供新 wiki 仓库 `CONTRIBUTING.md` 引用（issue #256/#257）。
