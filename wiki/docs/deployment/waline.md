# Waline 评论服务部署

评论服务端部署示例在 `deploy/wiki/waline/`（docker-compose + .env.example），
本站 [部署总览](./) 与 [OAuth Center 与 Hub OIDC](./oauth-center-oidc) 说明了
完整链路。本文只记录本站侧要点。

## 前端接线

`wiki/.vitepress/theme/Layout.vue` 在 `#doc-after` slot 挂载 Waline：

- 通过构建环境变量 `VITE_WALINE_SERVER_URL` 指定评论服务地址；
- **未配置时不渲染评论**（默认关闭，站点可无评论发布）；
- 评论按 `route.path` 分 key，每页独立评论线程；
- 路由切换时自动销毁并重建 Waline 实例。

```bash
# 构建时注入（CF Pages 环境变量或本地）
VITE_WALINE_SERVER_URL=https://comment.example.com pnpm build
```

## 服务端部署

```bash
cd deploy/wiki/waline
cp .env.example .env
docker compose up -d
```

- 端口 `8360`，反向代理到 `https://comment.example.com`。
- 存储：SQLite（默认）或 MySQL（生产推荐，避免 LeanCloud 停服风险）。
- 登录：`OAUTH_URL` 指向 OAuth Center（walinejs/auth），OAuth Center 再
  对接 YourTJ-Hub OIDC provider。

## 相关文档

- [OAuth Center 与 Hub OIDC](./oauth-center-oidc)：登录链路与 Hub client 注册
- [CF Pages 发布](./cloudflare-pages)：站点构建配置
