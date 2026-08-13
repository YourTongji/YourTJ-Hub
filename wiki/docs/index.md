---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "YourTJ Wiki"
  text: "同济大学校园社区平台知识库"
  tagline: VitePress · Pagefind 中文搜索 · Waline 评论接入 Hub OIDC
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/getting-started
    - theme: alt
      text: 部署指南
      link: /deployment/

features:
  - title: 内容主轨（git markdown）
    details: docs/ 目录下以 git 管理的 markdown 为唯一内容源，PR 审核后合入，构建即发布。
  - title: 飞书 CMS 辅轨
    details: 飞书文档经 scripts/sync-feishu.mjs 同步为 docs/feishu/ 下的 markdown，由 cron 触发。
  - title: 中文全文搜索
    details: vitepress-plugin-pagefind 离线全文搜索，含中文切词优化（Intl.Segmenter）。
  - title: OIDC 单点登录评论
    details: Waline 评论经 walinejs/auth OAuth Center 接入 YourTJ-Hub 内置 OIDC provider。
---

## 这是什么

YourTJ wiki 是同济大学校园社区平台 YourTJ 的公开知识库，沉淀：

- **使用指南**：平台功能、社区规范、常见问题
- **部署运维**：自托管部署、OIDC/评论/搜索等组件的配置与排障
- **开发文档**：架构说明、API 契约、贡献指南

评论系统通过 YourTJ-Hub 的 OIDC provider 登录，与论坛账号体系打通。
