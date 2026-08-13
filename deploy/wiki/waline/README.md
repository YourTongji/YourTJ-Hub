# Waline 评论服务部署

## 快速启动

```bash
cd deploy/wiki/waline
cp .env.example .env      # 按需编辑（SQLite 零配置即可跑）
docker compose up -d
```

反向代理 `https://comment.example.com` → `127.0.0.1:8360`（compose 仅回环绑定，
公网入口由宿主反代统一管理，与 forum 后端一致）。

## 存储选择

- **SQLite**（默认）：`SQLITE_PATH=/app/data`（数据目录，不是文件路径），开箱即用，适合低流量。
- **MySQL**（生产推荐）：取消 docker-compose.yml 中 `MYSQL_*` 注释并填 `.env`；
  需先导入建表 SQL（waline 仓库 `assets/waline.sql`，见 .env.example），Waline 不会自动建表。

## 登录（OIDC）

Waline 本身不存密码；登录走 `OAUTH_URL`（OAuth Center）提供的第三方登录：

```
Waline 登录页 → OAUTH_URL（walinejs/auth）→ YourTJ-Hub OIDC → 回跳
```

- `OAUTH_URL=https://auth.example.com` 指向 OAuth Center。
- OAuth Center 再对接 Hub OIDC（见 `../oauth-center/` 与
  [wiki 部署文档](../../../wiki/docs/deployment/oauth-center-oidc.md)）。

## 环境变量速查

| 变量 | 必填 | 说明 |
|---|---|---|
| `SITE_URL` | 是 | 站点地址（用于校验来源） |
| `OAUTH_URL` | 是 | OAuth Center 地址 |
| `JWT_TOKEN` | 是 | 登录 token 密钥（`openssl rand -hex 32` 生成） |
| `SQLITE_PATH` / `MYSQL_*` | 二选一 | 存储 |
| `SITE_NAME` | 否 | 站点名（评论框展示） |
| `SECURE_DOMAINS` | 否 | 允许评论的域名白名单，防跨站刷评；**必须包含 Waline 服务自身域名**（`/ui` 登录按钮的 Referer 校验），如 `wiki.example.com,comment.example.com` |
| `SMTP_*` | 否 | 评论邮件通知 |
| `AKISMET_KEY` | 否 | Akismet 反垃圾 |

完整变量清单以 [waline 官方文档](https://waline.js.org/reference/server/configuration.html) 为准。

## 管理

- 评论管理后台：`https://comment.example.com/ui`，登录方式同评论（OIDC）。
  注意 `/ui/login` 的邮箱/密码表单是 Waline **本地账号**用的（Hub 账号不在
  其密码表，用论坛密码登录必然失败），正确入口是 oidc 第三方登录按钮。
- 站点侧 `VITE_WALINE_SERVER_URL` 指向本服务后，评论即在前端渲染。
