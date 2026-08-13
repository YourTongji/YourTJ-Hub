# 内容规范

## 文档组织

- `docs/` 下的 markdown 是唯一内容源，跟随仓库走 PR 审核。
- 目录分类：`guide/`（指南）、`deployment/`（部署）、`feishu/`（飞书同步）。
- 新文档需在 `.vitepress/config.ts` 的 `sidebar` 登记。

## Frontmatter 约定

每个页面可写 `title` / `description`：

```md
---
title: 页面标题
description: 页面描述（用于搜索与 SEO）
---
```

## 搜索（Pagefind）

- 构建时由 `vitepress-plugin-pagefind` 生成离线索引。
- 中文切词使用浏览器 `Intl.Segmenter`（`chineseSearchOptimize`）。
- **已知限制**：安卓 WebView 对 `Intl.Segmenter` 支持不完整，中文搜索结果可能减少
  （pagefind issue #1176）。落地后需在安卓端实测，若不可用则回退 VitePress 本地搜索
  （`themeConfig.search.provider = 'local'` 已在配置中保留）。

## 评论（Waline）

- 评论组件挂在 `#doc-after` slot，按路由路径分 key。
- 未配置 `VITE_WALINE_SERVER_URL` 时不渲染评论，站点可正常发布。
- 评论登录走 YourTJ-Hub OIDC（经 walinejs/auth OAuth Center），见
  [OAuth Center 与 Hub OIDC](../deployment/oauth-center-oidc)。
