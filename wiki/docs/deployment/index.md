# 部署总览

YourTJ wiki 是一个纯静态 VitePress 站点，推荐部署到 **Cloudflare Pages**。

## 架构

```
VitePress 站点 (wiki/)  ── CF Pages 部署
   ├── 内容：git markdown（docs/）
   ├── 飞书 CMS 辅轨：飞书文档 ── scripts/sync-feishu.mjs ── 提交 git ── 构建
   ├── 搜索：vitepress-plugin-pagefind（中文切词优化）
   └── 评论：@waline/client（doc-after slot，按路由分 key）
              └── Waline 服务（自托管 Docker）
                   └── OAuth Center（walinejs/auth）
                        └── OIDC ── YourTJ-Hub /api/oauth
```

## 组件

| 组件 | 说明 | 部署方式 |
|---|---|---|
| VitePress 站点 | 静态站，构建产物 `.vitepress/dist` | CF Pages |
| Pagefind 搜索 | 构建时生成离线索引，随站点发布 | 随构建 |
| Waline 评论 | 自托管评论服务（SQLite/MySQL 存储） | Docker（见 [Waline 评论服务](./waline)） |
| OAuth Center | walinejs/auth，把 Hub OIDC 转给 Waline | Docker（见 [OAuth Center 与 Hub OIDC](./oauth-center-oidc)） |

## 快速导航

- [容器部署（main/dev 接入）](./container-deploy)：接入现有 deploy 链路（nginx 容器）
- [CF Pages 发布](./cloudflare-pages)：站点构建与发布
- [Waline 评论服务](./waline)：评论服务端部署
- [OAuth Center 与 Hub OIDC](./oauth-center-oidc)：登录打通
- [飞书同步](../feishu/)：飞书文档辅轨
