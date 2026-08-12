# OAuth Center 与 Hub OIDC

Waline 评论登录链路：

```
点评论 → Waline 前端 → walinejs/auth OAuth Center（登录页）
        → 跳转 YourTJ-Hub OIDC 授权端点（authorization code + PKCE）
        → 回调 auth → userinfo → 落库 → 发表评论
```

## 1. 在 YourTJ-Hub 注册 OIDC client（必做）

编辑 Hub 的 `config.toml`（生产环境 `main/config.toml`，由运维维护），
在 `[oidc]` 区块新增 `[[oidc.clients]]`：

```toml
[oidc]
# 默认关闭！生产开启是配置变更，需运维配合并重启服务
enabled = true
# issuer 默认取 site_url + /api/oauth；如域名不同需显式配置
# issuer = "https://forum.example.com/api/oauth"
signing_key_file = "./storage/oidc/signing_key.pem"

# Waline 评论的 OAuth Center（public 客户端：不填 secret，强制 PKCE S256）
[[oidc.clients]]
id = "yourtj-wiki-comment"
name = "YourTJ Wiki 评论"
redirect_uris = ["https://auth.example.com/api/oauth/redirect"]
```

关键点：

- **`oidc.enabled` 默认关闭**。开启前确认：生产签名密钥已持久化
  （`signing_key_file` 指向的文件存在且有备份），否则重启会生成新密钥，
  已签发的授权码/token 全部失效。
- **public 客户端不填 `secret`**，Hub 侧强制 PKCE S256（与 walinejs/auth
  默认行为一致）。
- **redirect_uris 必须与 walinejs/auth 的实际回调完全一致**（大小写、
  路径都算），落地第一步先验证这条。

## 2. 部署 OAuth Center（walinejs/auth）

`deploy/wiki/oauth-center/` 目录提供自托管示例（walinejs/auth 官方无
Dockerfile，以 Vercel 部署为主；自托管需自行 wrap Koa app）。仓库内见
`deploy/wiki/oauth-center/README.md`。

核心环境变量：

| 变量 | 示例 | 说明 |
|---|---|---|
| `OIDC_ID` | `yourtj-wiki-comment` | 与 Hub `[[oidc.clients]].id` 一致 |
| `OIDC_SECRET` | （留空） | public 客户端不填（PKCE S256） |
| `OIDC_ISSUER` | `https://forum.example.com/api/oauth` | Hub OIDC issuer（自动 discovery） |

> 注意：walinejs/auth 各 provider 环境变量名以实际版本为准，部署前
> 核对 [walinejs/auth 仓库](https://github.com/walinejs/auth) 的 README。
> 另可保留 GitHub 登录（`GITHUB_ID` / `GITHUB_SECRET`）。

## 3. 部署 Waline 服务端

`deploy/wiki/waline/` 目录提供 docker-compose 示例，见
`deploy/wiki/waline/README.md`。核心配置：

- 镜像 `lizheming/waline:latest`，端口 `8360`；
- 存储 SQLite（`SQLITE_PATH=/app/data`，目录）或 MySQL（需先导入
  [waline.sql](https://github.com/walinejs/waline/blob/main/assets/waline.sql)）；
- **必填 `JWT_TOKEN`**（评论登录 token 密钥）；
- **`OAUTH_URL` 指向 OAuth Center 自身**（不是 provider）——Waline server
  用它拼接 `${oauthUrl}/oidc` 跳转登录页；
- 不配置 LeanCloud（停服风险）。

## 4. 前端接线

- 站点构建时设置 `VITE_WALINE_SERVER_URL=https://comment.example.com`，
  评论组件自动挂载（`wiki/.vitepress/theme/Layout.vue`）。
- Waline 前端登录页会展示 OAuth Center 提供的登录方式（Hub OIDC）。

## 验收清单（落地第一步）

- [ ] Hub `oidc.enabled = true` 后，`GET {issuer}/.well-known/openid-configuration` 返回正常
- [ ] auth 的 OIDC 回调 URL 与 Hub `redirect_uris` 完全一致
- [ ] 评论页点登录 → 跳 Hub 授权页 → 登录 → 回跳 → 可发评论
